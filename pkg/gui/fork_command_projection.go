package gui

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type forkCommandVariant struct{ id, verb string }

// These upstream bindings share a physical key but perform different verbs.
// Separate routes keep their applicability live as the selection/mode changes.
var forkCommandVariants = map[string][]forkCommandVariant{
	"file.toggle-stage": {
		{"file.toggle-stage", "toggle-stage"}, {"conflict.options-menu", "conflict-options"},
	},
	"diff.primary-selection": {
		{"diff.stage-selection", "stage"},
		{"diff.unstage-selection", "unstage"},
		{"patch.toggle-selection", "toggle-patch-selection"},
		{"patch.remove-selection", "remove-from-patch"},
	},
	"diff.discard-selection": {
		{"diff.discard-worktree-selection", "discard"}, {"commit.discard-selection", "discard-from-commit"}, {"diff.unstage-selection", "unstage"},
	},
}

func (gui *Gui) projectForkBindings(bindings []*types.Binding) ([]*types.Binding, error) {
	output := make([]*types.Binding, 0, len(bindings))
	for _, original := range bindings {
		if !forkCommandView(original.ViewName) {
			output = append(output, original)
			continue
		}
		// Mouse registrations retain native identity and do not become keyboard
		// aliases merely because they share an action ID with a keyboard route.
		var mouseKeys []gocui.Key
		for _, key := range original.Keys {
			if forkMouseKey(key) {
				mouseKeys = append(mouseKeys, key)
			}
		}
		if len(mouseKeys) > 0 {
			mouse := *original
			mouse.Keys = mouseKeys
			output = append(output, &mouse)
			if len(original.Keys) == len(mouseKeys) {
				continue
			}
		}
		_, legacyExplicit, _, err := keymap.ResolvePaths(gui.c.UserConfig(), original.CommandID, original.CommandID, original.ViewName)
		if err != nil {
			return nil, err
		}
		if _, commandOverride := gui.c.UserConfig().CommandBindings[original.CommandID]; commandOverride {
			legacyExplicit = false
		}
		variants := forkCommandVariants[original.CommandID]
		if len(variants) == 0 || legacyExplicit {
			copy := *original
			if original.CommandIDFunc != nil {
				copy.CommandID = original.GetCommandID()
				copy.CommandIDFunc = nil
			}
			if err := gui.projectForkBinding(&copy, original.CommandID, legacyExplicit); err != nil {
				return nil, err
			}
			output = append(output, &copy)
			continue
		}
		for _, variant := range variants {
			copy := *original
			copy.CommandID, copy.Verb = variant.id, variant.verb
			copy.CommandIDFunc, copy.VerbFunc = nil, nil
			copy.GetDisabledReason = func() *types.DisabledReason {
				if original.GetCommandID() != variant.id {
					return &types.DisabledReason{Text: gui.Tr.NoApplicableCommandsInThisContext, Inapplicable: true}
				}
				if original.GetDisabledReason != nil {
					return original.GetDisabledReason()
				}
				return nil
			}
			if err := gui.projectForkBinding(&copy, original.CommandID, false); err != nil {
				return nil, err
			}
			output = append(output, &copy)
		}
	}
	return output, nil
}

func (gui *Gui) projectForkBinding(binding *types.Binding, sourceID string, legacyExplicit bool) error {
	paths, explicit, known, err := keymap.ResolvePaths(gui.c.UserConfig(), binding.CommandID, sourceID, binding.ViewName)
	if err != nil {
		return err
	}
	if !known {
		paths = forkNavigationPaths(binding.CommandID)
	}
	binding.DefaultCommand = !explicit
	if binding.Tag == "global" && binding.ViewName != "" {
		// A view-owned handler must not appear as an executable global action
		// while the palette is focused on a different pane.
		binding.Tag = ""
	}
	if !explicit {
		// User-approved exception: x in the staged diff keeps upstream's
		// immediate unstage operation, alongside the explicit unstage path.
		if sourceID == "diff.discard-selection" && binding.CommandID == "diff.unstage-selection" {
			paths = [][]gocui.Key{{gocui.NewKeyRune('x')}}
		}
		if binding.CommandID == "commit.create" && binding.ViewName != "files" {
			paths = nil
		}
		if forkInspectionCommand(binding.CommandID) {
			if binding.CommandID != forkPreferredInspector(binding.ViewName) {
				paths = nil
			} else if binding.ViewName == "files" {
				paths = [][]gocui.Key{{gocui.NewKeyRune('v')}}
			}
		}
		if binding.CommandID == "file.refresh" {
			// Fork F5 replaces the upstream Files-only refresh with a full
			// repository/model refresh, matching the global refresh action.
			binding.Handler = func() error { gui.c.Refresh(types.RefreshOptions{}); return nil }
		}
		if binding.CommandID == "navigation.focus-panel" {
			// Each numbered jump has the same verb/ID but a different target.
			// Preserve its constructor's individual configured number keys.
			return nil
		}
	}
	// Explicit legacy mode keys intentionally retain upstream mode-dependent
	// dispatch; explicit per-command paths above remain separate action routes.
	if legacyExplicit {
		binding.DefaultCommand = false
	}
	binding.SetCommandPaths(paths)
	return nil
}

