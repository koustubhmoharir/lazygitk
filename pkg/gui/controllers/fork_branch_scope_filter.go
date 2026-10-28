package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/filtering"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: w b in Branches shows the selected local branch with its upstream and their
// ancestors. The commit scope is stored by branch name, and Commits opens with the
// cursor on the branch's tip. The same filter is offered from Commits (see
// fork_commit_scope_filters.go). The integration point is in BranchesController.
func (self *BranchesController) branchScopeFilterBinding() *types.Binding {
	return &types.Binding{
		CommandID:   "filter.branch-and-upstream",
		Verb:        "filter",
		Description: self.c.Tr.FilterBranchAndUpstream,
		Handler:     self.withItem(self.applyBranchScopeFilter),
		GetDisabledReason: self.require(self.singleItemSelected(func(branch *models.Branch) *types.DisabledReason {
			if branch.DetachedHead {
				return &types.DisabledReason{Text: self.c.Tr.CommitScopeSelectBranch, Inapplicable: true}
			}
			return nil
		})),
	}
}

func (self *BranchesController) applyBranchScopeFilter(branch *models.Branch) error {
	return self.c.Helpers().Mode.SetCommitScopeFilter(filtering.CommitScope{Branch: branch.Name}, branch.CommitHash)
}
