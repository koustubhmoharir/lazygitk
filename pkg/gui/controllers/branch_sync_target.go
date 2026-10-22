package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchSyncTarget struct {
	c          *ControllerCommon
	git        *commands.GitCommand
	generation int
	source     branchSyncSource
}

func newBranchSyncTarget(c *ControllerCommon, source branchSyncSource) *branchSyncTarget {
	return &branchSyncTarget{c: c, git: c.Git(), generation: c.State().GetRepoGeneration(), source: source}
}

func (target *branchSyncTarget) checkRepository() error {
	if target.c.Git() != target.git || target.c.State().GetRepoGeneration() != target.generation {
		return errors.New(target.c.Tr.BranchSyncRepositoryChanged)
	}
	return nil
}

func (target *branchSyncTarget) inspect(callback func(git_commands.BranchSyncInfo) error) error {
	if err := target.checkRepository(); err != nil {
		return err
	}
	return target.c.WithWaitingStatus(target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := target.checkRepository(); err != nil {
			return err
		}
		info, err := target.git.Sync.InspectBranchSync(target.source.name)
		if err != nil {
			return err
		}
		if !target.source.selected && info.HeadRef != info.FullRef {
			return fmt.Errorf(target.c.Tr.BranchSyncNoLongerCurrent, target.source.name)
		}
		target.c.OnUIThread(func() error {
			if err := target.checkRepository(); err != nil {
				return err
			}
			return callback(info)
		})
		return nil
	})
}

type branchSyncOperationItem struct{ urn string }

func (item branchSyncOperationItem) URN() string { return item.urn }

func syncOperationItem(git *commands.GitCommand, branchName string) types.HasUrn {
	return branchSyncOperationItem{urn: "fork:branch-sync:" + git.RepoPaths.WorktreePath() + "\x00" + branchName}
}

func (target *branchSyncTarget) withOperation(operation types.ItemOperation, message string, run func(gocui.Task) error) error {
	if err := target.checkRepository(); err != nil {
		return err
	}
	item := syncOperationItem(target.git, target.source.name)
	ordinary := &models.Branch{Name: target.source.name}
	if target.c.State().GetItemOperation(item) != types.ItemOperationNone || target.c.State().GetItemOperation(ordinary) != types.ItemOperationNone {
		return errors.New(target.c.Tr.CantPullOrPushSameBranchTwice)
	}
	// Mark synchronously on the UI thread, before the worker is dispatched.
	target.c.State().SetItemOperation(item, operation)
	status := fmt.Sprintf(message, target.source.name)
	err := target.c.WithWaitingStatus(status, func(task gocui.Task) (err error) {
		defer target.c.State().ClearItemOperation(item)
		defer func() {
			if target.c.Git() == target.git && target.c.State().GetRepoGeneration() == target.generation {
				target.c.RefreshFromWorker(types.RefreshOptions{})
			}
		}()
		if err := target.checkRepository(); err != nil {
			return err
		}
		return run(task)
	})
	if err != nil {
		target.c.State().ClearItemOperation(item)
	}
	return err
}
