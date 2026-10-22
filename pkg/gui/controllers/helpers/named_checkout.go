package helpers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func (self *RefsHelper) namedCheckoutItem(oid string) *types.MenuItem {
	flow := &namedCheckoutFlow{c: self.c, git: self.c.Git(), oid: oid}
	var disabled *types.DisabledReason
	if !git_commands.IsFullObjectID(oid) {
		disabled = &types.DisabledReason{Text: self.c.Tr.NamedCheckoutSelectCommit}
	}
	return &types.MenuItem{
		CommandID: "branch.checkout-at-selected-commit", Verb: "checkout",
		Label: self.c.Tr.CheckoutAsBranch, Tooltip: self.c.Tr.NamedCheckoutTooltip,
		Keys: menuKey('b'), OpensMenu: true, DisabledReason: disabled,
		OnPress: flow.begin,
	}
}

// Capture both the source and repository before the checkout menu opens. Never
// read a later selection or redirect a pending prompt to a different repository.
type namedCheckoutFlow struct {
	c   *HelperCommon
	git *commands.GitCommand
	oid string
}

func (flow *namedCheckoutFlow) checkRepository() error {
	if flow.c.Git() != flow.git {
		return errors.New(flow.c.Tr.NamedCheckoutRepositoryChanged)
	}
	return nil
}

func (flow *namedCheckoutFlow) begin() error {
	if !git_commands.IsFullObjectID(flow.oid) {
		return errors.New(flow.c.Tr.NamedCheckoutSelectCommit)
	}
	if err := flow.checkRepository(); err != nil {
		return err
	}
	flow.c.Prompt(types.PromptOpts{
		Title:               fmt.Sprintf(flow.c.Tr.NamedCheckoutNamePrompt, utils.ShortHash(flow.oid)),
		FindSuggestionsFunc: NewSuggestionsHelper(flow.c).GetBranchNameSuggestionsFunc(),
		PreserveWhitespace:  true,
		AllowEditSuggestion: true,
		HandleConfirm:       func(name string) error { return flow.prepare(name, false) },
	})
	return nil
}

func (flow *namedCheckoutFlow) prepare(name string, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	return flow.c.WithWaitingStatus(flow.c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		plan, err := flow.git.Branch.PrepareNamedCheckout(name, flow.oid)
		if err != nil {
			return err
		}
		flow.c.OnUIThread(func() error { return flow.present(plan, changed) })
		return nil
	})
}

func (flow *namedCheckoutFlow) present(plan git_commands.NamedCheckoutPlan, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	if !plan.MovesBranch() && !changed {
		// Accepting the literal name requested this create/attach explicitly.
		return flow.apply(plan)
	}
	title := flow.c.Tr.NamedCheckoutMoveTitle
	oldTip, newTip := namedCheckoutTipLabels(plan.OldOID(), plan.TargetOID())
	prompt := fmt.Sprintf(flow.c.Tr.NamedCheckoutMovePrompt, plan.Name(), oldTip, newTip)
	if !plan.DestinationExists() {
		prompt = fmt.Sprintf(flow.c.Tr.NamedCheckoutCreatePrompt, plan.Name(), newTip)
	} else if !plan.MovesBranch() {
		prompt = fmt.Sprintf(flow.c.Tr.NamedCheckoutAttachPrompt, plan.Name(), newTip)
	}
	if changed {
		title = flow.c.Tr.NamedCheckoutChangedTitle
		prompt = flow.c.Tr.NamedCheckoutChangedNote + "\n\n" + prompt
	}
	head := "HEAD"
	if plan.HeadRef() != "" {
		head = fmt.Sprintf("%q", strings.TrimPrefix(plan.HeadRef(), "refs/heads/"))
	}
	if plan.HeadOID() != "" {
		head += " (" + utils.ShortHash(plan.HeadOID()) + ")"
	} else {
		head += " (" + flow.c.Tr.NoCommitsThisBranch + ")"
	}
	prompt += "\n\n" + fmt.Sprintf(flow.c.Tr.NamedCheckoutCurrentStatePrompt, head)
	flow.c.Confirm(types.ConfirmOpts{
		Title: title, Prompt: prompt,
		HandleConfirm: func() error { return flow.apply(plan) },
	})
	return nil
}

func (flow *namedCheckoutFlow) apply(plan git_commands.NamedCheckoutPlan) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	flow.c.LogAction(fmt.Sprintf("%s: %q → %s", flow.c.Tr.NamedCheckoutApplyingStatus, plan.Name(), utils.ShortHash(plan.TargetOID())))
	// Fork: unlike the upstream checkout helper, this flow never offers stash
	// or force. Buffer subsequent input until the changed checkout is refreshed.
	return flow.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: flow.c.Tr.NamedCheckoutApplyingStatus}, func(gocui.Task) error {
		err := flow.git.Branch.ApplyNamedCheckout(plan)
		if errors.Is(err, git_commands.ErrNamedCheckoutChanged) {
			// Prepare on this worker so the fresh warning is queued before the
			// blocking-status helper releases buffered input. Starting a second
			// worker from the UI would briefly replay keys against the old pane.
			fresh, prepareErr := flow.git.Branch.PrepareNamedCheckout(plan.Name(), flow.oid)
			if prepareErr != nil {
				flow.c.RefreshFromWorker(namedCheckoutFailureRefreshOptions())
				return prepareErr
			}
			flow.c.OnUIThread(func() error { return flow.present(fresh, true) })
			return nil
		}
		opts := namedCheckoutRefreshOptions()
		if err == nil {
			opts.Then = func() error {
				if flow.c.Context().Current() != flow.c.Contexts().Branches {
					flow.c.Context().Push(flow.c.Contexts().Branches, types.OnFocusOpts{})
				}
				return nil
			}
		} else {
			opts = namedCheckoutFailureRefreshOptions()
		}
		// A failed Git command can already have changed HEAD, index or files.
		// Refresh its actual result, preserve the failure, and never retry it.
		flow.c.RefreshFromWorker(opts)
		return err
	})
}

func namedCheckoutRefreshOptions() types.RefreshOptions {
	return types.RefreshOptions{
		Scope: []types.RefreshableView{
			types.BRANCHES, types.COMMITS, types.SUB_COMMITS, types.FILES,
			types.REFLOG, types.WORKTREES, types.BISECT_INFO,
		},
		BatchUIUpdates: true, BranchSelection: types.SelectCheckedOutBranch,
		CommitSelection: types.SelectHeadCommit, SelectTopReflogCommit: true,
	}
}

func namedCheckoutFailureRefreshOptions() types.RefreshOptions {
	opts := namedCheckoutRefreshOptions()
	// Refresh HEAD's marker without discarding the target the user is inspecting.
	// Ordinary rejection has no checkout/reflog entry to jump to; a hook failure
	// may have completed checkout, but still must not silently navigate the user.
	opts.CommitSelection = types.KeepCommitSelectionByHash
	opts.SelectTopReflogCommit = false
	return opts
}

// Keep old/new abbreviations distinguishable when commit IDs share a prefix.
func namedCheckoutTipLabels(oldOID, newOID string) (string, string) {
	if oldOID == newOID {
		return utils.ShortHash(oldOID), utils.ShortHash(newOID)
	}
	length := min(7, len(oldOID), len(newOID))
	for length < len(oldOID) && length < len(newOID) && oldOID[:length] == newOID[:length] {
		length++
	}
	if length == 0 {
		return utils.ShortHash(oldOID), utils.ShortHash(newOID)
	}
	return oldOID[:length], newOID[:length]
}
