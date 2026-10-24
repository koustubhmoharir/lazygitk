package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (self *BasicCommitsController) addCherryMarkBindings(bindings []*types.Binding) []*types.Binding {
	disabled := self.require(self.itemRangeSelected(func(commits []*models.Commit, _, _ int) *types.DisabledReason {
		for _, commit := range commits {
			if commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
				return &types.DisabledReason{Text: self.c.Tr.CannotCherryPickNonCommit, Inapplicable: true}
			}
		}
		return nil
	}))
	return append([]*types.Binding{
		{
			CommandID: "commit.mark-cherry-pick", Verb: "mark-cherry-pick", Description: self.c.Tr.MarkForCherryPick, GetDisabledReason: disabled,
			Handler: func() error { return self.setCherryMarks(true) },
		},
		{
			CommandID: "commit.unmark-cherry-pick", Verb: "unmark-cherry-pick", Description: self.c.Tr.UnmarkForCherryPick, GetDisabledReason: disabled,
			Handler: func() error { return self.setCherryMarks(false) },
		},
	}, bindings...)
}

func (self *BasicCommitsController) setCherryMarks(mark bool) error {
	commits, _, _ := self.context.GetSelectedItems()
	for _, commit := range commits {
		if commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
			return nil
		}
	}
	if len(commits) == 0 {
		return nil
	}
	if mark {
		self.c.Modes().CherryPicking.MarkCommits(commits)
	} else {
		self.c.Modes().CherryPicking.UnmarkCommits(commits)
	}
	for _, ctx := range []types.Context{self.c.Contexts().LocalCommits, self.c.Contexts().ReflogCommits, self.c.Contexts().SubCommits} {
		self.c.PostRefreshUpdate(ctx)
	}
	return nil
}
