package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// Each item with a key shows a cyan arrow between its key and its label; items
// without one get a blank cell there.
func TestMenuKeyArrowColumn(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	keyed := &types.MenuItem{Keys: []gocui.Key{gocui.NewKeyRune('c')}, LabelColumns: []string{"Delete"}}
	keyless := &types.MenuItem{LabelColumns: []string{"Other"}}
	menu.SetMenuItems([]*types.MenuItem{keyed, keyless}, nil)

	rows := menu.GetDisplayStrings(0, 0)
	// Columns: key, arrow, check mark, label.
	assert.Equal(t, style.FgCyan.Sprint("→"), rows[0][1])
	assert.Equal(t, "", rows[1][1])
	assert.Equal(t, "Delete", rows[0][3])
	assert.Equal(t, "Other", rows[1][3])
}

// A menu where no item has a key has no visible arrow column.
func TestMenuKeyArrowColumnHiddenWithoutKeys(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	menu.SetMenuItems([]*types.MenuItem{{LabelColumns: []string{"One"}}, {LabelColumns: []string{"Two"}}}, nil)

	lines, _ := utils.RenderDisplayStrings(menu.GetDisplayStrings(0, 0), nil)
	for _, line := range lines {
		assert.NotContains(t, line, "→")
	}
}

// Prefix menus build their rows through the same display strings.
func TestPrefixMenuKeyArrowColumn(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	menu.SetBlockUnmatchedKeys([]gocui.Key{gocui.NewKeyRune('q')})
	menu.SetMenuItems([]*types.MenuItem{{Keys: []gocui.Key{gocui.NewKeyRune('d')}, LabelColumns: []string{"Diff"}}}, nil)

	rows := menu.GetDisplayStrings(0, 0)
	assert.Equal(t, style.FgCyan.Sprint("→"), rows[0][1])
}
