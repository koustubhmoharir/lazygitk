package helpers

import (
	"errors"
	"fmt"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (review *capturedBranchDeleteReview) execute() error {
	flow := review.flow
	if err := flow.check(); err != nil {
		return err
	}
	flow.started = true
	repository := flow.repository
	c := repository.c
	// Credential prompts need keyboard input while a remote push is running.
	// Keep the worker responsive and rely on captured-target checks.
	return c.WithWaitingStatus(c.Tr.DeletingStatus, func(task gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		if err := repository.git.Branch.CheckBranchActionDeleteContext(flow.context); err != nil {
			return review.finish(err)
		}
		if review.mode == capturedDeleteRemote || review.mode == capturedDeleteBoth {
			c.LogAction(c.Tr.Actions.DeleteRemoteBranch)
			if err := repository.git.Sync.RunBranchActionRemoteDelete(review.remotePlan, task, repository.checkRepository); err != nil {
				return review.finish(err)
			}
		}
		if review.mode == capturedDeleteLocal || review.mode == capturedDeleteBoth {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			// Remote pushes may have intentionally removed tracking-cache refs.
			// The compound plan excluded those from its retained-history proofs;
			// its local targets, HEAD, config and other proofs remain bound.
			cmd, err := repository.git.Branch.BranchActionLocalDeleteCommand(review.localPlan)
			if err != nil {
				return review.finish(err)
			}
			c.LogAction(c.Tr.Actions.DeleteLocalBranch)
			for _, source := range review.localPlan.Sources() {
				c.LogCommand(fmt.Sprintf("Deleting branch %q at %s", source.Info.Name, source.Target.SourceOID()), false)
			}
			if err := repository.checkRepository(); err != nil {
				return err
			}
			return review.finish(cmd.Run())
		}
		return review.finish(nil)
	})
}

func (review *capturedBranchDeleteReview) finish(result error) error {
	flow := review.flow
	repository := flow.repository
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	// Do not revalidate deleted source refs after execution. Refresh actual
	// state on native errors too, including earlier successful remote steps.
	repository.c.OnUIThread(func() error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		// Only collapse the selection that launched the deletion. Navigation
		// during credential prompts must not reset a later selection. Remote-
		// only deletion from local branches keeps its local range unchanged.
		if result == nil && (!flow.local || review.mode != capturedDeleteRemote) && flow.selectionUnchanged() {
			if flow.local {
				repository.c.Contexts().Branches.CollapseRangeSelectionToTop()
			} else {
				repository.c.Contexts().RemoteBranches.CollapseRangeSelectionToTop()
			}
		}
		opts := types.RefreshOptions{
			Scope:          []types.RefreshableView{types.BRANCHES, types.REMOTES, types.FILES, types.COMMITS, types.SUB_COMMITS, types.REFLOG, types.WORKTREES},
			BatchUIUpdates: true, BranchSelection: types.KeepBranchSelectionByName, CommitSelection: types.KeepCommitSelectionByHash,
			Then: func() error {
				if err := repository.checkRepository(); err != nil {
					return err
				}

				return nil
			},
		}
		repository.c.Refresh(opts)
		return nil
	})
	return result
}

func (flow *capturedBranchDeleteFlow) selectionUnchanged() bool {
	var refs []string
	if flow.local {
		branches, _, _ := flow.repository.c.Contexts().Branches.GetSelectedItems()
		for _, branch := range branches {
			refs = append(refs, branch.FullRefName())
		}
	} else {
		branches, _, _ := flow.repository.c.Contexts().RemoteBranches.GetSelectedItems()
		for _, branch := range branches {
			refs = append(refs, branch.FullRefName())
		}
	}
	targets := flow.context.Targets()
	return len(refs) == len(targets) && slices.EqualFunc(refs, targets, func(ref string, target git_commands.BranchActionTarget) bool {
		return ref == target.SourceRef()
	})
}
