package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// pullBranchSource routes current-branch pulls through a captured review.
// Selected off-branch sources continue to use the established ref-only flow.
func (self *SyncController) pullBranchSource(source branchSyncSource) error {
	target := newBranchSyncTarget(self.c, source)
	return target.inspect(func(info git_commands.BranchSyncInfo) error {
		current := info.HeadRef == info.FullRef
		flow := &branchSyncUpdateFlow{target: target, info: info, options: branchSyncUpdateDefaults(info), currentMode: current}
		flow.stackChoice = func() error { return self.reviewBranchStackUpdate(target, flow.info) }
		return flow.menu()
	})
}

func (flow *branchSyncUpdateFlow) currentModeValid() error {
	return flow.target.checkRepository()
}

func (flow *branchSyncUpdateFlow) prepareLocalUpdate() error {
	if flow.currentMode {
		return flow.prepareCurrentLocalUpdate()
	}
	return flow.prepareOffBranchLocalUpdate()
}

func (flow *branchSyncUpdateFlow) prepareFetch() error {
	if flow.currentMode {
		return flow.prepareCurrentFetch()
	}
	return flow.prepareOffBranchFetch()
}

func (flow *branchSyncUpdateFlow) reinspect() error {
	if flow.currentMode {
		return flow.reinspectCurrent()
	}
	return flow.reinspectOffBranch()
}

func (flow *branchSyncUpdateFlow) prepareCurrentLocalUpdate() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	name, sourceOID, upstreamRef := flow.info.Name, flow.info.OID, flow.options.RemoteRef
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.currentModeValid(); err != nil {
			return err
		}
		review, targetOID, err := flow.target.git.Sync.PrepareLocalCurrentPullReview(name, sourceOID, upstreamRef)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.currentModeValid(); err != nil {
				return err
			}
			return flow.chooseIntegration(review, targetOID)
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) chooseIntegration(review git_commands.CurrentPullReview, targetOID string) error {
	if err := flow.currentModeValid(); err != nil {
		return err
	}
	if flow.fastForwardOnly {
		return flow.prepareIntegration(review, targetOID, git_commands.PullIntegrationFastForwardOnly)
	}
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		defaultPlan, defaultErr := flow.target.git.Sync.PrepareCurrentPullIntegration(review, targetOID, git_commands.PullIntegrationDefault)
		if errors.Is(defaultErr, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspectCurrent() })
			return nil
		}
		if defaultErr != nil && !isUnavailableCurrentPullDefault(defaultErr) {
			return defaultErr
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.currentModeValid(); err != nil {
				return err
			}
			return flow.presentIntegrationChoices(review, targetOID, defaultPlan, defaultErr)
		})
		return nil
	})
}

func isUnavailableCurrentPullDefault(err error) bool {
	return errors.Is(err, git_commands.ErrPullIntegrationChoiceRequired) ||
		errors.Is(err, git_commands.ErrRebaseRequiresClean) ||
		errors.Is(err, git_commands.ErrBranchSyncDiverged)
}

