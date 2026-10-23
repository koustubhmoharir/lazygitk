package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (review *capturedRebaseReview) execute(plan git_commands.BranchActionRebasePlan) error {
	if err := review.check(); err != nil {
		return err
	}
	review.started = true
	repository := review.flow.repository
	c := repository.c
	c.LogAction(c.Tr.Actions.RebaseBranch)
	return c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.RebasingStatus}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		if _, err := repository.git.Rebase.BranchActionRebaseCommand(plan); err != nil {
			c.OnUIThread(func() error {
				if err := repository.checkRepository(); err != nil {
					return err
				}
				c.Refresh(capturedRebaseRefreshOptions(false))
				return nil
			})
			return err
		}
		c.OnUIThread(func() error { return review.runTerminal(plan) })
		return nil
	})
}

func (review *capturedRebaseReview) runTerminal(plan git_commands.BranchActionRebasePlan) error {
	flow := review.flow
	if err := flow.checkReview(); err != nil {
		return err
	}
	repository := flow.repository
	c := repository.c
	// Revalidate after the worker/UI boundary, immediately before terminal
	// launch. The command retains the reviewed commits and captured Git backend.
	cmd, err := repository.git.Rebase.BranchActionRebaseCommand(plan)
	if err != nil {
		c.Refresh(capturedRebaseRefreshOptions(false))
		return err
	}
	_, result := c.RunSubprocess(cmd)
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	if err := c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.RebasingStatus}, func(gocui.Task) error {
		return review.finish(result, true)
	}); err != nil {
		c.Refresh(capturedRebaseRefreshOptions(false))
		return errors.Join(result, err)
	}
	return nil
}

// Query only the captured backend on the worker. Presentation, operation flags,
// marked-base changes and refresh completion all recheck the UI generation.
func (review *capturedRebaseReview) finish(result error, initial bool) error {
	flow := review.flow
	repository := flow.repository
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	rebasing, inspectionErr := repository.git.Status.IsInRebase()
	var unmerged bool
	if inspectionErr == nil && result != nil && rebasing {
		unmerged, inspectionErr = repository.git.Rebase.HasUnmergedFiles()
	}
	result = errors.Join(result, inspectionErr)
	if result != nil && inspectionErr == nil && rebasing && !unmerged {
		result = fmt.Errorf("%s\n\n%w", repository.c.Tr.BranchActionRebasePausedError, result)
	}
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	repository.c.OnUIThread(func() error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		c := repository.c
		if inspectionErr == nil {
			c.State().GetRepoState().SetMergeOrRebaseStartedInLazygit(rebasing)
		}
		opts := capturedRebaseRefreshOptions(result == nil)
		opts.Then = func() error {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			if initial && (result == nil || (inspectionErr == nil && rebasing)) {
				// Never erase a newer marked base installed while this completion
				// was queued. The executed range was the captured value.
				if c.Modes().MarkedBaseCommit.GetHash() == flow.markedBase {
					c.Modes().MarkedBaseCommit.Reset()
					c.PostRefreshUpdate(c.Contexts().LocalCommits)
				}
				if review.choice.insertBreak && result == nil {
					c.Context().Push(c.Contexts().LocalCommits, types.OnFocusOpts{})
				}
			}
			if unmerged && inspectionErr == nil {
				return review.conflicts()
			}
			return nil
		}
		c.Refresh(opts)
		return nil
	})
	if unmerged && inspectionErr == nil {
		return nil // Present conflict choices after refreshing the actual state.
	}
	return result
}

func (review *capturedRebaseReview) conflicts() error {
	repository := review.flow.repository
	if err := repository.checkRepository(); err != nil {
		return err
	}
	c := repository.c
	abortStarted := false
	return c.Menu(types.CreateMenuOptions{
		Title: c.Tr.FoundConflictsTitle, HideCancel: true, OnCancel: repository.checkRepository,
		Items: []*types.MenuItem{
			{
				CommandID: "conflict.view", Verb: "view-conflicts", Label: c.Tr.ViewConflictsMenuItem,
				OnPress: func() error {
					if err := repository.checkRepository(); err != nil {
						return err
					}
					c.Context().Push(c.Contexts().Files, types.OnFocusOpts{})
					return nil
				},
			},
			{
				CommandID: "operation.abort", Verb: "abort", Label: fmt.Sprintf(c.Tr.AbortMenuItem, "rebase"), Keys: menuKey('a'),
				OnPress: func() error {
					if err := repository.checkRepository(); err != nil {
						return err
					}
					if abortStarted {
						return git_commands.ErrBranchActionChanged
					}
					abortStarted = true
					return c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.RebasingStatus}, func(gocui.Task) error {
						if err := repository.checkRepository(); err != nil {
							return err
						}
						if err := repository.git.Rebase.CheckBranchActionRebaseInProgress(review.plan); err != nil {
							return err
						}
						cmd := repository.git.Rebase.GenericMergeOrRebaseActionCmdObj("rebase", "abort").SetWd(repository.git.RepoPaths.WorktreePath())
						if err := repository.checkRepository(); err != nil {
							return err
						}
						return review.finish(cmd.Run(), false)
					})
				},
			},
		},
	})
}
