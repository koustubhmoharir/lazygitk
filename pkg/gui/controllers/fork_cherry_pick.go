package controllers

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func forkCherryPickBinding(c *ControllerCommon) *types.Binding {
	return &types.Binding{
		CommandID: "commit.cherry-pick", Verb: "cherry-pick", Description: c.Tr.CherryPick, OpensMenu: true,
		Handler:           func() error { return beginForkCherryPick(c) },
		GetDisabledReason: func() *types.DisabledReason { return forkCherryPickDisabledReason(c) },
	}
}

func forkCherryPickSelection(c *ControllerCommon) []string {
	current := c.Context().CurrentStatic()
	commits, ok := current.(ContainsCommits)
	if !ok && current.GetKind() == types.MAIN_CONTEXT {
		commits, ok = c.Context().CurrentSide().(ContainsCommits)
	}
	if !ok {
		return nil
	}
	selected, _, _ := commits.GetSelectedItems()
	return forkCherryCommitIDs(selected)
}

func forkCherryCommitIDs(commits []*models.Commit) []string {
	ids := make([]string, 0, len(commits))
	for _, commit := range commits {
		if commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
			return nil
		}
		ids = append(ids, commit.Hash())
	}
	return ids
}

func forkCherryPickDisabledReason(c *ControllerCommon) *types.DisabledReason {
	if len(forkCherryCommitIDs(c.Modes().CherryPicking.CherryPickedCommits)) == 0 && len(forkCherryPickSelection(c)) == 0 {
		return &types.DisabledReason{Text: c.Tr.NoCopiedCommits, Inapplicable: true}
	}
	return nil
}

type forkCherryPickFlow struct {
	c               *ControllerCommon
	git             *commands.GitCommand
	generation      int
	oids            []string
	marked          []string
	usedMarked      bool
	ignoredApproved bool
	started         bool
}

func beginForkCherryPick(c *ControllerCommon) error {
	flow := &forkCherryPickFlow{c: c, git: c.Git(), generation: c.State().GetRepoGeneration()}
	marked := forkCherryCommitIDs(c.Modes().CherryPicking.CherryPickedCommits)
	selected := forkCherryPickSelection(c)
	if len(marked) == 0 && len(selected) == 0 {
		return nil
	}
	choose := func(ids []string, fromMarks bool) func() error {
		captured := slices.Clone(ids)
		return func() error {
			if err := flow.checkRepository(); err != nil {
				return err
			}
			flow.oids = captured
			flow.marked = slices.Clone(marked)
			flow.usedMarked = fromMarks
			return flow.prepare(false)
		}
	}
	if len(marked) > 0 && len(selected) > 0 {
		return c.Menu(types.CreateMenuOptions{
			Title: c.Tr.CherryPickSourceTitle,
			Items: []*types.MenuItem{
				{CommandID: "commit.cherry-pick.marked", Verb: "cherry-pick", Label: fmt.Sprintf(c.Tr.CherryPickMarkedSource, len(marked)), OnPress: choose(marked, true), OpensMenu: true},
				{CommandID: "commit.cherry-pick.selected", Verb: "cherry-pick", Label: fmt.Sprintf(c.Tr.CherryPickSelectedSource, len(selected)), OnPress: choose(selected, false), OpensMenu: true},
			}, OnCancel: flow.checkRepository,
		})
	}
	if len(marked) > 0 {
		return choose(marked, true)()
	}
	return choose(selected, false)()
}

func (flow *forkCherryPickFlow) checkRepository() error {
	if flow.c.Git() != flow.git || flow.c.State().GetRepoGeneration() != flow.generation {
		return errors.New(flow.c.Tr.BranchSyncRepositoryChanged)
	}
	return nil
}

func (flow *forkCherryPickFlow) prepare(changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	flow.ignoredApproved = false
	return flow.c.WithWaitingStatus(flow.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.checkRepository(); err != nil {
			return err
		}
		plan, err := flow.git.Rebase.PrepareCherryPickPlan(flow.oids)
		if err != nil {
			return err
		}
		flow.c.OnUIThread(func() error { return flow.review(plan, changed) })
		return nil
	})
}

