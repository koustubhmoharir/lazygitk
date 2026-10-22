package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// runReviewedIntegration executes an already reviewed, pinned integration.
// The native command runs through the UI terminal so interactive rebase and
// signing can use the terminal lifecycle. It never retries or changes the plan.
func (target *branchSyncTarget) runReviewedIntegration(
	plan git_commands.CurrentPullIntegrationPlan,
	onReviewRequired func(error) error,
	afterExecution func(error) error,
) error {
	return target.runReviewedIntegrationReporting(plan, onReviewRequired, afterExecution, nil)
}

// reportFailure adds workflow progress to errors at asynchronous boundaries.
// It must be idempotent: synchronous worker dispatch can surface an already
// reported error through the outer dispatch result as well.
func (target *branchSyncTarget) runReviewedIntegrationReporting(
	plan git_commands.CurrentPullIntegrationPlan,
	onReviewRequired func(error) error,
	afterExecution func(error) error,
	reportFailure func(error) error,
) error {
	report := func(err error) error {
		if err != nil && reportFailure != nil {
			return reportFailure(err)
		}
		return err
	}
	if err := target.checkRepository(); err != nil {
		return report(err)
	}
	if plan.Name() != target.source.name {
		return report(fmt.Errorf("integration plan for branch %q cannot run against captured branch %q", plan.Name(), target.source.name))
	}
	item := syncOperationItem(target.git, target.source.name)
	ordinary := &models.Branch{Name: target.source.name}
	if target.c.State().GetItemOperation(item) != types.ItemOperationNone || target.c.State().GetItemOperation(ordinary) != types.ItemOperationNone {
		return report(errors.New(target.c.Tr.CantPullOrPushSameBranchTwice))
	}
	target.c.State().SetItemOperation(item, types.ItemOperationPulling)
	active := true
	clear := func() {
		if active {
			target.c.State().ClearItemOperation(item)
			active = false
		}
	}
	refreshFromWorker := func() {
		if target.checkRepository() == nil {
			target.c.RefreshFromWorker(types.RefreshOptions{})
		}
	}
	refreshOnUI := func() {
		if target.checkRepository() == nil {
			target.c.Refresh(types.RefreshOptions{})
		}
	}
	finishWorker := func(err error) error {
		clear()
		refreshFromWorker()
		return report(err)
	}
	finishUI := func(err error) error {
		clear()
		refreshOnUI()
		return report(err)
	}
	reviewOnUI := func(reviewErr error) error {
		clear()
		refreshOnUI()
		if onReviewRequired == nil {
			return report(reviewErr)
		}
		if err := target.checkRepository(); err != nil {
			return report(errors.Join(reviewErr, err))
		}
		return report(onReviewRequired(reviewErr))
	}
	err := target.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: fmt.Sprintf(target.c.Tr.BranchSyncIntegrationStatus, plan.Name())}, func(task gocui.Task) error {
		if err := target.checkRepository(); err != nil {
			return finishWorker(err)
		}
		_, err := target.git.Sync.CurrentPullIntegrationCommand(plan)
		if err != nil {
			if errors.Is(err, git_commands.ErrBranchSyncChanged) || errors.Is(err, git_commands.ErrPullIgnoredApprovalRequired) {
				if onReviewRequired == nil {
					return finishWorker(err)
				}
				clear()
				target.c.OnUIThread(func() error {
					return reviewOnUI(err)
				})
				return nil
			}
			return finishWorker(err)
		}
		target.c.OnUIThread(func() error {
			if err := target.checkRepository(); err != nil {
				return finishUI(err)
			}
			// Rebuild and validate the command on the UI thread immediately before
			// terminal launch. HEAD and reviewed inputs can change while callbacks queue.
			cmd, err := target.git.Sync.CurrentPullIntegrationCommand(plan)
			if err != nil {
				if errors.Is(err, git_commands.ErrBranchSyncChanged) || errors.Is(err, git_commands.ErrPullIgnoredApprovalRequired) {
					return reviewOnUI(err)
				}
				return finishUI(err)
			}
			_, runErr := target.c.RunSubprocess(cmd)
			nativeErr := runErr
			dispatchErr := target.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: fmt.Sprintf(target.c.Tr.BranchSyncIntegrationStatus, plan.Name())}, func(gocui.Task) error {
				if err := target.checkRepository(); err != nil {
					return finishWorker(errors.Join(nativeErr, err))
				}
				if afterExecution != nil {
					runErr = afterExecution(runErr)
				}
				if err := target.checkRepository(); err != nil {
					return finishWorker(errors.Join(nativeErr, runErr, err))
				}
				return finishWorker(runErr)
			})
			if dispatchErr != nil {
				return finishUI(errors.Join(runErr, dispatchErr))
			}
			return nil
		})
		return nil
	})
	if err != nil {
		clear()
		refreshOnUI()
	}
	return report(err)
}
