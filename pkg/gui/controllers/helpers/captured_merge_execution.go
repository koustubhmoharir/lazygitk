package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (review *capturedMergeReview) execute(plan git_commands.BranchActionMergePlan) error {
	if err := review.check(); err != nil {
		return err
	}
	review.started = true
	repository := review.flow.repository
	c := repository.c
	c.LogAction(c.Tr.Actions.Merge)
	return c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.MergingStatus}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		if _, err := repository.git.Branch.BranchActionMergeCommand(plan); err != nil {
			c.OnUIThread(func() error {
				if err := repository.checkRepository(); err != nil {
					return err
				}
				c.Refresh(capturedRebaseRefreshOptions(false))
				return nil
			})
			return err
		}
		c.OnUIThread(func() error { return review.runMergeTerminal(plan) })
		return nil
	})
}

func (review *capturedMergeReview) runMergeTerminal(plan git_commands.BranchActionMergePlan) error {
	repository := review.flow.repository
	if err := repository.checkRepository(); err != nil {
		return err
	}
	cmd, err := repository.git.Branch.BranchActionMergeCommand(plan)
	if err != nil {
		repository.c.Refresh(capturedRebaseRefreshOptions(false))
		return err
	}
	_, result := repository.c.RunSubprocess(cmd)
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	err = repository.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: repository.c.Tr.MergingStatus}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return errors.Join(result, err)
		}
		if result == nil && plan.Mode() == git_commands.BranchActionMergeSquashAndCommit {
			commit, needed, err := repository.git.Branch.PrepareBranchActionSquashCommit(plan, review.commitOptions)
			if err != nil {
				return review.finish(err)
			}
			if needed {
				repository.c.OnUIThread(func() error { return review.runCommitTerminal(commit) })
				return nil
			}
		}
		return review.finish(result)
	})
	if err != nil {
		repository.c.Refresh(capturedRebaseRefreshOptions(false))
		return errors.Join(result, err)
	}
	return nil
}

func (review *capturedMergeReview) runCommitTerminal(plan git_commands.BranchActionSquashCommitPlan) error {
	repository := review.flow.repository
	if err := repository.checkRepository(); err != nil {
		return err
	}
	cmd, err := repository.git.Branch.BranchActionSquashCommitCommand(plan)
	if err != nil {
		repository.c.Refresh(capturedRebaseRefreshOptions(false))
		return err
	}
	_, result := repository.c.RunSubprocess(cmd)
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	err = repository.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: repository.c.Tr.MergingStatus}, func(gocui.Task) error {
		return review.finish(result)
	})
	if err != nil {
		repository.c.Refresh(capturedRebaseRefreshOptions(false))
		return errors.Join(result, err)
	}
	return nil
}

// A conflict is not a successful squash. Native failures, including signing or
// hook failures after partial writes, always stop the automatic commit phase.
func (review *capturedMergeReview) finish(result error) error {
	repository := review.flow.repository
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	merging, inspectionErr := repository.git.Status.IsInMergeState()
	var unmerged bool
	if inspectionErr == nil {
		unmerged, inspectionErr = repository.git.Rebase.HasUnmergedFiles()
	}
	result = errors.Join(result, inspectionErr)
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	repository.c.OnUIThread(func() error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		c := repository.c
		if inspectionErr == nil {
			c.State().GetRepoState().SetMergeOrRebaseStartedInLazygit(merging)
		}
		opts := capturedRebaseRefreshOptions(result == nil)
		opts.Then = func() error {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			if unmerged && inspectionErr == nil {
				return review.conflicts(merging)
			}
			return nil
		}
		c.Refresh(opts)
		return nil
	})
	if unmerged && inspectionErr == nil {
		return nil // Show actual conflict state after refresh; do not commit.
	}
	return result
}

func (review *capturedMergeReview) conflicts(merging bool) error {
	repository := review.flow.repository
	if err := repository.checkRepository(); err != nil {
		return err
	}
	c := repository.c
	items := []*types.MenuItem{{
		CommandID: "conflict.view", Verb: "view-conflicts", Label: c.Tr.ViewConflictsMenuItem,
		OnPress: func() error {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			c.Context().Push(c.Contexts().Files, types.OnFocusOpts{})
			return nil
		},
	}}
	prompt := ""
	if merging {
		abortStarted := false
		items = append(items, &types.MenuItem{
			CommandID: "operation.abort", Verb: "abort", Label: fmt.Sprintf(c.Tr.AbortMenuItem, "merge"), Keys: menuKey('a'),
			OnPress: func() error {
				if err := repository.checkRepository(); err != nil {
					return err
				}
				if abortStarted {
					return git_commands.ErrBranchActionChanged
				}
				abortStarted = true
				return c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.MergingStatus}, func(gocui.Task) error {
					if err := repository.checkRepository(); err != nil {
						return err
					}
					if err := repository.git.Branch.CheckBranchActionMergeInProgress(review.plan); err != nil {
						return err
					}
					cmd := repository.git.Rebase.GenericMergeOrRebaseActionCmdObj("merge", "abort").
						SetWd(repository.git.RepoPaths.WorktreePath()).DisableLockRetries()
					if err := repository.checkRepository(); err != nil {
						return err
					}
					return review.finish(cmd.Run())
				})
			},
		})
	} else {
		prompt = c.Tr.BranchActionMergeSquashConflicts
	}
	return c.Menu(types.CreateMenuOptions{Title: c.Tr.FoundConflictsTitle, Prompt: prompt, HideCancel: true, Items: items, OnCancel: repository.checkRepository})
}
