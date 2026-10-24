//go:build ignore

package main

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

func TestNormalizeKeyOnlyFoldsASCIICommandLetters(t *testing.T) {
	a := keyRow{Name: int(tcell.KeyRune), Str: "a", Mod: int(gocui.ModNone)}
	upper := keyRow{Name: int(tcell.KeyRune), Str: "A", Mod: int(gocui.ModNone)}
	shiftUpper := keyRow{Name: int(tcell.KeyRune), Str: "A", Mod: int(gocui.ModShift)}
	if normalizeRawKey(a) != normalizeRawKey(upper) || normalizeRawKey(a) != normalizeRawKey(shiftUpper) {
		t.Fatal("ASCII letter variants should normalize equally")
	}
	punct := keyRow{Name: int(tcell.KeyRune), Str: "!", Mod: int(gocui.ModNone)}
	shiftPunct := keyRow{Name: int(tcell.KeyRune), Str: "!", Mod: int(gocui.ModShift)}
	if normalizeRawKey(punct) == normalizeRawKey(shiftPunct) {
		t.Fatal("shift punctuation must remain distinct")
	}
	ctrlUpper := keyRow{Name: int(tcell.KeyRune), Str: "A", Mod: int(gocui.ModCtrl)}
	if normalizeRawKey(a) == normalizeRawKey(ctrlUpper) {
		t.Fatal("control combinations must remain distinct")
	}
	if normalizeRawKey(keyRow{Name: int(tcell.KeyRune), Str: "é"}) == normalizeRawKey(keyRow{Name: int(tcell.KeyRune), Str: "É"}) {
		t.Fatal("non-ASCII text must not be folded")
	}
}

func TestCollisionClassificationRetainsDeclarationOrder(t *testing.T) {
	rows := []bindingRow{
		{Declaration: 4, ID: "same", Verb: "open", View: "files", RawKeys: []keyRow{{Name: int(tcell.KeyRune), Str: "a"}}},
		{Declaration: 7, ID: "same", Verb: "open", View: "files", RawKeys: []keyRow{{Name: int(tcell.KeyRune), Str: "A"}}},
		{Declaration: 9, ID: "other", Verb: "open", View: "files", RawKeys: []keyRow{{Name: int(tcell.KeyRune), Str: "a"}}},
	}
	got := analyze("linux", rows, nil)
	if len(got) != 1 {
		t.Fatalf("collision count = %d, want 1", len(got))
	}
	if got[0].Class != "different-id-shared-verb" {
		t.Errorf("class = %q", got[0].Class)
	}
	if !reflect.DeepEqual(got[0].Order, []int{4, 7, 9}) {
		t.Errorf("order = %v", got[0].Order)
	}
	if !reflect.DeepEqual(got[0].IDs, []string{"same", "same", "other"}) {
		t.Errorf("IDs = %v", got[0].IDs)
	}
}

func TestCollisionClassificationsAndGlobalParentOrder(t *testing.T) {
	letter := keyRow{Name: int(tcell.KeyRune), Str: "x"}
	rows := []bindingRow{
		{Declaration: 0, ID: "alias", Verb: "open", View: "menuFilter", RawKeys: []keyRow{letter}},
		{Declaration: 1, ID: "alias", Verb: "open", View: "menuFilter", RawKeys: []keyRow{letter}},
		{Declaration: 2, ID: "different", Verb: "close", View: "menu", RawKeys: []keyRow{letter}},
		{Declaration: 3, ID: "global", Verb: "open", View: "", RawKeys: []keyRow{letter}},
	}
	edges := []parentEdge{{Child: "menuFilter", Parent: "menu", Evidence: "test injected edge"}}
	got := analyze("linux", rows, edges)
	find := func(kind string) collision {
		t.Helper()
		for _, item := range got {
			if item.ScopeKind == kind {
				return item
			}
		}
		t.Fatalf("no %s collision: %#v", kind, got)
		return collision{}
	}
	if c := find("view"); c.Class != "same-id-same-verb-alias" {
		t.Errorf("same view class = %s", c.Class)
	}
	parent := find("local-parent")
	if parent.Class != "different-id-different-verb" || !reflect.DeepEqual(parent.Order, []int{0, 1, 2}) {
		t.Errorf("parent candidate order/class = %v/%s", parent.Order, parent.Class)
	}
	findScope := func(scope string) collision {
		t.Helper()
		for _, item := range got {
			if item.ScopeKind == "local-global" && item.Scope == scope {
				return item
			}
		}
		t.Fatalf("no local-global collision for %s", scope)
		return collision{}
	}
	menuFilterGlobal := findScope("menuFilter")
	if !reflect.DeepEqual(menuFilterGlobal.Order, []int{0, 1, 3}) || menuFilterGlobal.Class != "different-id-shared-verb" {
		t.Errorf("menuFilter/global order and class = %v/%s", menuFilterGlobal.Order, menuFilterGlobal.Class)
	}
	menuGlobal := findScope("menu")
	if !reflect.DeepEqual(menuGlobal.Order, []int{2, 3}) || menuGlobal.Class != "different-id-different-verb" {
		t.Errorf("menu/global order and class = %v/%s", menuGlobal.Order, menuGlobal.Class)
	}
}

