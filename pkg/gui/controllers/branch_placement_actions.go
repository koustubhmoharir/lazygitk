package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func (self *BasicCommitsController) branchPlacementBinding(opts types.KeybindingsOpts) *types.Binding {
	return &types.Binding{
		CommandID: "branch.place-at-selected-commit", Verb: "place-branch",
		Keys:    opts.GetKeys(opts.Config.Commits.CreateMoveBranch),
		Handler: self.beginBranchPlacement,
		GetDisabledReason: func() *types.DisabledReason {
			_, reason := self.branchPlacementSource()
			return reason
		},
		Description: self.c.Tr.CreateMoveBranchHere,
		Tooltip:     self.c.Tr.PlaceBranchTooltip,
		OpensMenu:   true,
	}
}

func (self *BasicCommitsController) branchPlacementSource() (string, *types.DisabledReason) {
	selected, start, end := self.context.GetSelectedItems()
	if len(selected) != 1 || start != end || selected[0] == nil || !isFullSelectedCommitOID(selected[0].Hash()) {
		return "", &types.DisabledReason{Text: self.c.Tr.PlaceBranchSelectCommit}
	}
	return selected[0].Hash(), nil
}

func (self *BasicCommitsController) beginBranchPlacement() error {
	oid, reason := self.branchPlacementSource()
	if reason != nil {
		return disabledReasonError(reason)
	}
	flow := &branchPlacementFlow{c: self.c, git: self.c.Git(), oid: oid}
	self.c.Prompt(types.PromptOpts{
		Title:               fmt.Sprintf(self.c.Tr.PlaceBranchNamePrompt, utils.ShortHash(oid)),
		FindSuggestionsFunc: self.c.Helpers().Suggestions.GetBranchNameSuggestionsFunc(),
		PreserveWhitespace:  true,
		AllowEditSuggestion: true,
		HandleConfirm:       func(name string) error { return flow.prepare(name, false) },
	})
	return nil
}

// A flow captures repository identity and source before any prompt/worker.
// No callback reads a later selected commit or silently redirects to a new repo.
type branchPlacementFlow struct {
	c   *ControllerCommon
	git *commands.GitCommand
	oid string
}

func (flow *branchPlacementFlow) checkRepository() error {
	if flow.c.Git() != flow.git {
		return errors.New(flow.c.Tr.PlaceBranchRepositoryChanged)
	}
	return nil
}

func (flow *branchPlacementFlow) prepare(name string, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	return flow.c.WithWaitingStatus(flow.c.Tr.PlaceBranchWorkingStatus, func(gocui.Task) error {
		plan, err := flow.git.Branch.PrepareBranchPlacement(name, flow.oid)
		if err != nil {
			return err
		}
		flow.c.OnUIThread(func() error { return flow.present(plan, changed) })
		return nil
	})
}

func (flow *branchPlacementFlow) present(plan git_commands.BranchPlacementPlan, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	if plan.Action() == git_commands.BranchPlacementNoop {
		flow.c.Refresh(branchPlacementRefreshOptions())
		flow.c.Alert(flow.c.Tr.PlaceBranchAlreadyHereTitle,
			fmt.Sprintf(flow.c.Tr.PlaceBranchAlreadyHerePrompt, plan.Name(), utils.ShortHash(plan.TargetOID())))
		return nil
	}
	if plan.Action() != git_commands.BranchPlacementCreate && plan.Action() != git_commands.BranchPlacementMove {
		return fmt.Errorf("invalid branch placement plan")
	}
	if plan.Action() == git_commands.BranchPlacementCreate && !changed {
		return flow.apply(plan) // the name prompt explicitly requested this creation
	}
	title := flow.c.Tr.PlaceBranchMoveTitle
	oldTip, newTip := placementTipLabels(plan.OldOID(), plan.TargetOID())
	prompt := fmt.Sprintf(flow.c.Tr.PlaceBranchMovePrompt, plan.Name(), oldTip, newTip)
	if plan.Action() == git_commands.BranchPlacementCreate {
		prompt = fmt.Sprintf(flow.c.Tr.PlaceBranchCreateAgainPrompt, plan.Name(), utils.ShortHash(plan.TargetOID()))
	}
	if changed {
		title = flow.c.Tr.PlaceBranchChangedTitle
		prompt = flow.c.Tr.PlaceBranchChangedNote + "\n\n" + prompt
	}
	flow.c.Confirm(types.ConfirmOpts{
		Title: title, Prompt: prompt,
		HandleConfirm: func() error { return flow.apply(plan) },
	})
	return nil
}

// Extend a shared prefix so a move warning never displays identical old/new
// abbreviations for different tips. No Git calls belong in modal rendering.
func placementTipLabels(oldOID, newOID string) (string, string) {
	length := min(7, len(oldOID), len(newOID))
	for length < len(oldOID) && length < len(newOID) && oldOID[:length] == newOID[:length] {
		length++
	}
	if length == 0 {
		return utils.ShortHash(oldOID), utils.ShortHash(newOID)
	}
	return oldOID[:length], newOID[:length]
}

func (flow *branchPlacementFlow) apply(plan git_commands.BranchPlacementPlan) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	flow.c.LogAction(fmt.Sprintf("%s: %q → %s", flow.c.Tr.PlaceBranchApplyingStatus, plan.Name(), utils.ShortHash(plan.TargetOID())))
	return flow.c.WithWaitingStatus(flow.c.Tr.PlaceBranchApplyingStatus, func(gocui.Task) error {
		err := flow.git.Branch.ApplyBranchPlacement(plan)
		if errors.Is(err, git_commands.ErrBranchPlacementChanged) {
			flow.c.OnUIThread(func() error { return flow.prepare(plan.Name(), true) })
			return nil
		}
		// Refresh on failure too: a post-commit protocol failure reports
		// uncertainty, so the markers must reflect Git's actual current state.
		flow.c.RefreshFromWorker(branchPlacementRefreshOptions())
		return err
	})
}

func branchPlacementRefreshOptions() types.RefreshOptions {
	return types.RefreshOptions{
		Scope:          []types.RefreshableView{types.BRANCHES, types.COMMITS, types.SUB_COMMITS, types.REFLOG},
		BatchUIUpdates: true, CommitSelection: types.KeepCommitSelectionByHash,
		BranchSelection: types.KeepBranchSelectionByName,
	}
}
