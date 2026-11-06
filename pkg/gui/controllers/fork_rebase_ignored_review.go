package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

// Fork: generic history-editing starts (reword, move, drop, edit, squash and
// fixup, amend-to, author edits, custom patch and discard-from-commit) go through
// runWithIgnoredRebaseReview. The backend refuses such a start, before Git writes,
// when Git's rewind could overwrite ignored files, and the refusal arrives as an
// *IgnoredRebaseRiskError. Here the refusal opens the same review as the selected-
// commit rebase, and approval retries the same start under the reviewed fingerprint.
//
// start runs the rebase start on the worker, and finish runs once with its outcome,
// as the caller's own post-processing did. On a refusal, finish is not called until
// the approved start has run. A finish that needs the UI thread queues its own work.

// runWithIgnoredRebaseReview runs start and then finish. The caller must be on the
// worker or on the UI thread, as it is for start itself.
func runWithIgnoredRebaseReview(
	c *ControllerCommon,
	start func(rebase *git_commands.RebaseCommands) error,
	finish func(err error) error,
) error {
	git := c.Git()
	err := start(git.Rebase)
	var riskErr *git_commands.IgnoredRebaseRiskError
	if !errors.As(err, &riskErr) {
		return finishIgnoredRebaseStart(c, finish, err)
	}
	c.OnUIThread(func() error {
		return reviewIgnoredRebaseStart(c, git, start, finish, riskErr, false)
	})
	return nil
}

func reviewIgnoredRebaseStart(
	c *ControllerCommon,
	git *commands.GitCommand,
	start func(rebase *git_commands.RebaseCommands) error,
	finish func(err error) error,
	riskErr *git_commands.IgnoredRebaseRiskError,
	changed bool,
) error {
	if c.Git() != git {
		return errors.New(c.Tr.SelectedRebaseRepositoryChanged)
	}
	if changed {
		// As the selected-commit rebase does: the fresh list follows a notice that
		// previous approvals no longer apply.
		c.Confirm(types.ConfirmOpts{
			Title:  c.Tr.SelectedRebaseChangedTitle,
			Prompt: c.Tr.RebaseIgnoredChangedNote,
			HandleConfirm: func() error {
				return showIgnoredRebaseMenu(c, git, start, finish, riskErr)
			},
		})
		return nil
	}
	return showIgnoredRebaseMenu(c, git, start, finish, riskErr)
}

func showIgnoredRebaseMenu(
	c *ControllerCommon,
	git *commands.GitCommand,
	start func(rebase *git_commands.RebaseCommands) error,
	finish func(err error) error,
	riskErr *git_commands.IgnoredRebaseRiskError,
) error {
	// The menu itself is shared with the hard reset review (helpers.ShowIgnoredRiskMenu).
	return helpers.ShowIgnoredRiskMenu(c.HelperCommon, helpers.IgnoredRiskReview{
		CommandPrefix: "rebase",
		Title:         c.Tr.SelectedRebaseIgnoredTitle,
		Prompt:        c.Tr.SelectedRebaseIgnoredPrompt,
		ApproveLabel:  c.Tr.SelectedRebaseApproveIgnored,
		Paths:         riskErr.Paths,
		Approve: func() error {
			return approveIgnoredRebaseStart(c, git, start, finish, riskErr.Fingerprint)
		},
		Reopen: func() error { return showIgnoredRebaseMenu(c, git, start, finish, riskErr) },
	})
}

// approveIgnoredRebaseStart runs the start again under the approved fingerprint. The
// backend recomputes the risks, so a changed list comes back as a new refusal and
// is reviewed again.
func approveIgnoredRebaseStart(
	c *ControllerCommon,
	git *commands.GitCommand,
	start func(rebase *git_commands.RebaseCommands) error,
	finish func(err error) error,
	fingerprint string,
) error {
	return c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.SelectedRebaseApplying}, func(gocui.Task) error {
		err := git.Rebase.ApproveIgnoredRebaseRisks(fingerprint, func() error { return start(git.Rebase) })
		var riskErr *git_commands.IgnoredRebaseRiskError
		if errors.As(err, &riskErr) {
			c.OnUIThread(func() error {
				return reviewIgnoredRebaseStart(c, git, start, finish, riskErr, true)
			})
			return nil
		}
		return finishIgnoredRebaseStart(c, finish, err)
	})
}

// finishIgnoredRebaseStart runs a caller's finish on the outcome of a start. The
// backend's sentinel refusals reach the user as fork strings, mapped here, so
// every wrapped caller shows the same wording.
func finishIgnoredRebaseStart(c *ControllerCommon, finish func(err error) error, err error) error {
	return mapIgnoredRebaseError(c.Tr, finish(err))
}

func mapIgnoredRebaseError(tr *i18n.TranslationSet, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, git_commands.ErrRebaseRiskAfterCommit):
		return errors.New(tr.RebaseRiskAfterCommit)
	case errors.Is(err, git_commands.ErrRebaseIgnoredApprovalRequired):
		return errors.New(tr.RebaseIgnoredApprovalRequired)
	}
	return err
}
