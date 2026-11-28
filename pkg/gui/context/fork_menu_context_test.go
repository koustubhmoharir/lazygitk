package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/jesseduffield/lazygit/pkg/utils/promptmark"
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

// Fork: role markers in a label show as the prompt styles, so a hash in a
// label is coloured like a hash in a prompt.
func TestMenuLabelMarksResolveToColour(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	menu.SetMenuItems([]*types.MenuItem{{LabelColumns: []string{"Mark " + promptmark.Hash("5d59a479") + " as bad"}}}, nil)

	rows := menu.GetDisplayStrings(0, 0)
	// Columns: key, arrow, check mark, label.
	assert.Equal(t, "Mark "+style.FgYellow.Sprint("5d59a479")+" as bad", rows[0][3])
}

// Fork: a disabled item's first column is struck through as a whole, so it
// shows no inner colours and no markers. Its other columns still resolve.
func TestMenuLabelMarksDisabledItemHasNoInnerColour(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	menu.SetMenuItems([]*types.MenuItem{{
		LabelColumns:   []string{"Filter by " + promptmark.Path("path/file.go"), promptmark.Hash("abc123")},
		DisabledReason: &types.DisabledReason{Text: "not now"},
	}}, nil)

	rows := menu.GetDisplayStrings(0, 0)
	assert.Equal(t, style.FgDefault.SetStrikethrough().Sprint("Filter by path/file.go"), rows[0][3])
	assert.Equal(t, style.FgYellow.Sprint("abc123"), rows[0][4])
}

// Fork: filtering matches the plain text of a marked label, not its markers.
func TestMenuLabelMarksFilterMatchesPlainText(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	marked := &types.MenuItem{LabelColumns: []string{"Filter by " + promptmark.Path("path/file.go")}}
	other := &types.MenuItem{LabelColumns: []string{"Other"}}
	menu.SetMenuItems([]*types.MenuItem{marked, other}, nil)

	menu.SetFilter("by path/file.go", false)
	assert.Equal(t, []*types.MenuItem{marked}, menu.GetFilteredList())

	menu.SetFilter(string(rune(0xE000)), false)
	assert.Empty(t, menu.GetFilteredList())
}