func (flow *branchSyncUpdateFlow) presentIntegrationChoices(review git_commands.CurrentPullReview, targetOID string, defaultPlan git_commands.CurrentPullIntegrationPlan, defaultErr error) error {
	if err := flow.currentModeValid(); err != nil {
		return err
	}
	methods := []struct {
		method git_commands.PullIntegrationMethod
		label  string
	}{
		{git_commands.PullIntegrationDefault, flow.target.c.Tr.BranchSyncIntegrationDefault},
		{git_commands.PullIntegrationFastForwardOnly, flow.target.c.Tr.BranchSyncIntegrationFFOnly},
		{git_commands.PullIntegrationMerge, flow.target.c.Tr.BranchSyncIntegrationMerge},
		{git_commands.PullIntegrationMergeNoFF, flow.target.c.Tr.BranchSyncIntegrationNoFF},
		{git_commands.PullIntegrationRebase, flow.target.c.Tr.BranchSyncIntegrationRebase},
		{git_commands.PullIntegrationRebaseMerges, flow.target.c.Tr.BranchSyncIntegrationMerges},
		{git_commands.PullIntegrationRebaseInteractive, flow.target.c.Tr.BranchSyncIntegrationInteractive},
	}
	items := make([]*types.MenuItem, 0, len(methods))
	for _, choice := range methods {
		item := &types.MenuItem{CommandID: "branch.pull-current." + pullIntegrationLeaf(choice.method), Verb: "integrate", Label: choice.label, OnPress: func() error { return flow.prepareIntegration(review, targetOID, choice.method) }}
		if choice.method == git_commands.PullIntegrationDefault {
			if defaultErr != nil {
				item.DisabledReason = &types.DisabledReason{Text: defaultErr.Error()}
				item.Tooltip = fmt.Sprintf(flow.target.c.Tr.BranchSyncIntegrationDefaultError, defaultErr)
			}
			if defaultErr == nil {
				if defaultPlan.UsesConfiguredRebaseOptions() {
					item.Label = flow.target.c.Tr.BranchSyncIntegrationConfiguredRebase
				}
				if defaultPlan.Method() == git_commands.PullIntegrationConfiguredMerge {
					item.Label = flow.target.c.Tr.BranchSyncIntegrationConfiguredMerge
				}
			}
		}
		items = append(items, item)
	}
	prompt := flow.collectionContext() + fmt.Sprintf(flow.target.c.Tr.BranchSyncIntegrationSummary, review.Name(), review.SourceOID(), targetOID)
	return flow.target.c.Menu(types.CreateMenuOptions{Title: flow.target.c.Tr.BranchSyncIntegrationMenu, Prompt: prompt, Items: items, OnCancel: flow.target.checkRepository})
}

func pullIntegrationLeaf(method git_commands.PullIntegrationMethod) string {
	switch method {
	case git_commands.PullIntegrationDefault:
		return "default"
	case git_commands.PullIntegrationFastForwardOnly:
		return "ff-only"
	case git_commands.PullIntegrationMerge:
		return "merge"
	case git_commands.PullIntegrationMergeNoFF:
		return "merge-no-ff"
	case git_commands.PullIntegrationRebase:
		return "rebase"
	case git_commands.PullIntegrationRebaseMerges:
		return "rebase-merges"
	case git_commands.PullIntegrationRebaseInteractive:
		return "rebase-interactive"
	default:
		panic("unknown current pull integration method")
	}
}

