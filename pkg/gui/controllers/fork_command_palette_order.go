package controllers

import (
	"fmt"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

// Fork: the `?` palette lists every command in sections: the focused pane with
// the global commands, then one section per other pane, then the focused pane's
// navigation. Within a section, commands are ordered by key group (see
// sortPaletteBindingsByKeyGroup). Command usage counts do not order the palette.

// forkPalettePaneOrder is the layout order of the panes in the palette. Views
// with bindings that are not listed here follow it, in name order, so that no
// command is dropped when a view is added.
var forkPalettePaneOrder = []string{
	"status", "files", "worktrees", "submodules",
	"localBranches", "remotes", "remoteBranches", "tags",
	"commits", "reflogCommits", "subCommits", "commitFiles", "stash",
	// The main area: staging and patch building are commands of the main view.
	"main", "secondary", "mergeConflicts",
	"extras", "information",
}

// forkPalettePopupViews are the popups. Their commands are listed only while the
// popup is focused, in the first section. From another pane they are noise: a
// popup is open over that pane, and its commands are not that pane's commands.
var forkPalettePopupViews = map[string]bool{
	"commitMessage":     true,
	"commitDescription": true,
	"confirmation":      true,
	"prompt":            true,
	"menu":              true,
	"search":            true,
	"suggestions":       true,
}

// orderPaletteViews returns views in palette order: those in forkPalettePaneOrder
// first, then the others sorted by name.
func orderPaletteViews(views []string) []string {
	rank := func(view string) int {
		if i := slices.Index(forkPalettePaneOrder, view); i >= 0 {
			return i
		}
		return len(forkPalettePaneOrder)
	}
	ordered := slices.Clone(views)
	slices.SortStableFunc(ordered, func(a, b string) int {
		if ra, rb := rank(a), rank(b); ra != rb {
			return ra - rb
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})
	return ordered
}

// paletteViewTitle returns the name a pane shows to the user: its tab title for
// side panels, and its frame title otherwise. The main views are named as the
// cheatsheet names them, because their frame title is always "Diff".
func paletteViewTitle(view string, tr *i18n.TranslationSet) string {
	switch view {
	case "":
		return tr.GlobalTitle
	case "status":
		return tr.StatusTitle
	case "files":
		return tr.FilesTitle
	case "worktrees":
		return tr.WorktreesTitle
	case "submodules":
		return tr.SubmodulesTitle
	case "localBranches":
		return tr.LocalBranchesTitle
	case "remotes":
		return tr.RemotesTitle
	case "remoteBranches":
		return tr.RemoteBranchesTitle
	case "tags":
		return tr.TagsTitle
	case "commits":
		return tr.CommitsTitle
	case "reflogCommits":
		return tr.ReflogCommitsTitle
	case "subCommits":
		return tr.SubCommitsTitle
	case "commitFiles":
		return tr.CommitFilesTitle
	case "stash":
		return tr.StashTitle
	case "main":
		return tr.NormalTitle
	case "secondary":
		return tr.SecondaryTitle
	case "mergeConflicts":
		return tr.MergingTitle
	case "commitMessage":
		return tr.CommitSummaryTitle
	case "commitDescription":
		return tr.CommitDescriptionTitle
	case "extras":
		return tr.ExtrasTitle
	case "information":
		return tr.InformationTitle
	case "confirmation":
		return tr.ConfirmationTitle
	case "prompt":
		return tr.PromptTitle
	case "menu":
		return tr.MenuTitle
	case "search":
		return tr.SearchTitle
	case "suggestions":
		return tr.SuggestionsCheatsheetTitle
	}
	return view
}

// paletteFocusedTitle titles the first section: the focused pane and the globals.
func paletteFocusedTitle(view string, tr *i18n.TranslationSet) string {
	if view == "" {
		return tr.GlobalTitle
	}
	return fmt.Sprintf(tr.CommandPaletteFocusedSectionTitle, paletteViewTitle(view, tr))
}

// paletteKeyGroups are the first keys of multi-key paths that get their own
// group, in display order. Any other first key forms a group after them,
// sorted by key.
var paletteKeyGroups = []string{"d", "s", "n", "c", "u", "y", "g", "t", "w"}

// Group numbers: 0 for single keys, 1 to 9 for paletteKeyGroups, then other
// first keys, then commands without a key path.
const (
	paletteGroupSingleKey = 0
	paletteGroupOtherKey  = 10
	paletteGroupNoPath    = 11
)

// paletteKeyGroup returns the group of a key path given as key labels.
func paletteKeyGroup(path []string) int {
	switch {
	case len(path) == 0:
		return paletteGroupNoPath
	case len(path) == 1:
		return paletteGroupSingleKey
	}
	if i := slices.Index(paletteKeyGroups, path[0]); i >= 0 {
		return i + 1
	}
	return paletteGroupOtherKey
}

// comparePaletteKeyPaths orders key paths by group, then by their label sequence.
func comparePaletteKeyPaths(a, b []string) int {
	if ga, gb := paletteKeyGroup(a), paletteKeyGroup(b); ga != gb {
		return ga - gb
	}
	return slices.Compare(a, b)
}

// paletteKeyPaths returns the key paths a binding is reachable by, as key labels.
func paletteKeyPaths(binding *types.Binding) [][]string {
	var paths [][]string
	for _, key := range binding.Keys {
		paths = append(paths, []string{config.LabelForKey(key)})
	}
	for _, path := range binding.CommandPaths {
		labels := make([]string, len(path))
		for i, key := range path {
			labels[i] = config.LabelForKey(key)
		}
		paths = append(paths, labels)
	}
	return paths
}

// paletteSortPath returns the path a binding is sorted by: its first path in
// key group order, or nil if it has no path.
func paletteSortPath(binding *types.Binding) []string {
	var best []string
	for _, path := range paletteKeyPaths(binding) {
		if best == nil || comparePaletteKeyPaths(path, best) < 0 {
			best = path
		}
	}
	return best
}

// sortPaletteBindingsByKeyGroup returns a copy of bindings ordered by key group:
// single keys first, then the groups of paletteKeyGroups, then other first
// keys, then commands without a key path. Within a group, commands are ordered
// by their key path. Ties keep their order. It calls no binding callbacks.
func sortPaletteBindingsByKeyGroup(bindings []*types.Binding) []*types.Binding {
	type ranked struct {
		binding *types.Binding
		path    []string
	}
	items := make([]ranked, len(bindings))
	for i, binding := range bindings {
		items[i] = ranked{binding: binding, path: paletteSortPath(binding)}
	}
	slices.SortStableFunc(items, func(a, b ranked) int {
		return comparePaletteKeyPaths(a.path, b.path)
	})
	sorted := make([]*types.Binding, len(items))
	for i, item := range items {
		sorted[i] = item.binding
	}
	return sorted
}
