package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// A wider review collects approvals without moving branch tips. Fetch retains
// its separate approval; only integration/ref updates wait for batch approval.
type branchSyncUpdateCollector struct {
	context       string
	acceptForward func(git_commands.BranchFastForwardPlan) error
	acceptCurrent func(git_commands.CurrentPullIntegrationPlan) error
}

func (flow *branchSyncUpdateFlow) collectionContext() string {
	if flow.collector == nil {
		return ""
	}
	return flow.collector.context
}

func (flow *branchSyncUpdateFlow) collectFastForward(plan git_commands.BranchFastForwardPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if flow.collector.acceptForward == nil {
		return errors.New("fast-forward review collector is not configured")
	}
	accept := flow.collector.acceptForward
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		if err := flow.target.git.Sync.CheckBranchFastForward(plan); err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			return accept(plan)
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) collectIntegration(plan git_commands.CurrentPullIntegrationPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if flow.collector.acceptCurrent == nil {
		return errors.New("current integration review collector is not configured")
	}
	accept := flow.collector.acceptCurrent
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		if _, err := flow.target.git.Sync.CurrentPullIntegrationCommand(plan); err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			return accept(plan)
		})
		return nil
	})
}
