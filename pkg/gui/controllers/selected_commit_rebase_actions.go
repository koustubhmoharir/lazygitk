package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func (self *BasicCommitsController) selectedCommitRebaseBinding(opts types.KeybindingsOpts) *types.Binding {
	return &types.Binding{
		CommandID: "branch.rebase-current-onto-selected-commit", Verb: "rebase",
		Keys:    opts.GetKeys(opts.Config.Commits.RebaseOntoSelectedCommit),
		Handler: self.beginSelectedCommitRebase,
		GetDisabledReason: func() *types.DisabledReason {
			_, reason := self.selectedRebaseSource()
			return reason
		},
		Description: self.c.Tr.RebaseOntoSelectedCommit,
		Tooltip:     self.c.Tr.SelectedRebaseTooltip, OpensMenu: true,
	}
}

func (self *BasicCommitsController) selectedRebaseSource() (string, *types.DisabledReason) {
	selected, start, end := self.context.GetSelectedItems()
	if len(selected) != 1 || start != end || selected[0] == nil || !isFullSelectedCommitOID(selected[0].Hash()) {
		return "", &types.DisabledReason{Text: self.c.Tr.SelectedRebaseSelectCommit}
	}
	return selected[0].Hash(), nil
}

func (self *BasicCommitsController) beginSelectedCommitRebase() error {
	oid, reason := self.selectedRebaseSource()
	if reason != nil {
		return disabledReasonError(reason)
	}
	flow := &selectedCommitRebaseFlow{c: self.c, git: self.c.Git(), oid: oid}
	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.RebaseOntoSelectedCommit,
		Items: []*types.MenuItem{
			{
				CommandID: "branch.rebase-current-onto-selected-commit.ordinary", Verb: "rebase",
				Label: self.c.Tr.SelectedRebaseOrdinary, Keys: menuKey('r'), OpensMenu: true,
				OnPress: func() error { return flow.prepare(false) },
			},
			{
				CommandID: "branch.rebase-current-onto-selected-commit.preserve-merges", Verb: "rebase",
				Label: self.c.Tr.SelectedRebasePreserveMerges, Keys: menuKey('m'), OpensMenu: true,
				OnPress: func() error { return flow.prepare(true) },
			},
		},
	})
}

// Keep source and repository captured across topology, history and risk modals.
// Never read a later commit selection or the marked-base mode in this flow.
type selectedCommitRebaseFlow struct {
	c   *ControllerCommon
	git *commands.GitCommand
	oid string
}

func (flow *selectedCommitRebaseFlow) checkRepository() error {
	if flow.c.Git() != flow.git {
		return errors.New(flow.c.Tr.SelectedRebaseRepositoryChanged)
	}
	return nil
}

func (flow *selectedCommitRebaseFlow) prepare(preserveMerges bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	return flow.c.WithWaitingStatus(flow.c.Tr.SelectedRebaseChecking, func(gocui.Task) error {
		plan, err := flow.git.Rebase.PrepareSelectedCommitRebase(flow.oid, preserveMerges)
		if err != nil {
			return err
		}
		flow.c.OnUIThread(func() error { return flow.present(plan, false) })
		return nil
	})
}

func (flow *selectedCommitRebaseFlow) present(plan git_commands.SelectedCommitRebasePlan, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	topology := flow.c.Tr.SelectedRebaseOrdinary
	if plan.PreserveMerges() {
		topology = flow.c.Tr.SelectedRebasePreserveMerges
	}
	prompt := fmt.Sprintf(flow.c.Tr.SelectedRebaseConfirmPrompt,
		strings.TrimPrefix(plan.HeadRef(), "refs/heads/"), utils.ShortHash(plan.HeadOID()), plan.TargetOID(), topology)
	title := flow.c.Tr.SelectedRebaseConfirmTitle
	if changed {
		title = flow.c.Tr.SelectedRebaseChangedTitle
		prompt = flow.c.Tr.SelectedRebaseChangedNote + "\n\n" + prompt
	}
	flow.c.Confirm(types.ConfirmOpts{
		Title: title, Prompt: prompt,
		HandleConfirm: func() error { return flow.reviewIgnoredFiles(plan) },
	})
	return nil
}

