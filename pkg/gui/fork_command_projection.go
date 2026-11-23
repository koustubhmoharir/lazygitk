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
		variants := forkCommandVariants[original.CommandID]
		if len(variants) == 0 {
			copy := *original
			if original.CommandIDFunc != nil {
				copy.CommandID = original.GetCommandID()
				copy.CommandIDFunc = nil
			}
			if err := gui.projectForkBinding(&copy, original.CommandID); err != nil {
				return nil, err
			}
			output = append(output, &copy)
			// Fork: an explicit legacy key of a listing command is its own explicit route,
			// kept apart from the default listing routes (see forkListingLegacyRoutes).
			if legacy := gui.forkListingLegacyRoutes(&copy); len(legacy) > 0 {
				explicitCopy := copy
				explicitCopy.SetCommandPaths(legacy)
				explicitCopy.DefaultCommand = false
				output = append(output, &explicitCopy)
			}
			continue
		}
		for _, variant := range variants {
			copy := *original
			copy.CommandID, copy.Verb = variant.id, variant.verb
			copy.CommandIDFunc, copy.VerbFunc = nil, nil
			copy.CommandActive = func() bool { return original.GetCommandID() == variant.id }
			gui.labelForkCommandVariant(&copy)
			copy.GetDisabledReason = func() *types.DisabledReason {
				if !copy.CommandActive() {
					return &types.DisabledReason{Text: gui.Tr.NoApplicableCommandsInThisContext, Inapplicable: true}
				}
				if original.GetDisabledReason != nil {
					return original.GetDisabledReason()
				}
				return nil
			}
			if err := gui.projectForkBinding(&copy, original.CommandID); err != nil {
				return nil, err
			}
			output = append(output, &copy)
		}
	}
	return output, nil
}

func (gui *Gui) projectForkBinding(binding *types.Binding, sourceID string) error {
	if binding.CommandID == "commit.search" {
		if _, configured := gui.c.UserConfig().CommandBindings[binding.CommandID]; !configured {
			if _, shared := gui.c.UserConfig().CommandBindings["search.open"]; shared {
				sourceID = "search.open"
			}
		}
	}
	if binding.CommandID == "file.refresh" {
		_, configured, _, err := keymap.ResolvePaths(gui.c.UserConfig(), binding.CommandID, sourceID, binding.ViewName)
		if err != nil {
			return err
		}
		if !configured {
			// Fork: the default Files-only action is an alias of full refresh.
			// Keep its native identity/handler only for explicit legacy overrides.
			binding.CommandID, sourceID = "app.refresh", "app.refresh"
		}
	}
	paths, explicit, known, err := keymap.ResolvePaths(gui.c.UserConfig(), binding.CommandID, sourceID, binding.ViewName)
	if err != nil {
		return err
	}
	if !known {
		paths = forkNavigationPaths(binding.CommandID)
	}
	if forkListingCommand(binding.CommandID) && !gui.forkCommandOverridden(binding.CommandID) {
		// Fork: the listing routes are the fork's default routes, so an explicit
		// legacy goInto does not make them explicit (its own route is added by
		// projectForkBindings). Explicit claims of their keys by other commands win
		// through the usual explicit-leaf rule of the prefix menu.
		paths, explicit = forkListingDefaultRoutes(binding.CommandID, binding.ViewName), false
	}
	if binding.CommandID == "commit.search" {
		// Fork: the history-loading search replaces the generic search handler
		// in Commits. Give native same-key aliases one semantic identity so a
		// single search key does not open a choice between duplicate commands.
		binding.CommandID = "search.open"
	}
	binding.DefaultCommand = !explicit
	if binding.CommandID == "app.refresh" {
		binding.Handler = gui.refreshAllForkState
		binding.Description, binding.Tooltip = gui.Tr.Refresh, gui.Tr.RefreshTooltip
		binding.ShortDescription = ""
		binding.DescriptionFunc, binding.ShortDescriptionFunc, binding.TooltipFunc = nil, nil, nil
	}
	if binding.Tag == "global" && binding.ViewName != "" {
		// A view-owned handler must not appear as an executable global action
		// while the palette is focused on a different pane.
		binding.Tag = ""
	}
	if !explicit {
		if forkReplacedCommand(binding.CommandID) {
			// Fork: combine and the selection-based c p replace these upstream
			// palette actions. Explicit legacy shortcuts still restore them.
			binding.Description, binding.ShortDescription, binding.Tooltip = "", "", ""
			binding.DescriptionFunc, binding.ShortDescriptionFunc, binding.TooltipFunc = nil, nil, nil
			binding.DisplayOnScreen = false
		}
		if handler := gui.forkSelectedCheckoutHandler(binding.CommandID, binding.ViewName); handler != nil {
			binding.Handler = handler
			// Only the name prompts of commits and tags open a popup; direct
			// checkouts of a branch or remote branch run without one.
			binding.OpensMenu = binding.CommandID == "tag.checkout" || binding.CommandID == "commit.checkout-menu"
		}
		if binding.CommandID == "repo.switch-menu" && binding.ViewName == "" {
			// Fork: in Status, s is for repositories. The global recent-repos
			// binding adds no route; Status's own binding keeps s r.
			paths = nil
		}
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
				paths = forkWithoutInspectionKeys(paths)
			} else if binding.ViewName == "files" {
				// 0 belongs to Files' focus-only navigation.focus-main route.
				paths = [][]gocui.Key{{gocui.NewKeyRune('v')}}
			} else if binding.ViewName == "commits" {
				// Fork: Space toggles the commit selection in Commits (selection.commit.toggle);
				// v still inspects there.
				paths = forkWithoutSpaceKey(paths)
			}
		}
		if strings.HasPrefix(binding.CommandID, "navigation.focus-panel.") {
			// Each numbered jump has the same verb/ID but a different target.
			// Preserve its constructor's individual configured number keys.
			return nil
		}
	}
	binding.SetCommandPaths(paths)
	return nil
}

