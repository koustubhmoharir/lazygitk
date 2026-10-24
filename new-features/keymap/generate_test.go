//go:build ignore

package main

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
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

func TestCommandPathsRemainFullAndDoNotCollideOnSharedRoot(t *testing.T) {
	g := keyRow{Name: int(tcell.KeyRune), Str: "g"}
	a := keyRow{Name: int(tcell.KeyRune), Str: "a"}
	b := keyRow{Name: int(tcell.KeyRune), Str: "b"}
	rows := []bindingRow{
		{Declaration: 0, ID: "first", Verb: "first", View: "commits", RawCommandPaths: [][]keyRow{{g, a}}},
		{Declaration: 1, ID: "second", Verb: "second", View: "commits", RawCommandPaths: [][]keyRow{{g, b}}},
	}
	if collisions := analyze("linux", rows, nil); len(collisions) != 0 {
		t.Fatalf("shared prefix root was reported as a collision: %#v", collisions)
	}
	rows[1].RawCommandPaths = [][]keyRow{{g, a}}
	collisions := analyze("linux", rows, nil)
	if len(collisions) != 1 || !reflect.DeepEqual(collisions[0].Path, canonicalPath([]keyRow{g, a})) {
		t.Fatalf("complete-path collision = %#v", collisions)
	}
}

func TestRowsRetainCommandPathLabelsAndRawTuples(t *testing.T) {
	path := []gocui.Key{gocui.NewKeyRune('g'), gocui.NewKey(gocui.KeyName(tcell.KeyRune), "A", gocui.ModShift)}
	rows := rowsFor([]*types.Binding{{CommandID: "sample", CommandPaths: [][]gocui.Key{path}}}, nil)
	wantRaw := [][]keyRow{{
		{Name: int(path[0].KeyName()), Str: path[0].Str(), Mod: int(path[0].Mod()), Label: config.LabelForKey(path[0])},
		{Name: int(path[1].KeyName()), Str: path[1].Str(), Mod: int(path[1].Mod()), Label: config.LabelForKey(path[1])},
	}}
	if !reflect.DeepEqual(rows[0].CommandPaths, []string{"g <shift+A>"}) || !reflect.DeepEqual(rows[0].RawCommandPaths, wantRaw) {
		t.Fatalf("command paths lost labels or raw tuples: %#v", rows[0])
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

func TestClearedCommandOverridesRetainMetadataAndPathsDisappear(t *testing.T) {
	guiInstance, cleanup, _ := instantiate("linux", false)
	defer cleanup()
	defaults := rowsFor(guiInstance.GetCheatsheetKeybindings(), guiInstance.State.Contexts.Flatten())
	edges := viewParentEdges(guiInstance)
	if len(edges) != 2 || edges[0].ActiveByDefault || edges[1].ActiveByDefault {
		t.Fatalf("initialized parent view edges = %#v, want two inactive-by-default candidates", edges)
	}
	clearedGui, clearCleanup, clearedCommandIDs := instantiate("linux", true)
	defer clearCleanup()
	cleared := rowsFor(clearedGui.GetCheatsheetKeybindings(), clearedGui.State.Contexts.Flatten())
	matchClearedRows(defaults, cleared)
	clearedIDs := make(map[string]bool, len(clearedCommandIDs))
	for _, id := range clearedCommandIDs {
		clearedIDs[id] = true
	}
	defaultByIdentity := make(map[string]bindingRow, len(defaults))
	for _, row := range defaults {
		defaultByIdentity[identityKey(row)] = row
	}
	for _, row := range cleared {
		if defaultRow, ok := defaultByIdentity[identityKey(row)]; ok && row.DefaultDeclaration != nil {
			if row.ID != defaultRow.ID || row.Verb != defaultRow.Verb || row.Description != defaultRow.Description || row.View != defaultRow.View || row.Tag != defaultRow.Tag {
				t.Errorf("clearing routes changed action metadata for %s", row.ID)
			}
			if len(defaultRow.RawCommandPaths) > 0 && (len(row.RawCommandPaths) != 0 || len(row.CommandPaths) != 0) {
				t.Errorf("cleared command %s retained command paths: %v", row.ID, row.CommandPaths)
			}
		}
	}
	for _, row := range defaults {
		if len(row.RawCommandPaths) > 0 && !clearedIDs[row.ID] {
			t.Errorf("projected command %s has no explicit cleared override", row.ID)
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
