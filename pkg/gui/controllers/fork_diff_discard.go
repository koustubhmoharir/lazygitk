package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// The staged-side operation is only unstage and remains immediate. Destructive
// worktree discards always require approval, including when upstream's optional
// warning is disabled in user config.
func (self *WorkingTreeDiffActions) confirmDiffDiscard(staged bool, apply func() error) error {
	if staged {
		return apply()
	}
	git, generation := self.c.Git(), self.c.State().GetRepoGeneration()
	self.c.Confirm(types.ConfirmOpts{
		Title: self.c.Tr.DiscardChangeTitle, Prompt: self.c.Tr.ForkDiscardChangePrompt,
		HandleConfirm: func() error {
			if self.c.Git() != git || self.c.State().GetRepoGeneration() != generation {
				return errors.New(self.c.Tr.BranchSyncRepositoryChanged)
			}
			return apply()
		},
	})
	return nil
}
