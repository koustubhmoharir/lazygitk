package controllers

import "errors"

// Terminal output is shown directly, not captured in the exit error. Inspect
// native operation/index state instead of searching an exit status for Git's
// conflict diagnostics. A paused operation alone is not evidence of conflict:
// hook/signing failures must remain errors, without automatic skip or continue.
func (target *branchSyncTarget) integrationSubprocessResult(result error) error {
	if err := target.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	rebasing, err := target.git.Status.IsInRebase()
	if err != nil {
		return errors.Join(result, err)
	}
	merging, err := target.git.Status.IsInMergeState()
	if err != nil {
		return errors.Join(result, err)
	}
	if err := target.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	target.c.State().GetRepoState().SetMergeOrRebaseStartedInLazygit(rebasing || merging)
	if result == nil || (!rebasing && !merging) {
		return result
	}
	unmerged, err := target.git.Rebase.HasUnmergedFiles()
	if err != nil {
		return errors.Join(result, err)
	}
	if err := target.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	if unmerged {
		return target.c.Helpers().MergeAndRebase.PromptForConflictHandling()
	}
	return result
}