func (flow *forkCherryPickFlow) review(plan git_commands.CherryPickPlan, changed bool) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	minimum := 0
	for _, commit := range plan.Commits() {
		if len(commit.Parents) > 1 && (minimum == 0 || len(commit.Parents) < minimum) {
			minimum = len(commit.Parents)
		}
	}
	if minimum > 0 && plan.Mainline() == 0 {
		items := make([]*types.MenuItem, 0, minimum)
		for parent := 1; parent <= minimum; parent++ {
			items = append(items, &types.MenuItem{
				CommandID: "commit.cherry-pick.choose-mainline", Verb: "choose-mainline", Label: fmt.Sprintf(flow.c.Tr.CherryPickMainlineChoice, parent), OpensMenu: true,
				OnPress: func() error {
					chosen, err := plan.WithMainline(parent)
					if err != nil {
						return err
					}
					return flow.review(chosen, changed)
				},
			})
		}
		return flow.c.Menu(types.CreateMenuOptions{Title: flow.c.Tr.CherryPickMainlineTitle, Prompt: flow.c.Tr.CherryPickMainlinePrompt, Items: items, OnCancel: flow.checkRepository})
	}
	if risks := plan.IgnoredFiles(); len(risks) > 0 && !flow.ignoredApproved {
		flow.c.Confirm(types.ConfirmOpts{
			Title: flow.c.Tr.SelectedRebaseIgnoredTitle, Prompt: flow.c.Tr.CherryPickIgnoredPrompt + "\n\n- " + strings.Join(risks, "\n- "),
			HandleClose: flow.checkRepository,
			HandleConfirm: func() error {
				flow.ignoredApproved = true
				return flow.review(plan.WithIgnoredFilesApproved(), changed)
			},
		})
		return nil
	}
	destination := plan.HeadRef()
	if destination == "" {
		destination = flow.c.Tr.BranchActionDetachedHead
	}
	head := plan.HeadOID()
	if head == "" {
		head = flow.c.Tr.BranchActionUnbornHead
	}
	lines := make([]string, 0, len(plan.Commits()))
	for i, commit := range plan.Commits() {
		line := fmt.Sprintf("%d. %s %s", i+1, commit.OID, commit.Subject)
		if len(commit.Parents) > 1 {
			line += fmt.Sprintf(flow.c.Tr.CherryPickMergeParentDetail, plan.Mainline(), commit.Parents[plan.Mainline()-1])
		}
		lines = append(lines, line)
	}
	prompt := fmt.Sprintf(flow.c.Tr.CherryPickReviewPrompt, destination, head, strings.Join(lines, "\n"))
	if changed {
		prompt = flow.c.Tr.CherryPickChangedNote + "\n\n" + prompt
	}
	flow.c.Confirm(types.ConfirmOpts{Title: flow.c.Tr.CherryPick, Prompt: prompt, HandleClose: flow.checkRepository, HandleConfirm: func() error { return flow.apply(plan) }})
	return nil
}

func (flow *forkCherryPickFlow) apply(plan git_commands.CherryPickPlan) error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	if flow.started {
		return git_commands.ErrBranchActionChanged
	}
	flow.started = true
	return flow.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: flow.c.Tr.CherryPickingStatus, HideWorkingTreeState: true}, func(gocui.Task) error {
		if err := flow.checkRepository(); err != nil {
			return err
		}
		cmd, err := flow.git.Rebase.CherryPickCommand(plan)
		if errors.Is(err, git_commands.ErrBranchActionChanged) || errors.Is(err, git_commands.ErrRebaseIgnoredApprovalRequired) {
			next := *flow
			next.started = false
			flow.c.OnUIThread(func() error { return next.prepare(true) })
			return nil
		}
		if err != nil {
			return err
		}
		flow.c.LogAction(flow.c.Tr.Actions.CherryPick)
		result := cmd.Run()
		if err := flow.checkRepository(); err != nil {
			return errors.Join(result, err)
		}
		// Preserve upstream continuation/conflict handling, including picking
		// while an interactive rebase is paused. Only source capture/review
		// and the removal of implicit stashing differ from upstream paste.
		handled := flow.c.Helpers().MergeAndRebase.CheckMergeOrRebaseWithRefreshOptions(result, types.RefreshOptions{BatchUIUpdates: true})
		if handled != nil {
			return handled
		}
		if result == nil && flow.usedMarked {
			flow.c.OnUIThread(func() error {
				if err := flow.checkRepository(); err != nil {
					return err
				}
				if slices.Equal(flow.marked, forkCherryCommitIDs(flow.c.Modes().CherryPicking.CherryPickedCommits)) {
					flow.c.Modes().CherryPicking.DidPaste = true
				}
				return nil
			})
		}
		return nil
	})
}
