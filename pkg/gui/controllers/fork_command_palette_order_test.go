package controllers

import (
	"reflect"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

func keyPath(keys ...rune) []gocui.Key {
	path := make([]gocui.Key, len(keys))
	for i, r := range keys {
		path[i] = gocui.NewKeyRune(r)
	}
	return path
}

func TestSortPaletteBindingsByKeyGroup(t *testing.T) {
	bindings := []*types.Binding{
		{CommandID: "no-path", Description: "No path"},
		{CommandID: "other-z", Description: "Other z", CommandPaths: [][]gocui.Key{keyPath('z', 'q')}},
		{CommandID: "w", Description: "W", CommandPaths: [][]gocui.Key{keyPath('w', 'a')}},
		{CommandID: "t", Description: "T", CommandPaths: [][]gocui.Key{keyPath('t', 'a')}},
		{CommandID: "y", Description: "Y", CommandPaths: [][]gocui.Key{keyPath('y', 'a')}},
		{CommandID: "u", Description: "U", CommandPaths: [][]gocui.Key{keyPath('u', 'a')}},
		{CommandID: "c", Description: "C", CommandPaths: [][]gocui.Key{keyPath('c', 'a')}},
		{CommandID: "n", Description: "N", CommandPaths: [][]gocui.Key{keyPath('n', 'a')}},
		{CommandID: "s", Description: "S", CommandPaths: [][]gocui.Key{keyPath('s', 'a')}},
		{CommandID: "g-b", Description: "G b", CommandPaths: [][]gocui.Key{keyPath('g', 'b')}},
		{CommandID: "g-a", Description: "G a", CommandPaths: [][]gocui.Key{keyPath('g', 'a')}},
		{CommandID: "d-w", Description: "D w", CommandPaths: [][]gocui.Key{keyPath('d', 'w')}},
		{CommandID: "a-b", Description: "A b", CommandPaths: [][]gocui.Key{keyPath('a', 'b')}},
		{CommandID: "single-x", Description: "Single x", Keys: []gocui.Key{gocui.NewKeyRune('x')}},
		{CommandID: "single-b", Description: "Single b", Keys: []gocui.Key{gocui.NewKeyRune('b')}},
		// A single key path sorts ahead of the multi-key paths of the same command.
		{CommandID: "mixed", Description: "Mixed", Keys: []gocui.Key{gocui.NewKeyRune('m')}, CommandPaths: [][]gocui.Key{keyPath('d', 'a')}},
	}

	got := bindingIDs(sortPaletteBindingsByKeyGroup(bindings))
	want := []string{
		"single-b", "mixed", "single-x",
		"d-w", "s", "n", "c", "u", "y", "g-a", "g-b", "t", "w",
		"a-b", "other-z",
		"no-path",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestSortPaletteBindingsByKeyGroupKeepsDeclarationOrderForTies(t *testing.T) {
	bindings := []*types.Binding{
		{CommandID: "first", CommandPaths: [][]gocui.Key{keyPath('d', 'x')}},
		{CommandID: "no-path-1"},
		{CommandID: "second", CommandPaths: [][]gocui.Key{keyPath('d', 'x')}},
		{CommandID: "no-path-2"},
	}

	got := bindingIDs(sortPaletteBindingsByKeyGroup(bindings))
	if want := []string{"first", "second", "no-path-1", "no-path-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if bindings[0].CommandID != "first" {
		t.Fatal("sorting changed the input slice")
	}
}

func TestSortPaletteBindingsByKeyGroupDoesNotCallCallbacks(t *testing.T) {
	calls := 0
	binding := &types.Binding{
		CommandID:       "dynamic",
		DescriptionFunc: func() string { calls++; return "dynamic" },
		TooltipFunc:     func() string { calls++; return "" },
		GetDisabledReason: func() *types.DisabledReason {
			calls++
			return nil
		},
	}

	sortPaletteBindingsByKeyGroup([]*types.Binding{binding})
	if calls != 0 {
		t.Fatalf("callbacks called %d times", calls)
	}
}

func TestOrderPaletteViewsFollowsPaneOrderThenName(t *testing.T) {
	got := orderPaletteViews([]string{"zzz", "commits", "mergeConflicts", "aaa", "status", "stash"})
	want := []string{"status", "commits", "stash", "mergeConflicts", "aaa", "zzz"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestPaletteViewTitles(t *testing.T) {
	tr := i18n.EnglishTranslationSet()

	if got, want := paletteViewTitle("localBranches", tr), "Local branches"; got != want {
		t.Fatalf("localBranches title = %q, want %q", got, want)
	}
	if got, want := paletteFocusedTitle("localBranches", tr), "Local branches and global"; got != want {
		t.Fatalf("focused title = %q, want %q", got, want)
	}
	if got, want := paletteViewTitle("unlisted", tr), "unlisted"; got != want {
		t.Fatalf("unlisted view title = %q, want the view name %q", got, want)
	}
}
