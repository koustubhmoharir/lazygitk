package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork replacement for post-commit placement: decide whether a rebase is
// required and reject remaining tracked dirt before creating any commit. The
// ignored-file review also runs before the commit, so a refusal leaves nothing
// behind. Only the placement refusal can arrive after the commit; that one is
// returned as an error, and is not retried, so the commit is never made twice.
func (self *LocalCommitsController) createCommitWithFixupPlacement(
	target *models.Commit, selectedIdx int, commits []*models.Commit, branches []*models.Branch,
	includeFileChanges bool, create func() error,
) error {
	ownerIdx := self.fixupOwnerBranchIndex(target, selectedIdx, commits, branches)
	if ownerIdx >= 0 {
		if err := self.c.Git().Rebase.CheckCleanAfterCommit(includeFileChanges); err != nil {
			return err
		}
		if err := self.c.Git().Rebase.CheckCompoundRebaseIgnoredRisks(commits, ownerIdx, includeFileChanges); err != nil {
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
	return placementRiskAfterCommit(self.c.Git().Rebase.MoveFixupCommitDown(commits, ownerIdx))
}

// finishAfterFixupPlacement is the finish of the fixup and amend flows. A placement
// refused after the commit was made still means the commit exists: its message
// is no longer a draft, and the list refreshes, so a resubmit cannot make it twice.
func (self *LocalCommitsController) finishAfterFixupPlacement(err error, reportSuccess func()) error {
	if err != nil && !errors.Is(err, git_commands.ErrRebaseRiskAfterCommit) {
		return err
	}
	if reportSuccess != nil {
		reportSuccess()
	}
	self.c.RefreshFromWorker(types.RefreshOptions{BatchUIUpdates: true})
	return err
}

// placementRiskAfterCommit reports a placement refused after its commit was made.
func placementRiskAfterCommit(err error) error {
	var riskErr *git_commands.IgnoredRebaseRiskError
	if errors.As(err, &riskErr) {
		return git_commands.ErrRebaseRiskAfterCommit
	}
	return err
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
