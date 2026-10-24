package controllers

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (self *BasicCommitsController) combineSelectedBinding() *types.Binding {
	return &types.Binding{
		CommandID: "commit.combine-selected", Verb: "combine", Description: self.c.Tr.CombineSelected,
		Handler:   self.beginCombineSelected,
		OpensMenu: true,
		GetDisabledReason: func() *types.DisabledReason {
			if _, ok := forkCombineSelection(self.context); !ok {
				return &types.DisabledReason{Text: self.c.Tr.CombineSelectCommits, Inapplicable: true}
			}
			return nil
		},
	}
}

func forkCombineSelection(context ContainsCommits) ([]string, bool) {
	selected, _, _ := context.GetSelectedItems()
	if len(selected) < 2 {
		return nil, false
	}
	oids := make([]string, len(selected))
	for i, commit := range selected {
		if commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
			return nil, false
		}
		oids[i] = commit.Hash()
	}
	return oids, true
}

func (self *BasicCommitsController) beginCombineSelected() error {
	oids, ok := forkCombineSelection(self.context)
	if !ok {
		return errors.New(self.c.Tr.CombineSelectCommits)
	}
	flow := &forkCombineFlow{c: self.c, git: self.c.Git(), generation: self.c.State().GetRepoGeneration(), oids: slices.Clone(oids)}
	return flow.prepare(false)
}

type forkCombineFlow struct {
	c              *ControllerCommon
	git            *commands.GitCommand
	generation     int
	oids           []string
	draft          string
	editorPlan     *git_commands.CapturedCombinePlan
	started        bool
	reviewAccepted bool
}

func (flow *forkCombineFlow) acceptReviewOnce() bool {
	if flow.reviewAccepted {
		return false
	}
	flow.reviewAccepted = true
	return true
}

func (flow *forkCombineFlow) claimMutationOnce() bool {
	if flow.started {
		return false
	}
	flow.started = true
	return true
}

func (flow *forkCombineFlow) checkRepository() error {
	if flow.c.Git() != flow.git || flow.c.State().GetRepoGeneration() != flow.generation {
		return errors.New(flow.c.Tr.CombineRepositoryChanged)
	}
	return nil
}

func (flow *forkCombineFlow) prepare(changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	draft, editorPlan := flow.draft, flow.editorPlan
	return flow.c.WithWaitingStatus(flow.c.Tr.CombineChecking, func(gocui.Task) error {
		plan, err := flow.git.Rebase.PrepareCombinePlan(slices.Clone(flow.oids))
		if err != nil {
			if editorPlan != nil {
				flow.c.OnUIThread(func() error { return flow.recoverDraft(*editorPlan, draft, err) })
				return nil
			}
			return err
		}
		flow.c.OnUIThread(func() error { return flow.review(plan, changed) })
		return nil
	})
}

func (flow *forkCombineFlow) review(plan git_commands.CapturedCombinePlan, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	paths := plan.IgnoredFiles()
	if len(paths) > 0 {
		quoted := make([]string, len(paths))
		for i, path := range paths {
			quoted[i] = fmt.Sprintf("%q", path)
		}
		flow.c.Confirm(types.ConfirmOpts{
			Title:         flow.c.Tr.CombineIgnoredTitle,
			Prompt:        fmt.Sprintf(flow.c.Tr.CombineIgnoredPrompt, strings.Join(quoted, "\n")),
			HandleClose:   flow.checkRepository,
			HandleConfirm: func() error { return flow.reviewApproved(plan.WithIgnoredFilesApproved(), changed) },
		})
		return nil
	}
	return flow.reviewApproved(plan, changed)
}

func (flow *forkCombineFlow) reviewApproved(plan git_commands.CapturedCombinePlan, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	topology := plan.Topology()
	selected := topology.SelectedCommits()
	lines := make([]string, 0, len(selected))
	for i, commit := range selected {
		subject, _, _ := strings.Cut(commit.Message, "\n")
		lines = append(lines, fmt.Sprintf("%d. %s %s", i+1, commit.OID, subject))
	}
	headRef := strings.TrimPrefix(plan.HeadRef(), "refs/heads/")
	if headRef == "" {
		headRef = flow.c.Tr.BranchActionDetachedHead
	}
	prompt := fmt.Sprintf(flow.c.Tr.CombineReviewPrompt, headRef, topology.HeadOID(), topology.KeeperOID(), strings.Join(lines, "\n"))
	if changed {
		prompt = flow.c.Tr.CombineChanged + "\n\n" + prompt
	}
	flow.c.Confirm(types.ConfirmOpts{Title: flow.c.Tr.CombineReviewTitle, Prompt: prompt, HandleClose: flow.checkRepository, HandleConfirm: func() error {
		if err := flow.checkRepository(); err != nil {
			return err
		}
		if !flow.acceptReviewOnce() {
			return errors.New("combine review has already been accepted")
		}
		message := flow.draft
		if message == "" {
			message = topology.Summary() + "\n\n" + topology.Description()
		}
		return flow.openMessageEditor(message, plan)
	}})
	return nil
}

