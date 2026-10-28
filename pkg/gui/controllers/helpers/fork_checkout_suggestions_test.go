package helpers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

const (
	forkTestTipOID   = "1111111111111111111111111111111111111111"
	forkTestOtherOID = "2222222222222222222222222222222222222222"
)

func forkSuggestionValues(suggestions []*types.Suggestion) []string {
	values := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		values = append(values, suggestion.Value)
	}
	return values
}

func TestBranchNameSuggestionsAtCommitListBranchesThereFirst(t *testing.T) {
	ui := &namedCheckoutFakeIGuiCommon{}
	ui.model.Branches = []*models.Branch{
		{Name: "main", CommitHash: forkTestOtherOID},
		{Name: "feature-a", CommitHash: forkTestTipOID},
		{Name: "detached-entry", CommitHash: forkTestTipOID, DetachedHead: true},
		{Name: "zeta", CommitHash: forkTestTipOID},
		{Name: "dev", CommitHash: forkTestOtherOID},
	}
	c := &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}

	values := forkSuggestionValues(NewSuggestionsHelper(c).GetBranchNameSuggestionsAtCommitFunc(forkTestTipOID)(""))

	assert.Equal(t, []string{"feature-a", "zeta", "main", "detached-entry", "dev"}, values)
}

func TestBranchNameSuggestionsAtCommitHaveNoDuplicates(t *testing.T) {
	ui := &namedCheckoutFakeIGuiCommon{}
	ui.model.Branches = []*models.Branch{
		{Name: "main", CommitHash: forkTestTipOID},
		{Name: "dev", CommitHash: forkTestOtherOID},
	}
	c := &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}

	values := forkSuggestionValues(NewSuggestionsHelper(c).GetBranchNameSuggestionsAtCommitFunc(forkTestTipOID)(""))

	assert.Equal(t, []string{"main", "dev"}, values)
}

func TestNamedCheckoutBeginUsesCommitSuggestions(t *testing.T) {
	ui := &namedCheckoutFakeIGuiCommon{}
	ui.model.Branches = []*models.Branch{
		{Name: "main", CommitHash: forkTestOtherOID},
		{Name: "topic", CommitHash: forkTestTipOID},
	}
	c := &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}
	flow := &namedCheckoutFlow{c: c, oid: forkTestTipOID, nameSuggestions: func() func(string) []*types.Suggestion {
		return NewSuggestionsHelper(c).GetBranchNameSuggestionsAtCommitFunc(forkTestTipOID)
	}}

	if err := flow.begin(); err != nil {
		t.Fatal(err)
	}

	if assert.Len(t, ui.prompts, 1) {
		assert.Equal(t, []string{"topic", "main"}, forkSuggestionValues(ui.prompts[0].FindSuggestionsFunc("")))
	}
}