func forkReplacedCommand(id string) bool {
	switch id {
	case "commit.squash-down", "commit.mark-fixup", "commit.set-fixup-message", "commit.create-fixup-menu", "commit.autosquash", "commit.find-fixup-target":
		return true
	}
	return false
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
	case "file.enter", "remote.view-branches", "navigation.open-commits", "navigation.open-diff-files", "navigation.open-history-file-diff", "navigation.focus-diff", "submodule.enter":
		return true
	}
	return false
}

// forkWithoutInspectionKeys drops the shared v/Space inspection routes from a
// pane's non-preferred inspector, keeping its own keys such as focus-diff's 0.
func forkWithoutInspectionKeys(paths [][]gocui.Key) [][]gocui.Key {
	var kept [][]gocui.Key
	for _, path := range paths {
		if len(path) == 1 && (path[0] == gocui.NewKeyRune('v') || path[0] == gocui.NewKeyRune(' ')) {
			continue
		}
		kept = append(kept, path)
	}
	return kept
}

// forkWithoutSpaceKey drops the Space route from a pane's inspection route, keeping its v.
func forkWithoutSpaceKey(paths [][]gocui.Key) [][]gocui.Key {
	var kept [][]gocui.Key
	for _, path := range paths {
		if len(path) == 1 && path[0] == gocui.NewKeyRune(' ') {
			continue
		}
		kept = append(kept, path)
	}
	return kept
}

// forkListingCommand is the changed-files and full-listing pair (item 1 of
// "browse a commit's files"), which Commits, Sub-commits and Reflog reach by v and
// Space as a menu, and Stash reaches by a direct v and Space.
func forkListingCommand(id string) bool {
	return id == "navigation.open-diff-files" || id == "navigation.open-tree-files"
}

// forkCommandOverridden reports whether CommandBindings sets the keys of a command
// itself, which replaces every route the fork gives it.
func (gui *Gui) forkCommandOverridden(id string) bool {
	_, overridden := gui.c.UserConfig().CommandBindings[id]
	return overridden
}

// forkListingDefaultRoutes returns the fork's default routes of a listing command in a
// pane: the menu routes, or Stash's direct v and Space.
func forkListingDefaultRoutes(id, view string) [][]gocui.Key {
	defaults, _ := keymap.DefaultPaths(id)
	return forkListingPaths(id, view, defaults)
}

// forkListingLegacyRoutes returns the explicitly configured legacy key (goInto) of
// a listing command, as its own route. Upstream bound goInto to the changed-files
// listing only in Commits, Sub-commits and Stash; in Reflog it opens Sub-commits, so
// no goInto route is given there. Returns nil when the key is not explicit.
func (gui *Gui) forkListingLegacyRoutes(binding *types.Binding) [][]gocui.Key {
	id := binding.CommandID
	if id != "navigation.open-diff-files" || gui.forkCommandOverridden(id) {
		return nil
	}
	switch binding.ViewName {
	case "commits", "subCommits", "stash":
	default:
		return nil
	}
	paths, explicit, known, err := keymap.ResolvePaths(gui.c.UserConfig(), id, id, binding.ViewName)
	if err != nil || !known || !explicit {
		return nil
	}
	return paths
}

// forkListingPaths chooses the routes of a listing command for a pane. Commits
// has v as the menu prefix and keeps Space for toggling the selection. Sub-commits
// and Reflog open the same menu on both v and Space. Stash keeps v and Space as a
// direct leaf for its changed files, with no menu, and the full listing is not
// offered there. The bare v and Space paths would be a leaf beside the menu's
// children, which the command tree does not allow, so the menu panes drop them.
func forkListingPaths(id, view string, paths [][]gocui.Key) [][]gocui.Key {
	var kept [][]gocui.Key
	for _, path := range paths {
		switch view {
		case "commits", "subCommits", "reflogCommits":
			if len(path) != 2 {
				continue
			}
			if path[0] == gocui.NewKeyRune('v') || (view != "commits" && path[0] == gocui.NewKeyRune(' ')) {
				kept = append(kept, path)
			}
		case "stash":
			if id == "navigation.open-diff-files" && len(path) == 1 {
				kept = append(kept, path)
			}
		}
	}
	return kept
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
	case "submodules":
		return "submodule.enter"
	case "worktrees":
		return "navigation.open-commits"
	}
	// Unlike file.enter, this never toggles a directory: it focuses the
	// read-only details already rendered there. Files keeps this route for
	// its non-submodule entries; FilesController.InspectSelected enters a
	// submodule entry itself.
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
		labels = "<shift+up> K"
	case "selection.list.extend-down", "selection.diff.extend-down":
		labels = "<shift+down> J"
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
