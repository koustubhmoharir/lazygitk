package controllers

import (
	"reflect"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

func paletteEntryIDs(entries []paletteEntry) []string {
	ids := make([]string, len(entries))
	for i, entry := range entries {
		ids[i] = entry.binding.GetCommandID()
	}
	return ids
}

func paletteEntryScopes(entries []paletteEntry) []paletteScope {
	scopes := make([]paletteScope, len(entries))
	for i, entry := range entries {
		scopes[i] = entry.scope
	}
	return scopes
}

func TestOrderPaletteByUsageSortsByCountDescending(t *testing.T) {
	local := []*types.Binding{{CommandID: "a"}, {CommandID: "b"}, {CommandID: "c"}}
	global := []*types.Binding{{CommandID: "d"}, {CommandID: "e"}}
	counts := map[string]int{"b": 5, "d": 9, "e": 1, "c": 1}

	entries := orderPaletteByUsage(local, global, counts)

	if got, want := paletteEntryIDs(entries), []string{"d", "b", "c", "e", "a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if got, want := paletteEntryScopes(entries), []paletteScope{paletteScopeGlobal, paletteScopeLocal, paletteScopeLocal, paletteScopeGlobal, paletteScopeLocal}; !reflect.DeepEqual(got, want) {
		t.Fatalf("scopes = %v, want %v", got, want)
	}
}

func TestOrderPaletteByUsageSortsNegativeCountsBelowUnused(t *testing.T) {
	local := []*types.Binding{{CommandID: "shortcut"}, {CommandID: "unused"}}
	global := []*types.Binding{{CommandID: "used"}}
	counts := map[string]int{"shortcut": -3, "used": 1}

	entries := orderPaletteByUsage(local, global, counts)

	if got, want := paletteEntryIDs(entries), []string{"used", "unused", "shortcut"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestOrderPaletteByUsageTiesKeepDeclarationOrder(t *testing.T) {
	local := []*types.Binding{{CommandID: "local-1"}, {CommandID: "local-2"}}
	global := []*types.Binding{{CommandID: "global-1"}, {CommandID: "global-2"}}

	entries := orderPaletteByUsage(local, global, map[string]int{"global-2": 2, "local-2": 2})

	if got, want := paletteEntryIDs(entries), []string{"local-2", "global-2", "local-1", "global-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestOrderPaletteByUsageWithoutCountsKeepsLocalThenGlobal(t *testing.T) {
	local := []*types.Binding{{CommandID: "local-1"}, {CommandID: "local-2"}}
	global := []*types.Binding{{CommandID: "global-1"}}

	entries := orderPaletteByUsage(local, global, nil)

	if got, want := paletteEntryIDs(entries), []string{"local-1", "local-2", "global-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestOrderPaletteByUsageUsesDynamicCommandIDs(t *testing.T) {
	local := []*types.Binding{{CommandID: "stale", CommandIDFunc: func() string { return "dynamic" }}, {CommandID: "plain"}}

	entries := orderPaletteByUsage(local, nil, map[string]int{"dynamic": 4})

	if got, want := paletteEntryIDs(entries), []string{"dynamic", "plain"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestForkTagPaletteItemAddsScopeColumnAndFiltersByLabel(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	local := &types.MenuItem{Label: "Stage file"}
	global := &types.MenuItem{Label: "Refresh"}

	forkTagPaletteItem(local, paletteScopeLocal, tr)
	forkTagPaletteItem(global, paletteScopeGlobal, tr)

	if got, want := local.LabelColumns, []string{"Stage file", style.FgDefault.SetDim().Sprint("local")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("local columns = %v, want %v", got, want)
	}
	if got, want := global.LabelColumns, []string{"Refresh", style.FgDefault.SetDim().Sprint("global")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("global columns = %v, want %v", got, want)
	}
	if got, want := global.FilterColumns, []string{"Refresh"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("filter columns = %v, want %v", got, want)
	}
}