func (flow *branchSyncUpdateFlow) prepareIntegration(review git_commands.CurrentPullReview, targetOID string, requested git_commands.PullIntegrationMethod) error {
	if err := flow.currentModeValid(); err != nil {
		return err
	}
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		plan, err := flow.target.git.Sync.PrepareCurrentPullIntegration(review, targetOID, requested)
		if err != nil {
			if errors.Is(err, git_commands.ErrBranchSyncChanged) {
				flow.target.c.OnUIThread(func() error { return flow.reinspectCurrent() })
				return nil
			}
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.currentModeValid(); err != nil {
				return err
			}
			return flow.reviewIntegration(plan)
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) reviewIntegration(plan git_commands.CurrentPullIntegrationPlan) error {
	if err := flow.currentModeValid(); err != nil {
		return err
	}
	prompt := flow.integrationReviewPrompt(plan)
	flow.target.c.Confirm(types.ConfirmOpts{Title: flow.target.c.Tr.BranchSyncIntegrationReviewTitle, Prompt: prompt, HandleConfirm: func() error {
		if err := flow.currentModeValid(); err != nil {
			return err
		}
		return flow.reviewIgnoredIntegration(plan)
	}, HandleClose: flow.target.checkRepository})
	return nil
}

func (flow *branchSyncUpdateFlow) integrationReviewPrompt(plan git_commands.CurrentPullIntegrationPlan) string {
	prompt := flow.collectionContext() + fmt.Sprintf(flow.target.c.Tr.BranchSyncIntegrationReviewPrompt, plan.TargetOID(), plan.Name(), plan.SourceOID(), flow.integrationMethodLabel(plan))
	if plan.UsesConfiguredRebaseOptions() {
		prompt += flow.target.c.Tr.BranchSyncIntegrationRebaseOptions
	}
	if plan.Method() == git_commands.PullIntegrationConfiguredMerge {
		prompt += fmt.Sprintf(flow.target.c.Tr.BranchSyncIntegrationMergeOptions, plan.MergeOptions(), plan.ConfiguredMergeFF())
	}
	if plan.ReplayBaseOID() != "" && plan.ReplayBaseOID() != plan.TargetOID() {
		prompt += fmt.Sprintf(flow.target.c.Tr.BranchSyncIntegrationReplayBase, plan.ReplayBaseOID())
	}
	return prompt
}

func (flow *branchSyncUpdateFlow) reviewIgnoredIntegration(plan git_commands.CurrentPullIntegrationPlan) error {
	if err := flow.currentModeValid(); err != nil {
		return err
	}
	paths := plan.IgnoredFiles()
	if len(paths) == 0 {
		return flow.executeIntegration(plan)
	}
	items := []*types.MenuItem{{CommandID: "branch.pull-current.cancel-ignored-risk", Verb: "cancel", Label: flow.target.c.Tr.Cancel, OnPress: flow.target.checkRepository}, {CommandID: "branch.pull-current.approve-ignored-risk", Verb: "integrate", Label: flow.target.c.Tr.BranchSyncIntegrationApproveIgnored, OnPress: func() error { return flow.executeIntegration(plan.WithIgnoredFilesApproved()) }}}
	for _, path := range paths {
		quoted := fmt.Sprintf("%q", path)
		items = append(items, &types.MenuItem{CommandID: "branch.pull-current.inspect-ignored-path", Verb: "inspect", Label: quoted, Tooltip: quoted, OpensMenu: true, OnPress: func() error { return flow.reviewIgnoredIntegration(plan) }})
	}
	return flow.target.c.Menu(types.CreateMenuOptions{Title: flow.target.c.Tr.BranchSyncIntegrationIgnoredTitle, Prompt: flow.integrationReviewPrompt(plan) + flow.target.c.Tr.BranchSyncIntegrationIgnoredPrompt, HideCancel: true, Items: items, OnCancel: flow.target.checkRepository})
}

func (flow *branchSyncUpdateFlow) integrationMethodLabel(plan git_commands.CurrentPullIntegrationPlan) string {
	if plan.UsesConfiguredRebaseOptions() {
		return flow.target.c.Tr.BranchSyncIntegrationConfiguredRebase
	}
	switch plan.Method() {
	case git_commands.PullIntegrationFastForwardOnly:
		return flow.target.c.Tr.BranchSyncIntegrationFFOnly
	case git_commands.PullIntegrationMergeNoFF:
		return flow.target.c.Tr.BranchSyncIntegrationNoFF
	case git_commands.PullIntegrationRebase:
		return flow.target.c.Tr.BranchSyncIntegrationRebase
	case git_commands.PullIntegrationRebaseMerges:
		return flow.target.c.Tr.BranchSyncIntegrationMerges
	case git_commands.PullIntegrationRebaseInteractive:
		return flow.target.c.Tr.BranchSyncIntegrationInteractive
	case git_commands.PullIntegrationConfiguredMerge:
		return flow.target.c.Tr.BranchSyncIntegrationConfiguredMerge
	default:
		return flow.target.c.Tr.BranchSyncIntegrationMerge
	}
}

func (flow *branchSyncUpdateFlow) executeIntegration(plan git_commands.CurrentPullIntegrationPlan) error {
	if err := flow.currentModeValid(); err != nil {
		return err
	}
	if flow.collector != nil {
		return flow.collectIntegration(plan)
	}
	helper := flow.target.c.Helpers().MergeAndRebase
	return flow.target.runReviewedIntegration(plan, func(err error) error {
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			return flow.reprepareAfterChange(plan)
		}
		if errors.Is(err, git_commands.ErrPullIgnoredApprovalRequired) {
			return flow.reprepareRisk(plan)
		}
		return err
	}, func(result error) error {
		helper.RecordWhetherMergeOrRebaseStartedInLazygit()
		return helper.CheckForConflicts(result)
	})
}