func combineMessage(summary, description string) string {
	if description == "" {
		return summary
	}
	return summary + "\n\n" + description
}

func (flow *forkCombineFlow) stale(plan git_commands.CapturedCombinePlan) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	return flow.c.Menu(types.CreateMenuOptions{Title: flow.c.Tr.CombineReviewTitle, Prompt: flow.c.Tr.CombineChanged, Items: []*types.MenuItem{
		{CommandID: "commit.combine-selected.review-fresh", Verb: "review", Label: flow.c.Tr.CombineFreshReview, OnPress: func() error { return flow.prepare(true) }},
		{CommandID: "commit.combine-selected.edit-draft", Verb: "edit", Label: flow.c.Tr.Edit, OnPress: func() error { return flow.openMessageEditor(flow.draft, plan) }},
		{CommandID: "commit.combine-selected.cancel-stale", Verb: "cancel", Label: flow.c.Tr.Cancel, OnPress: flow.checkRepository},
	}})
}

func (flow *forkCombineFlow) openMessageEditor(message string, plan git_commands.CapturedCombinePlan) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	flow.draft = message
	flow.editorPlan = &plan
	flow.c.Helpers().Commits.OpenCommitMessagePanel(&helpers.OpenCommitMessagePanelOpts{SummaryTitle: flow.c.Tr.CombineKeeperTitle, DescriptionTitle: flow.c.Tr.CombineDescriptionTitle, InitialMessage: message, OnConfirm: func(summary, description string) error { return flow.apply(plan, combineMessage(summary, description)) }})
	return nil
}

func (flow *forkCombineFlow) recoverDraft(plan git_commands.CapturedCombinePlan, message string, cause error) error {
	if err := flow.checkRepository(); err != nil {
		return errors.Join(cause, err)
	}
	if err := flow.openMessageEditor(message, plan); err != nil {
		return errors.Join(cause, err)
	}
	flow.c.Alert(flow.c.Tr.CombineErrorTitle, cause.Error())
	return nil
}

func (flow *forkCombineFlow) apply(plan git_commands.CapturedCombinePlan, message string) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	if !flow.claimMutationOnce() {
		return errors.New("combine command has already started")
	}
	flow.draft = message
	signing := flow.git.Config.NeedsGpgSubprocessForCommit()
	return flow.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: flow.c.Tr.CombineApplying}, func(gocui.Task) error {
		if err := flow.checkRepository(); err != nil {
			flow.c.RefreshFromWorker(selectedRebaseRefreshOptions(false))
			flow.c.OnUIThread(func() error {
				flow.started = false
				return flow.checkRepository()
			})
			return nil
		}
		cmd, cleanup, err := flow.git.Rebase.CombineCommand(plan, message)
		if err != nil {
			if errors.Is(err, git_commands.ErrBranchActionChanged) || errors.Is(err, git_commands.ErrRebaseIgnoredApprovalRequired) {
				flow.c.RefreshFromWorker(selectedRebaseRefreshOptions(false))
				flow.c.OnUIThread(func() error { flow.started = false; flow.reviewAccepted = false; return flow.stale(plan) })
				return nil
			}
			flow.c.RefreshFromWorker(selectedRebaseRefreshOptions(false))
			flow.c.OnUIThread(func() error {
				flow.started = false
				return flow.recoverDraft(plan, message, err)
			})
			return nil
		}
		if signing {
			flow.c.OnUIThread(func() error { return flow.runTerminal(cmd, cleanup) })
			return nil
		}
		defer cleanup()
		result := cmd.Run()
		return flow.c.Helpers().MergeAndRebase.CheckMergeOrRebaseWithRefreshOptions(result, selectedRebaseRefreshOptions(result == nil))
	})
}

func (flow *forkCombineFlow) runTerminal(cmd *oscommands.CmdObj, cleanup func()) error {
	defer cleanup()
	if err := flow.checkRepository(); err != nil {
		return err
	}
	_, result := flow.c.RunSubprocess(cmd)
	return flow.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: flow.c.Tr.CombineApplying}, func(gocui.Task) error {
		return flow.c.Helpers().MergeAndRebase.CheckRebaseSubprocessResult(result, selectedRebaseRefreshOptions(result == nil))
	})
}
