package helpers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// CheckRebaseSubprocessResult runs on a worker after terminal interaction.
// Terminal output has already been shown to the user, but its exit error does
// not contain Git's conflict diagnostics. Inspect actual state, not that text.
func (self *MergeAndRebaseHelper) CheckRebaseSubprocessResult(result error, opts types.RefreshOptions) error {
	self.refreshAfterMergeOrRebase(opts, true)
	self.RecordWhetherMergeOrRebaseStartedInLazygit()
	if result == nil {
		return nil
	}
	rebasing, err := self.c.Git().Status.IsInRebase()
	if err != nil {
		return errors.Join(result, err)
	}
	if !rebasing {
		return result
	}
	unmerged, err := self.c.Git().Rebase.HasUnmergedFiles()
	if err != nil {
		return errors.Join(result, err)
	}
	if unmerged {
		return self.PromptForConflictHandling()
	}
	// Signing and hook failures can leave a paused rebase without conflicts.
	// Preserve those errors; the normal continue/skip/abort menu remains usable.
	return result
}
