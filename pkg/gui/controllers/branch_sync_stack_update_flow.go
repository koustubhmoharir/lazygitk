package controllers

import (
	"errors"
	"fmt"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func (self *SyncController) reviewBranchStackUpdate(target *branchSyncTarget, info git_commands.BranchSyncInfo) error {
	return target.inspectStack(info, func(members []git_commands.BranchStackMember) error {
		if len(members) == 1 {
			return target.c.Menu(types.CreateMenuOptions{
				Title: target.c.Tr.BranchSyncStackUpdateTitle, Prompt: target.c.Tr.BranchSyncStackUpdateNoMembers,
				Items:    []*types.MenuItem{{CommandID: "branch.update-stack.no-members", Verb: "cancel", Label: target.c.Tr.Cancel, OnPress: target.checkRepository}},
				OnCancel: target.checkRepository,
			})
		}
		batch := &branchStackUpdateFlow{target: target, members: members}
		return batch.reviewNext()
	})
}

type reviewedBranchUpdate struct {
	forward *git_commands.BranchFastForwardPlan
	current *git_commands.CurrentPullIntegrationPlan
}

func (plan reviewedBranchUpdate) name() string {
	if plan.forward != nil {
		return plan.forward.Name()
	}
	if plan.current != nil {
		return plan.current.Name()
	}
	return ""
}

func (plan reviewedBranchUpdate) sourceOID() string {
	if plan.forward != nil {
		return plan.forward.OldOID()
	}
	if plan.current != nil {
		return plan.current.SourceOID()
	}
	return ""
}

func (plan reviewedBranchUpdate) targetOID() string {
	if plan.forward != nil {
		return plan.forward.TargetOID()
	}
	if plan.current != nil {
		return plan.current.TargetOID()
	}
	return ""
}

func (plan reviewedBranchUpdate) check(target *branchSyncTarget) error {
	if (plan.forward == nil) == (plan.current == nil) {
		return errors.New("update review must contain exactly one prepared plan")
	}
	if plan.forward != nil {
		return target.git.Sync.CheckBranchFastForward(*plan.forward)
	}
	_, err := target.git.Sync.CurrentPullIntegrationCommand(*plan.current)
	return err
}

type branchStackUpdateFlow struct {
	target  *branchSyncTarget
	members []git_commands.BranchStackMember
	plans   []reviewedBranchUpdate
	started bool
}

func (batch *branchStackUpdateFlow) reviewNext() error {
	if err := batch.target.checkRepository(); err != nil {
		return err
	}
	if batch.started || len(batch.members) == 0 || batch.members[0].Name != batch.target.source.name {
		return git_commands.ErrBranchSyncChanged
	}
	index := len(batch.plans)
	if index > len(batch.members) {
		return git_commands.ErrBranchSyncChanged
	}
	if index == len(batch.members) {
		return batch.reviewBatch()
	}
	member := batch.members[index]
	target := &branchSyncTarget{
		c: batch.target.c, git: batch.target.git, generation: batch.target.generation,
		source: branchSyncSource{name: member.Name, selected: index != 0 || batch.target.source.selected},
	}
	return target.inspect(func(info git_commands.BranchSyncInfo) error {
		if info.OID != member.OID {
			return git_commands.ErrBranchSyncChanged
		}
		flow := &branchSyncUpdateFlow{
			target: target, info: info, options: branchSyncUpdateDefaults(info), currentMode: info.HeadRef == info.FullRef,
		}
		flow.collector = &branchSyncUpdateCollector{
			context: fmt.Sprintf(target.c.Tr.BranchSyncStackUpdateContext, index+1, len(batch.members)),
			acceptForward: func(plan git_commands.BranchFastForwardPlan) error {
				return batch.accept(index, reviewedBranchUpdate{forward: &plan})
			},
			acceptCurrent: func(plan git_commands.CurrentPullIntegrationPlan) error {
				return batch.accept(index, reviewedBranchUpdate{current: &plan})
			},
		}
		return flow.menu()
	})
}

func (batch *branchStackUpdateFlow) accept(index int, plan reviewedBranchUpdate) error {
	if err := batch.target.checkRepository(); err != nil {
		return err
	}
	if (plan.forward == nil) == (plan.current == nil) {
		return git_commands.ErrBranchSyncChanged
	}
	if batch.started || index != len(batch.plans) || index >= len(batch.members) || plan.name() != batch.members[index].Name || plan.sourceOID() != batch.members[index].OID {
		return git_commands.ErrBranchSyncChanged
	}
	for _, previous := range batch.plans {
		if previous.name() == plan.name() || (previous.current != nil && plan.current != nil) {
			return git_commands.ErrBranchSyncChanged
		}
	}
	batch.plans = append(batch.plans, plan)
	return batch.reviewNext()
}

// Keep ref-only work before current-worktree integration, as described in the
// final approval. Review order alone never determines mutation order.
func (batch *branchStackUpdateFlow) executionOrder() []reviewedBranchUpdate {
	ordered := make([]reviewedBranchUpdate, 0, len(batch.plans))
	for _, plan := range batch.plans {
		if plan.forward != nil {
			ordered = append(ordered, plan)
		}
	}
	for _, plan := range batch.plans {
		if plan.current != nil {
			ordered = append(ordered, plan)
		}
	}
	return ordered
}

func (batch *branchStackUpdateFlow) hasCompleteReview() bool {
	if len(batch.plans) == 0 || len(batch.plans) != len(batch.members) || batch.members[0].Name != batch.target.source.name {
		return false
	}
	names := map[string]bool{}
	currentCount := 0
	for index, plan := range batch.plans {
		if (plan.forward == nil) == (plan.current == nil) || names[plan.name()] || plan.name() != batch.members[index].Name || plan.sourceOID() != batch.members[index].OID {
			return false
		}
		names[plan.name()] = true
		if plan.current != nil {
			currentCount++
		}
	}
	return currentCount <= 1
}

func (batch *branchStackUpdateFlow) planPrompt(plan reviewedBranchUpdate) string {
	if plan.current != nil {
		flow := &branchSyncUpdateFlow{target: batch.target}
		prompt := flow.integrationReviewPrompt(*plan.current)
		if paths := plan.current.IgnoredFiles(); len(paths) > 0 {
			prompt += fmt.Sprintf(batch.target.c.Tr.BranchSyncStackUpdateIgnoredApproved, len(paths))
		}
		return prompt
	}
	if !plan.forward.NeedsUpdate() {
		return fmt.Sprintf(batch.target.c.Tr.BranchSyncAlreadyAhead, plan.name(), plan.sourceOID(), plan.targetOID())
	}
	return fmt.Sprintf(batch.target.c.Tr.BranchSyncUpdateReviewPrompt, plan.name(), plan.sourceOID(), plan.targetOID())
}

func (batch *branchStackUpdateFlow) reviewBatch() error {
	if err := batch.target.checkRepository(); err != nil {
		return err
	}
	if batch.started || !batch.hasCompleteReview() {
		return git_commands.ErrBranchSyncChanged
	}
	items := []*types.MenuItem{{CommandID: "branch.update-stack.cancel", Verb: "cancel", Label: batch.target.c.Tr.Cancel, OnPress: batch.target.checkRepository}}
	for _, plan := range batch.executionOrder() {
		prompt := batch.planPrompt(plan)
		label := fmt.Sprintf("%q %s → %s", plan.name(), utils.ShortHash(plan.sourceOID()), utils.ShortHash(plan.targetOID()))
		if plan.current != nil {
			label = fmt.Sprintf(batch.target.c.Tr.BranchSyncStackUpdateCurrentLabel, plan.name(), utils.ShortHash(plan.targetOID()), utils.ShortHash(plan.sourceOID())) + batch.target.c.Tr.BranchSyncStackUpdateCurrentLast
		} else if !plan.forward.NeedsUpdate() {
			label = fmt.Sprintf(batch.target.c.Tr.BranchSyncStackUpdateNoopLabel, plan.name(), utils.ShortHash(plan.sourceOID()), utils.ShortHash(plan.targetOID()))
		}
		items = append(items, &types.MenuItem{
			CommandID: "branch.update-stack.inspect-plan", Verb: "inspect", OpensMenu: true, Label: label, Tooltip: prompt,
			OnPress: func() error {
				if err := batch.target.checkRepository(); err != nil {
					return err
				}
				return batch.target.c.Menu(types.CreateMenuOptions{
					Title: batch.target.c.Tr.BranchSyncStackUpdateTitle, Prompt: prompt,
					Items:    []*types.MenuItem{{CommandID: "branch.update-stack.back", Verb: "back", Label: batch.target.c.Tr.Close, OpensMenu: true, OnPress: batch.reviewBatch}},
					OnCancel: batch.reviewBatch,
				})
			},
		})
	}
	items = append(items, &types.MenuItem{CommandID: "branch.update-stack.execute", Verb: "update-stack", Label: batch.target.c.Tr.BranchSyncStackUpdateExecute, OnPress: batch.execute})
	return batch.target.c.Menu(types.CreateMenuOptions{
		Title: batch.target.c.Tr.BranchSyncStackUpdateTitle, Prompt: batch.target.c.Tr.BranchSyncStackUpdatePrompt,
		Items: items, HideCancel: true, OnCancel: batch.target.checkRepository,
	})
}

func (batch *branchStackUpdateFlow) execute() error {
	if err := batch.target.checkRepository(); err != nil {
		return err
	}
	if batch.started || !batch.hasCompleteReview() {
		return git_commands.ErrBranchSyncChanged
	}
	ordered := batch.executionOrder()
	offItems := []types.HasUrn{}
	var currentItem types.HasUrn
	for _, plan := range ordered {
		item := syncOperationItem(batch.target.git, plan.name())
		ordinary := &models.Branch{Name: plan.name()}
		if batch.target.c.State().GetItemOperation(item) != types.ItemOperationNone || batch.target.c.State().GetItemOperation(ordinary) != types.ItemOperationNone {
			return errors.New(batch.target.c.Tr.CantPullOrPushSameBranchTwice)
		}
		if plan.current != nil {
			currentItem = item
		} else {
			offItems = append(offItems, item)
		}
	}
	for _, item := range offItems {
		batch.target.c.State().SetItemOperation(item, types.ItemOperationPulling)
	}
	currentHeld := currentItem != nil
	if currentHeld {
		batch.target.c.State().SetItemOperation(currentItem, types.ItemOperationPulling)
	}
	releaseCurrent := func() {
		if currentHeld {
			batch.target.c.State().ClearItemOperation(currentItem)
			currentHeld = false
		}
	}
	clear := func() {
		for _, item := range offItems {
			batch.target.c.State().ClearItemOperation(item)
		}
		releaseCurrent()
	}
	batch.started = true
	err := batch.target.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: batch.target.c.Tr.BranchSyncStackUpdateStatus}, func(gocui.Task) error {
		defer clear()
		defer func() {
			if batch.target.checkRepository() == nil {
				batch.target.c.RefreshFromWorker(types.RefreshOptions{})
			}
		}()
		if err := batch.target.checkRepository(); err != nil {
			return err
		}
		for _, plan := range ordered {
			if err := plan.check(batch.target); err != nil {
				return fmt.Errorf(batch.target.c.Tr.BranchSyncStackUpdatePreflightFailed, plan.name(), err)
			}
		}
		advanced := []string{}
		for _, plan := range ordered {
			if err := batch.target.checkRepository(); err != nil {
				return fmt.Errorf(batch.target.c.Tr.BranchSyncStackUpdateStopped, plan.name(), advanced, err)
			}
			if plan.current != nil {
				// Release this phase's current flag before queuing terminal work.
				// Its executor claims its own flag; deferred cleanup must not clear it.
				releaseCurrent()
				completed := slices.Clone(advanced)
				batch.target.c.OnUIThread(func() error { return batch.integrateCurrent(*plan.current, completed) })
				return nil
			}
			if err := batch.target.git.Sync.ApplyBranchFastForward(*plan.forward); err != nil {
				return fmt.Errorf(batch.target.c.Tr.BranchSyncStackUpdateStopped, plan.name(), advanced, err)
			}
			if plan.forward.NeedsUpdate() {
				advanced = append(advanced, plan.name())
			}
		}
		if err := batch.target.checkRepository(); err != nil {
			return fmt.Errorf(batch.target.c.Tr.BranchSyncStackUpdateStopped, ordered[len(ordered)-1].name(), advanced, err)
		}
		return nil
	})
	if err != nil {
		clear()
	}
	return err
}

func (batch *branchStackUpdateFlow) integrateCurrent(plan git_commands.CurrentPullIntegrationPlan, advanced []string) error {
	stopped := func(err error) error {
		var progress *branchUpdateProgressError
		if errors.As(err, &progress) && progress.batch == batch {
			return err
		}
		return &branchUpdateProgressError{
			batch: batch,
			cause: fmt.Errorf(batch.target.c.Tr.BranchSyncStackUpdateCurrentStopped, plan.Name(), advanced, err),
		}
	}
	if err := batch.target.checkRepository(); err != nil {
		return stopped(err)
	}
	target := &branchSyncTarget{
		c: batch.target.c, git: batch.target.git, generation: batch.target.generation,
		source: branchSyncSource{name: plan.Name(), selected: true},
	}
	return target.runReviewedIntegrationReporting(plan, nil, target.integrationSubprocessResult, stopped)
}

type branchUpdateProgressError struct {
	batch *branchStackUpdateFlow
	cause error
}

func (err *branchUpdateProgressError) Error() string { return err.cause.Error() }
func (err *branchUpdateProgressError) Unwrap() error { return err.cause }
