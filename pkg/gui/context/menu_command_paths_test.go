package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestMenuCommandPathsDisplayAndFilterAsWholePaths(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	binding := &types.Binding{CommandPaths: [][]gocui.Key{{gocui.NewKeyRune('d'), gocui.NewKeyRune('w')}}}
	compare := &types.MenuItem{Binding: binding, LabelColumns: []string{"Compare"}}
	ordinary := &types.MenuItem{Keys: []gocui.Key{gocui.NewKeyRune('z'), gocui.NewKeyRune('a')}, LabelColumns: []string{"Other"}}
	menu.SetMenuItems([]*types.MenuItem{compare, ordinary}, nil)
	labels := menu.GetDisplayStrings(0, 0)
	assert.Contains(t, labels[0][0], "d w")
	assert.Contains(t, labels[1][0], "z")
	assert.NotContains(t, labels[1][0], "a", "ordinary menus retain a single displayed accelerator")
	menu.SetAllowFilteringKeybindings(true)
	menu.SetFilter("@d w", false)
	assert.Equal(t, []*types.MenuItem{compare}, menu.GetItems())
	menu.SetFilter("@a", false)
	assert.Equal(t, []*types.MenuItem{ordinary}, menu.GetItems(), "undisplayed alternates remain searchable")
	assert.Empty(t, compare.Keys, "displaying a sequence must not register its continuation as a direct accelerator")
}