func (flow *branchSyncUpdateFlow) reprepareAfterChange(plan git_commands.CurrentPullIntegrationPlan) error {
	review := plan.Review()
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		fresh, err := flow.target.git.Sync.PrepareCurrentPullIntegration(review, plan.TargetOID(), plan.RequestedMethod())
		flow.target.c.OnUIThread(func() error {
			if errors.Is(err, git_commands.ErrBranchSyncChanged) {
				return flow.reinspectCurrent()
			}
			if err != nil {
				return err
			}
			return flow.reviewIntegration(fresh)
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) reprepareRisk(plan git_commands.CurrentPullIntegrationPlan) error {
	return flow.prepareIntegration(plan.Review(), plan.TargetOID(), plan.RequestedMethod())
}

func (flow *branchSyncUpdateFlow) prepareCurrentFetch() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if flow.options.Remote == "." {
		return flow.prepareLocalUpdate()
	}
	name, sourceOID, options := flow.info.Name, flow.info.OID, flow.options
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(task gocui.Task) error {
		if err := flow.currentModeValid(); err != nil {
			return err
		}
		review, err := flow.target.git.Sync.PrepareCurrentPullReview(name, sourceOID, options.TrackingRef)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		if err != nil {
			return err
		}
		fetch, err := flow.target.git.Sync.PrepareCurrentBranchFetch(name, sourceOID, options)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.currentModeValid(); err != nil {
				return err
			}
			return flow.confirmCurrentFetch(review, fetch)
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) confirmCurrentFetch(review git_commands.CurrentPullReview, plan git_commands.BranchFetchPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	opts := plan.Options()
	prompt := fmt.Sprintf(flow.target.c.Tr.BranchSyncFetchReviewPrompt, opts.Remote, opts.RemoteRef, opts.TrackingRef, plan.Name(), utils.ShortHash(plan.SourceOID()))
	flow.target.c.Confirm(types.ConfirmOpts{Title: flow.target.c.Tr.BranchSyncFetchReviewTitle, Prompt: prompt, HandleConfirm: func() error {
		if err := flow.currentModeValid(); err != nil {
			return err
		}
		return flow.executeCurrentFetch(review, plan)
	}, HandleClose: flow.target.checkRepository})
	return nil
}

func (flow *branchSyncUpdateFlow) executeCurrentFetch(review git_commands.CurrentPullReview, plan git_commands.BranchFetchPlan) error {
	if err := flow.currentModeValid(); err != nil {
		return err
	}
	return flow.target.withOperation(types.ItemOperationPulling, flow.target.c.Tr.BranchSyncFetchStatus, func(task gocui.Task) error {
		cmd, err := flow.target.git.Sync.BranchFetchCommand(plan, task)
		if err != nil {
			if errors.Is(err, git_commands.ErrBranchSyncChanged) {
				flow.target.c.OnUIThread(func() error { return flow.reinspect() })
				return nil
			}
			return err
		}
		if err := cmd.Run(); err != nil {
			return err
		}
		tip, err := flow.target.git.Sync.FetchedBranchTip(plan)
		if err != nil {
			if errors.Is(err, git_commands.ErrBranchSyncChanged) {
				flow.target.c.OnUIThread(func() error { return flow.reinspect() })
				return nil
			}
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.currentModeValid(); err != nil {
				return err
			}
			return flow.chooseIntegration(review, tip)
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) reinspectCurrent() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if flow.collector != nil {
		return git_commands.ErrBranchSyncChanged
	}
	return flow.target.inspect(func(info git_commands.BranchSyncInfo) error {
		if info.HeadRef != info.FullRef {
			return fmt.Errorf(flow.target.c.Tr.BranchSyncNoLongerCurrent, info.Name)
		}
		flow.info = info
		flow.fetch, flow.update = nil, nil
		return flow.menu()
	})
}
