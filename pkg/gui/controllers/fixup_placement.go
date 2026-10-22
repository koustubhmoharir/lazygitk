package controllers

import "github.com/jesseduffield/lazygit/pkg/commands/models"

// Fork replacement for post-commit placement: decide whether a rebase is
// required and reject remaining tracked dirt before creating any commit.
func (self *LocalCommitsController) createCommitWithFixupPlacement(
	target *models.Commit, selectedIdx int, commits []*models.Commit, branches []*models.Branch,
	includeFileChanges bool, create func() error,
) error {
	ownerIdx := self.fixupOwnerBranchIndex(target, selectedIdx, commits, branches)
	if ownerIdx >= 0 {
		if err := self.c.Git().Rebase.CheckCleanAfterCommit(includeFileChanges); err != nil {
			return err
		}
	}
	if err := create(); err != nil {
		return err
	}
	if ownerIdx < 0 {
		return nil
	}
	// The backend still checks again after the commit, including hook changes.
	return self.c.Git().Rebase.MoveFixupCommitDown(commits, ownerIdx)
}

func (self *LocalCommitsController) fixupOwnerBranchIndex(
	target *models.Commit, selectedIdx int, commits []*models.Commit, branches []*models.Branch,
) int {
	git := self.c.Git()
	if git.Version.IsOlderThan(2, 38, 0) || git.Status.WorkingTreeState().Any() ||
		target.Status == models.StatusMerged || !git.Config.GetRebaseUpdateRefs() {
		return -1
	}
	for i := selectedIdx; i > 0; i-- {
		for _, branch := range branches {
			if branch.CommitHash == commits[i].Hash() {
				return i
			}
		}
	}
	return -1
}
