package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// An inapplicable reason is faded, while a failed precondition is struck through.
func TestMenuDisabledItemsDisplayByReason(t *testing.T) {
	menu, _ := newMenuBeforeAcceptContext()
	inapplicable := &types.MenuItem{LabelColumns: []string{"Elsewhere"}, DisabledReason: &types.DisabledReason{Text: "Available in Commits", Inapplicable: true}}
	precondition := &types.MenuItem{LabelColumns: []string{"Blocked"}, DisabledReason: &types.DisabledReason{Text: "Nothing to do"}}
	menu.SetMenuItems([]*types.MenuItem{inapplicable, precondition}, nil)

	// Each row starts with the key and check mark columns.
	labels := menu.GetDisplayStrings(0, 0)
	assert.Equal(t, style.FgDefault.SetDim().Sprint("Elsewhere"), labels[0][2], "inapplicable items are dimmed")
	assert.Equal(t, style.FgDefault.SetStrikethrough().Sprint("Blocked"), labels[1][2], "precondition failures are struck through")
	assert.Equal(t, []string{"Elsewhere"}, inapplicable.LabelColumns, "styling does not change the item")

	again := menu.GetDisplayStrings(0, 0)
	assert.Equal(t, labels, again, "rendering again does not accumulate styling")
}