func TestCanonicalCollisionKeyLabelMatchesRawTuple(t *testing.T) {
	got := canonicalKey(keyRow{Name: int(tcell.KeyRune), Str: "A", Mod: int(gocui.ModShift)})
	if got.Str != "a" || got.Mod != int(gocui.ModNone) || got.Label != "a" {
		t.Fatalf("canonical key has inconsistent tuple and label: %#v", got)
	}
}

func TestClearedRowsMatchByIdentityOrdinal(t *testing.T) {
	defaults := []bindingRow{{Declaration: 2, View: "files", ID: "same", Verb: "edit", IdentityOrdinal: 0}, {Declaration: 8, View: "files", ID: "same", Verb: "edit", IdentityOrdinal: 1}}
	cleared := []bindingRow{{Declaration: 0, View: "files", ID: "same", Verb: "edit", IdentityOrdinal: 1}}
	matchClearedRows(defaults, cleared)
	if cleared[0].DefaultDeclaration == nil || *cleared[0].DefaultDeclaration != 8 {
		t.Fatalf("matched default declaration = %v, want 8", cleared[0].DefaultDeclaration)
	}
}

func TestMergedLegacyAliasAppearsInGuiSnapshotAndClears(t *testing.T) {
	guiInstance, cleanup := instantiate("linux", false)
	defer cleanup()
	defaults := rowsFor(guiInstance.GetCheatsheetKeybindings(), guiInstance.State.Contexts.Flatten())
	edges := viewParentEdges(guiInstance)
	if len(edges) != 2 || edges[0].ActiveByDefault || edges[1].ActiveByDefault {
		t.Fatalf("initialized parent view edges = %#v, want two inactive-by-default candidates", edges)
	}
	clearedGui, clearCleanup := instantiate("linux", true)
	defer clearCleanup()
	cleared := rowsFor(clearedGui.GetCheatsheetKeybindings(), clearedGui.State.Contexts.Flatten())
	matchClearedRows(defaults, cleared)
	var quit *bindingRow
	for i := range defaults {
		if defaults[i].ID == "app.quit" {
			quit = &defaults[i]
			break
		}
	}
	if quit == nil {
		t.Fatal("serialized GUI inventory has no app.quit")
	}
	foundAlias := false
	for _, key := range quit.Keys {
		if key == "<ctrl+c>" {
			foundAlias = true
		}
	}
	if !foundAlias {
		t.Fatalf("merged legacy Ctrl+C quit alias absent from effective GUI keys: %v", quit.Keys)
	}
	for _, row := range cleared {
		if row.ID == "app.quit" {
			for _, key := range row.Keys {
				if key == "<ctrl+c>" {
					t.Fatal("cleared snapshot recreated legacy Ctrl+C")
				}
			}
		}
	}
	foundHardcoded := false
	for _, row := range cleared {
		if row.ID == "ui.information.activate" && len(row.Keys) > 0 {
			foundHardcoded = true
		}
	}
	if !foundHardcoded {
		t.Fatal("cleared snapshot lost known hardcoded information activation key")
	}
}