func (flow *selectedCommitRebaseFlow) reviewIgnoredFiles(plan git_commands.SelectedCommitRebasePlan) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	paths := plan.IgnoredFiles()
	if len(paths) == 0 {
		return flow.apply(plan)
	}
	items := []*types.MenuItem{
		{
			CommandID: "branch.rebase-current-onto-selected-commit.cancel-risk", Verb: "cancel",
			Label: flow.c.Tr.Cancel, OnPress: func() error { return nil },
		},
		{
			CommandID: "branch.rebase-current-onto-selected-commit.approve-ignored-overwrite", Verb: "rebase",
			Label: flow.c.Tr.SelectedRebaseApproveIgnored, Keys: menuKey('p'),
			OnPress: func() error { return flow.apply(plan.WithIgnoredFilesApproved()) },
		},
	}
	// File rows, rather than an unbounded prompt header, keep the full risk
	// list scrollable. Enter on a file only reopens review, never approves it.
	for _, path := range paths {
		quoted := fmt.Sprintf("%q", path)
		items = append(items, &types.MenuItem{
			CommandID: "branch.rebase-current-onto-selected-commit.inspect-ignored-path", Verb: "inspect",
			Label: quoted, Tooltip: quoted, OpensMenu: true,
			OnPress: func() error { return flow.reviewIgnoredFiles(plan) },
		})
	}
	return flow.c.Menu(types.CreateMenuOptions{
		Title:      flow.c.Tr.SelectedRebaseIgnoredTitle,
		Prompt:     flow.c.Tr.SelectedRebaseIgnoredPrompt,
		HideCancel: true, Items: items,
	})
}

func (flow *selectedCommitRebaseFlow) apply(plan git_commands.SelectedCommitRebasePlan) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	signing := flow.git.Config.NeedsGpgSubprocessForCommit()
	helper := flow.c.Helpers().MergeAndRebase
	return flow.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: flow.c.Tr.SelectedRebaseApplying}, func(gocui.Task) error {
		cmd, err := flow.git.Rebase.SelectedCommitRebaseCommand(plan)
		if errors.Is(err, git_commands.ErrSelectedCommitRebaseChanged) {
			fresh, prepareErr := flow.git.Rebase.PrepareSelectedCommitRebase(flow.oid, plan.PreserveMerges())
			if prepareErr != nil {
				flow.c.RefreshFromWorker(selectedRebaseRefreshOptions(false))
				return prepareErr
			}
			// Queue fresh review before buffered input can replay against old state.
			flow.c.OnUIThread(func() error { return flow.present(fresh, true) })
			return nil
		}
		if err != nil {
			flow.c.RefreshFromWorker(selectedRebaseRefreshOptions(false))
			return err
		}
		if signing {
			flow.c.OnUIThread(func() error { return flow.runTerminal(cmd) })
			return nil
		}
		result := cmd.Run()
		return helper.CheckMergeOrRebaseWithRefreshOptions(result, selectedRebaseRefreshOptions(result == nil))
	})
}

func (flow *selectedCommitRebaseFlow) runTerminal(cmd *oscommands.CmdObj) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	_, result := flow.c.RunSubprocess(cmd)
	helper := flow.c.Helpers().MergeAndRebase
	return flow.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: flow.c.Tr.SelectedRebaseApplying}, func(gocui.Task) error {
		return helper.CheckRebaseSubprocessResult(result, selectedRebaseRefreshOptions(result == nil))
	})
}

func selectedRebaseRefreshOptions(success bool) types.RefreshOptions {
	selection := types.KeepCommitSelectionByHash
	if success {
		selection = types.SelectHeadCommit
	}
	return types.RefreshOptions{
		Scope:          []types.RefreshableView{types.BRANCHES, types.COMMITS, types.SUB_COMMITS, types.FILES, types.REFLOG, types.WORKTREES, types.BISECT_INFO},
		BatchUIUpdates: true, CommitSelection: selection,
		BranchSelection: types.KeepBranchSelectionByName,
	}
}
