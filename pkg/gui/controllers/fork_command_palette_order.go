package controllers

import (
	"cmp"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

// Fork: the `?` palette orders local and global commands by usage count
// (roadmap item 9). The `gui.commandPaletteOrder: static` option keeps the
// upstream Local/Global/Navigation sections instead. Navigation stays static.

type paletteScope int

const (
	paletteScopeLocal paletteScope = iota
	paletteScopeGlobal
)

type paletteEntry struct {
	binding *types.Binding
	scope   paletteScope
}

// orderPaletteByUsage merges local and global commands into one list sorted by
// usage count, highest first. Counts may be negative, and a missing count is
// zero. Ties keep declaration order: local before global, then the original
// order within each list.
func orderPaletteByUsage(local, global []*types.Binding, counts map[string]int) []paletteEntry {
	entries := make([]paletteEntry, 0, len(local)+len(global))
	for _, binding := range local {
		entries = append(entries, paletteEntry{binding: binding, scope: paletteScopeLocal})
	}
	for _, binding := range global {
		entries = append(entries, paletteEntry{binding: binding, scope: paletteScopeGlobal})
	}
	uses := make([]int, len(entries))
	for i, entry := range entries {
		uses[i] = counts[entry.binding.GetCommandID()]
	}
	order := make([]int, len(entries))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(uses[b], uses[a])
	})
	sorted := make([]paletteEntry, len(entries))
	for i, index := range order {
		sorted[i] = entries[index]
	}
	return sorted
}

// forkTagPaletteItem adds a dim scope column after the label. Filtering keeps
// matching only the label, so typing "global" does not select every global command.
func forkTagPaletteItem(item *types.MenuItem, scope paletteScope, tr *i18n.TranslationSet) {
	tag := tr.CommandPaletteScopeGlobal
	if scope == paletteScopeLocal {
		tag = tr.CommandPaletteScopeLocal
	}
	item.LabelColumns = []string{item.Label, style.FgDefault.SetDim().Sprint(tag)}
	item.FilterColumns = []string{item.Label}
}
