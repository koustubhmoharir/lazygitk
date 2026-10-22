package controllers

import (
	"errors"
	"fmt"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchStackPushFlow struct {
	target  *branchSyncTarget
	members []git_commands.BranchStackMember
	plans   []git_commands.BranchPushPlan
	started bool
}

func (batch *branchStackPushFlow) reviewNext() error {
	if err := batch.target.checkRepository(); err != nil {
		return err
	}
	if len(batch.members) == 0 || batch.members[0].Name != batch.target.source.name {
		return git_commands.ErrBranchSyncChanged
	}
	index := len(batch.plans)
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
		flow := newBranchSyncPushFlow(target, info)
		flow.reviewContext = fmt.Sprintf(target.c.Tr.BranchSyncStackMemberReview, index+1, len(batch.members))
		flow.acceptPlan = func(plan git_commands.BranchPushPlan) error {
			if err := batch.target.checkRepository(); err != nil {
				return err
			}
			if batch.started || len(batch.plans) != index || plan.Name() != member.Name || plan.SourceOID() != member.OID {
				return git_commands.ErrBranchSyncChanged
			}
			for _, previous := range batch.plans {
				if stackPushDestinationsOverlap(previous, plan) {
					return fmt.Errorf("branches %q and %q share a configured push destination; choose distinct destinations or push them separately", previous.Name(), plan.Name())
				}
			}
			batch.plans = append(batch.plans, plan)
			return batch.reviewNext()
		}
		return flow.menu()
	})
}

func stackPushDestinationsOverlap(first, second git_commands.BranchPushPlan) bool {
	if first.Options().DestinationRef != second.Options().DestinationRef {
		return false
	}
	if first.Options().Remote == second.Options().Remote {
		return true
	}
	for _, url := range first.ConfiguredPushURLs() {
		if slices.Contains(second.ConfiguredPushURLs(), url) {
			return true
		}
	}
	return false
}

func (batch *branchStackPushFlow) planPrompt(plan git_commands.BranchPushPlan) string {
	flow := &branchSyncPushFlow{target: batch.target}
	prompt := flow.planReviewPrompt(plan)
	if urls := plan.ConfiguredPushURLs(); len(urls) > 1 {
		prompt += "\n\n" + batch.target.c.Tr.BranchSyncMultipleURLsPrompt
		for _, raw := range urls {
			prompt += "\n" + safeBranchSyncURL(raw)
		}
	}
	return prompt
}

func (batch *branchStackPushFlow) reviewBatch() error {
	if err := batch.target.checkRepository(); err != nil {
		return err
	}
	if batch.started || len(batch.plans) == 0 || len(batch.plans) != len(batch.members) {
		return git_commands.ErrBranchSyncChanged
	}
	items := []*types.MenuItem{{CommandID: "branch.push-stack.cancel", Verb: "cancel", Label: batch.target.c.Tr.Cancel, OnPress: batch.target.checkRepository}}
	for _, plan := range batch.plans {
		options := plan.Options()
		prompt := batch.planPrompt(plan)
		items = append(items, &types.MenuItem{
			CommandID: "branch.push-stack.inspect-plan", Verb: "inspect", OpensMenu: true,
			Label: fmt.Sprintf("%q %s → %q %q", plan.Name(), plan.SourceOID(), options.Remote, options.DestinationRef), Tooltip: prompt,
			OnPress: func() error {
				if err := batch.target.checkRepository(); err != nil {
					return err
				}
				return batch.target.c.Menu(types.CreateMenuOptions{
					Title: batch.target.c.Tr.BranchSyncReviewTitle, Prompt: prompt,
					Items:    []*types.MenuItem{{CommandID: "branch.push-stack.back", Verb: "back", Label: batch.target.c.Tr.Close, OpensMenu: true, OnPress: batch.reviewBatch}},
					OnCancel: batch.reviewBatch,
				})
			},
		})
	}
	items = append(items, &types.MenuItem{CommandID: "branch.push-stack.execute", Verb: "push-stack", Label: batch.target.c.Tr.BranchSyncStackPushExecute, OnPress: batch.execute})
	return batch.target.c.Menu(types.CreateMenuOptions{
		Title: batch.target.c.Tr.BranchSyncStackPushTitle, Prompt: batch.target.c.Tr.BranchSyncStackPushPrompt,
		Items: items, HideCancel: true, OnCancel: batch.target.checkRepository,
	})
}

func (batch *branchStackPushFlow) execute() error {
	if err := batch.target.checkRepository(); err != nil {
		return err
	}
	if batch.started || len(batch.plans) == 0 || len(batch.plans) != len(batch.members) {
		return git_commands.ErrBranchSyncChanged
	}
	// These are UI busy indicators, not repository/ref/network locks. Other Git
	// processes remain governed by Git's normal concurrency and lease checks.
	items := make([]types.HasUrn, 0, len(batch.plans))
	for _, plan := range batch.plans {
		item := syncOperationItem(batch.target.git, plan.Name())
		ordinary := &models.Branch{Name: plan.Name()}
		if batch.target.c.State().GetItemOperation(item) != types.ItemOperationNone || batch.target.c.State().GetItemOperation(ordinary) != types.ItemOperationNone {
			return errors.New(batch.target.c.Tr.CantPullOrPushSameBranchTwice)
		}
		items = append(items, item)
	}
	for _, item := range items {
		batch.target.c.State().SetItemOperation(item, types.ItemOperationPushing)
	}
	clear := func() {
		for _, item := range items {
			batch.target.c.State().ClearItemOperation(item)
		}
	}
	// Approval is consumed even after a failure: replaying the same batch can
	// repeat already successful remote writes. Reopen review instead.
	batch.started = true
	plans := slices.Clone(batch.plans)
	err := batch.target.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: batch.target.c.Tr.BranchSyncStackPushStatus}, func(task gocui.Task) error {
		defer clear()
		defer func() {
			if batch.target.checkRepository() == nil {
				batch.target.c.RefreshFromWorker(types.RefreshOptions{})
			}
		}()
		if err := batch.target.checkRepository(); err != nil {
			return err
		}
		// All plans are checked before the first push, and each is checked again
		// immediately before its native command. This is not an atomic batch.
		for _, plan := range plans {
			if _, err := batch.target.git.Sync.BranchPushCommand(plan, task); err != nil {
				return fmt.Errorf("stack push preflight failed for %q; no branches were pushed: %w", plan.Name(), err)
			}
		}
		completed := []string{}
		for _, plan := range plans {
			if err := batch.target.checkRepository(); err != nil {
				return fmt.Errorf(batch.target.c.Tr.BranchSyncStackPushFailed, plan.Name(), completed, err)
			}
			cmd, err := batch.target.git.Sync.BranchPushCommand(plan, task)
			if err == nil {
				err = cmd.Run()
				if err == nil {
					completed = append(completed, plan.Name())
				}
			}
			err = errors.Join(err, batch.target.checkRepository())
			if err != nil {
				return fmt.Errorf(batch.target.c.Tr.BranchSyncStackPushFailed, plan.Name(), completed, err)
			}
		}
		return nil
	})
	if err != nil {
		clear()
	}
	return err
}
