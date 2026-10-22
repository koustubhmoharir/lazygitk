package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// renameCapturedBranch runs the selected-branch rename without consulting the
// branch list again. The name, repository instance, and repository generation
// captured here remain the target for every later prompt and callback.
func (self *BranchesController) renameCapturedBranch(name string) error {
	target := newBranchSyncTarget(self.c, branchSyncSource{name: name, selected: true})
	flow := &branchRenameFlow{controller: self, target: target}
	return target.inspect(func(info git_commands.BranchSyncInfo) error {
		flow.info = info
		if info.UpstreamRemote != "" || info.UpstreamRef != "" {
			self.c.Confirm(types.ConfirmOpts{
				Title:         self.c.Tr.RenameBranch,
				Prompt:        self.c.Tr.RenameBranchWarning,
				HandleConfirm: flow.promptName,
				HandleClose:   target.checkRepository,
			})
			return nil
		}
		return flow.promptName()
	})
}

type branchRenameFlow struct {
	controller *BranchesController
	target     *branchSyncTarget
	info       git_commands.BranchSyncInfo
}

func (flow *branchRenameFlow) promptName() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	flow.controller.c.Prompt(types.PromptOpts{
		Title:               flow.controller.c.Tr.NewBranchNamePrompt + " " + flow.info.Name + ":",
		InitialContent:      flow.info.Name,
		PreserveWhitespace:  true,
		AllowEditSuggestion: true,
		HandleClose:         flow.target.checkRepository,
		HandleConfirm:       flow.prepare,
	})
	return nil
}

func (flow *branchRenameFlow) prepare(newName string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.controller.c.WithWaitingStatus(flow.controller.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		plan, err := flow.target.git.Sync.PrepareBranchRename(flow.info.Name, flow.info.OID, newName)
		if err != nil {
			return err
		}
		flow.controller.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			if !plan.NeedsUpdate() {
				return nil
			}
			prompt := fmt.Sprintf(flow.controller.c.Tr.BranchRenameReviewPrompt, plan.OldName(), plan.NewName(), plan.SourceOID())
			flow.controller.c.Confirm(types.ConfirmOpts{
				Title:         flow.controller.c.Tr.RenameBranch,
				Prompt:        prompt,
				HandleClose:   flow.target.checkRepository,
				HandleConfirm: func() error { return flow.apply(plan) },
			})
			return nil
		})
		return nil
	})
}

func (flow *branchRenameFlow) apply(plan git_commands.BranchRenamePlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.controller.c.WithWaitingStatus(flow.controller.c.Tr.RenameBranch, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		flow.controller.c.LogAction(flow.controller.c.Tr.Actions.RenameBranch)
		err := flow.target.git.Sync.ApplyBranchRename(plan)
		if repositoryErr := flow.target.checkRepository(); repositoryErr != nil {
			return errors.Join(err, repositoryErr)
		}
		if err != nil {
			flow.controller.c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.BRANCHES, types.WORKTREES}})
			return err
		}
		flow.controller.c.RefreshFromWorker(types.RefreshOptions{
			Scope: []types.RefreshableView{types.BRANCHES, types.WORKTREES},
			Then: func() error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				for i, branch := range flow.controller.c.Model().Branches {
					if branch.Name == plan.NewName() {
						flow.controller.context().SetSelection(i)
						flow.controller.context().HandleRender()
						break
					}
				}
				return nil
			},
		})
		return nil
	})
}