func forkCommandView(view string) bool {
	switch view {
	case "", "status", "files", "localBranches", "remotes", "remoteBranches", "tags", "commits", "subCommits", "reflogCommits", "commitFiles", "stash", "worktrees", "submodules", "main", "secondary", "mergeConflicts", "extras":
		return true
	}
	return false
}

func forkMouseKey(key gocui.Key) bool {
	return gocui.IsMouseKey(key)
}

func forkInspectionCommand(id string) bool {
	switch id {
	case "file.enter", "remote.view-branches", "navigation.open-commits", "navigation.open-diff-files", "navigation.open-history-file-diff", "navigation.focus-diff":
		return true
	}
	return false
}

func forkPreferredInspector(view string) string {
	switch view {
	case "localBranches", "remoteBranches", "tags":
		return "navigation.open-commits"
	case "commits", "subCommits", "reflogCommits", "stash":
		return "navigation.open-diff-files"
	case "remotes":
		return "remote.view-branches"
	case "commitFiles":
		return "navigation.open-history-file-diff"
	}
	// Unlike file.enter, this never enters a submodule repository or toggles
	// a directory: it focuses the read-only details already rendered there.
	return "navigation.focus-diff"
}

func forkNavigationPaths(id string) [][]gocui.Key {
	var labels string
	switch id {
	case "navigation.list.previous", "navigation.diff.previous-line", "navigation.conflict.hunk.previous", "navigation.command-log.scroll-up":
		labels = "k <up>"
	case "navigation.list.next", "navigation.diff.next-line", "navigation.conflict.hunk.next", "navigation.command-log.scroll-down":
		labels = "j <down>"
	case "navigation.page.previous", "navigation.command-log.page-up":
		labels = "<pgup>"
	case "navigation.page.next", "navigation.command-log.page-down":
		labels = "<pgdown>"
	case "navigation.list.first", "navigation.diff.first", "navigation.command-log.first":
		labels = "<home>"
	case "navigation.list.last", "navigation.diff.last", "navigation.command-log.last":
		labels = "<end>"
	case "selection.list.extend-up", "selection.diff.extend-up":
		labels = "<shift+up>"
	case "selection.list.extend-down", "selection.diff.extend-down":
		labels = "<shift+down>"
	case "navigation.panel.previous":
		labels = "h"
	case "navigation.panel.next":
		labels = "l"
	case "navigation.tab.previous":
		labels = "<backtab>"
	case "navigation.tab.next":
		labels = "<tab>"
	case "navigation.scroll.left":
		labels = "<alt+left>"
	case "navigation.scroll.right":
		labels = "<alt+right>"
	case "navigation.main.scroll-up":
		labels = "<alt+up> <pgup>"
	case "navigation.main.scroll-down":
		labels = "<alt+down> <pgdown>"
	case "navigation.tree.collapse":
		labels = "<left>"
	case "navigation.tree.expand":
		labels = "<right>"
	case "app.cancel", "navigation.return-files", "navigation.diff.escape", "commit.cancel-drag":
		labels = "<esc>"
	}
	labelList := strings.Fields(labels)
	paths := make([][]gocui.Key, 0, len(labelList))
	for _, label := range labelList {
		key, ok := config.KeyFromLabel(label)
		if !ok {
			panic("invalid fork navigation key: " + label)
		}
		paths = append(paths, []gocui.Key{key})
	}
	return paths
}
