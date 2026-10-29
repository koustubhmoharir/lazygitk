package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: w b in Branches shows the selected local branch with its upstream and their
// ancestors. The commit scope is stored by full ref name, and Commits opens with the
// cursor on the branch's tip. Commits offers the same filter for every ref at a commit
// (see fork_commit_scope_filters.go), and the shared logic is in fork_ref_scope_filter.go.
// The integration point is in BranchesController.
func (self *BranchesController) branchScopeFilterBinding() *types.Binding {
	return &types.Binding{
		CommandID:   "filter.ref",
		Verb:        "filter",
		Description: self.c.Tr.FilterRef,
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
	return applyRefScope(self.c, branch.FullRefName(), branch.CommitHash)
}
