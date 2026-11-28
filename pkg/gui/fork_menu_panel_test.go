package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func forkDigitTestItem(label string, keys ...gocui.Key) *types.MenuItem {
	return &types.MenuItem{LabelColumns: []string{label}, Keys: keys}
}

func forkDigitKey(r rune) gocui.Key { return gocui.NewKeyRune(r) }

func TestForkAssignDigitKeysNumbersKeylessItemsInOrder(t *testing.T) {
	a := forkDigitTestItem("a")
	b := forkDigitTestItem("b")
	c := forkDigitTestItem("c", forkDigitKey('x'))
	d := forkDigitTestItem("d")
	forkAssignDigitKeys(types.CreateMenuOptions{Items: []*types.MenuItem{a, b, c, d}})

	assert.Equal(t, []gocui.Key{forkDigitKey('1')}, a.Keys)
	assert.Equal(t, []gocui.Key{forkDigitKey('2')}, b.Keys)
	assert.Equal(t, []gocui.Key{forkDigitKey('x')}, c.Keys, "items with a key keep it")
	assert.Equal(t, []gocui.Key{forkDigitKey('3')}, d.Keys)
}

func TestForkAssignDigitKeysSkipsDigitsUsedByOtherItems(t *testing.T) {
	a := forkDigitTestItem("a")
	used := forkDigitTestItem("used", forkDigitKey('1'))
	binding := &types.MenuItem{LabelColumns: []string{"bound"}, Binding: &types.Binding{Keys: []gocui.Key{forkDigitKey('2')}}}
	b := forkDigitTestItem("b")
	forkAssignDigitKeys(types.CreateMenuOptions{Items: []*types.MenuItem{a, used, binding, b}})

	assert.Equal(t, []gocui.Key{forkDigitKey('3')}, a.Keys, "1 and 2 are used, so the first free digit is 3")
	assert.Equal(t, []gocui.Key{forkDigitKey('4')}, b.Keys)
}

func TestForkAssignDigitKeysLeavesCancelUnnumbered(t *testing.T) {
	a := forkDigitTestItem("a")
	cancel := &types.MenuItem{LabelColumns: []string{"Cancel"}, CommandID: "menu.cancel"}
	forkAssignDigitKeys(types.CreateMenuOptions{Items: []*types.MenuItem{a, cancel}})

	assert.Equal(t, []gocui.Key{forkDigitKey('1')}, a.Keys)
	assert.Empty(t, cancel.Keys)
}

func TestForkAssignDigitKeysGivesDisabledItemsDigits(t *testing.T) {
	disabled := forkDigitTestItem("disabled")
	disabled.DisabledReason = &types.DisabledReason{Text: "Nothing to do"}
	forkAssignDigitKeys(types.CreateMenuOptions{Items: []*types.MenuItem{disabled}})

	assert.Equal(t, []gocui.Key{forkDigitKey('1')}, disabled.Keys)
}

func TestForkAssignDigitKeysLeavesFilteringMenusAlone(t *testing.T) {
	a := forkDigitTestItem("a")
	forkAssignDigitKeys(types.CreateMenuOptions{Items: []*types.MenuItem{a}, FilterAsYouType: true})
	assert.Empty(t, a.Keys)
}

func TestForkAssignDigitKeysLeavesPrefixMenusAlone(t *testing.T) {
	a := forkDigitTestItem("a")
	forkAssignDigitKeys(types.CreateMenuOptions{Items: []*types.MenuItem{a}, BlockUnmatchedKeys: []gocui.Key{forkDigitKey('q')}})
	assert.Empty(t, a.Keys)
}

func TestForkAssignDigitKeysStopsAfterNineFreeDigits(t *testing.T) {
	items := make([]*types.MenuItem, 11)
	for i := range items {
		items[i] = forkDigitTestItem("item")
	}
	forkAssignDigitKeys(types.CreateMenuOptions{Items: items})

	for i := range 9 {
		assert.Equal(t, []gocui.Key{forkDigitKey(rune('1' + i))}, items[i].Keys)
	}
	assert.Empty(t, items[9].Keys, "only digits 1-9 exist")
	assert.Empty(t, items[10].Keys)
}

func TestForkAssignDigitKeysSkipsBindingItemsWithoutLabels(t *testing.T) {
	// A binding-backed item shows its labels from the binding, so a digit given
	// to it would run without being shown. Such items are left alone.
	bound := &types.MenuItem{LabelColumns: []string{"bound"}, Binding: &types.Binding{}}
	forkAssignDigitKeys(types.CreateMenuOptions{Items: []*types.MenuItem{bound}})
	assert.Empty(t, bound.Keys)
}
