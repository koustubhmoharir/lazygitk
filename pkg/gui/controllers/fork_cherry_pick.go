package controllers

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func forkCherryPickBinding(c *ControllerCommon) *types.Binding {
	return &types.Binding{
		CommandID: "commit.cherry-pick", Verb: "cherry-pick", Description: c.Tr.CherryPick, OpensMenu: true,
		Handler:           func() error { return beginForkCherryPick(c) },
		GetDisabledReason: func() *types.DisabledReason { return forkCherryPickDisabledReason(c) },
	}
}

// forkCherryPickSource returns the commit list that c p picks from, and the hashes its
// selection covers, including members the list does not show. ok is false when the focus
// is not a commit list.
func forkCherryPickSource(c *ControllerCommon) (source ContainsCommits, hashes []string, ok bool) {
	current := c.Context().CurrentStatic()
	source, ok = current.(ContainsCommits)
	if !ok && current.GetKind() == types.MAIN_CONTEXT {
		source, ok = c.Context().CurrentSide().(ContainsCommits)
	}
	if !ok {
		return nil, nil, false
	}
	return source, forkSelectionHashes(source), true
}

// forkCherryCommitIDs returns the hashes when all of them are full object IDs, and nil
// otherwise.
func forkCherryCommitIDs(hashes []string) []string {
	for _, hash := range hashes {
		if !isFullSelectedCommitOID(hash) {
			return nil
		}
	}
	return slices.Clone(hashes)
}

func forkCherryPickDisabledReason(c *ControllerCommon) *types.DisabledReason {
	_, hashes, ok := forkCherryPickSource(c)
	if !ok || len(hashes) == 0 {
		return &types.DisabledReason{Text: c.Tr.NoCommitsSelected, Inapplicable: true}
	}
	if len(forkCherryCommitIDs(hashes)) == 0 {
		return &types.DisabledReason{Text: c.Tr.CannotCherryPickNonCommit, Inapplicable: true}
	}
	return nil
}

type forkCherryPickFlow struct {
	c               *ControllerCommon
	git             *commands.GitCommand
	generation      int
	source          ContainsCommits // the list whose selection is picked
	oids            []string        // the selection when c p began, sorted by hash
	ignoredApproved bool
	started         bool
}

func beginForkCherryPick(c *ControllerCommon) error {
	source, hashes, ok := forkCherryPickSource(c)
	oids := forkCherryCommitIDs(hashes)
	if !ok || len(oids) == 0 {
		return nil
	}
	slices.Sort(oids)
	flow := &forkCherryPickFlow{c: c, git: c.Git(), generation: c.State().GetRepoGeneration(), source: source, oids: oids}
	return flow.prepare(false)
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
		// Decided by the state the operation was left in, not by the raw pick result:
		// before Git 2.45 an empty pick stops Git, and the helper skips it, which can
		// finish the whole operation without an error.
		if flow.c.Git().Status.WorkingTreeState().None() {
			flow.c.OnUIThread(flow.clearPickedSelection)
		} else {
			// A pick that stops on conflicts keeps the selection until the operation it
			// left is continued or skipped to its end; an abort keeps it (see settleOperationFinished).
			flow.c.OnUIThread(flow.waitForOperationFinished)
		}
		return nil
	})
}

// waitForOperationFinished registers clearPickedSelection to run when the operation
// left by the pick ends. It runs on the UI thread, which may be later than the worker
// that saw the operation in progress; if the operation has ended since, nothing is
// registered and the selection stays.
func (flow *forkCherryPickFlow) waitForOperationFinished() error {
	if flow.c.Git().Status.WorkingTreeState().None() {
		return nil
	}
	flow.c.Helpers().MergeAndRebase.SetOnOperationFinished(flow.clearPickedSelection)
	return nil
}

// clearPickedSelection clears the selection that c p picked, after the pick succeeded.
// A selection that is no longer the one picked is left alone.
func (flow *forkCherryPickFlow) clearPickedSelection() error {
	if err := flow.checkRepository(); err != nil {
		return err
	}
	current := forkCherryCommitIDs(forkSelectionHashes(flow.source))
	slices.Sort(current)
	if !slices.Equal(current, flow.oids) {
		return nil
	}
	if commits, ok := flow.source.(*context.LocalCommitsContext); ok {
		commits.ClearMultiSelection()
	} else {
		flow.source.GetList().CancelRangeSelect()
	}
	flow.c.PostRefreshUpdate(flow.source)
	return nil
}
