package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type branchSyncUpdateFlow struct {
	target  *branchSyncTarget
	info    git_commands.BranchSyncInfo
	options git_commands.BranchFetchOptions
	fetch   *git_commands.BranchFetchPlan
	update  *git_commands.BranchFastForwardPlan
}

// updateBranchSource is an additive controller entry point for direct callers
// and focused tests. Default binding integration is owned by the caller.
func (self *SyncController) updateBranchSource(source branchSyncSource) error {
	target := newBranchSyncTarget(self.c, source)
	return target.inspect(func(info git_commands.BranchSyncInfo) error {
		// An explicitly selected current branch is still the current checkout.
		// This prototype only moves refs that are off this worktree.
		if info.HeadRef == info.FullRef {
			return fmt.Errorf(self.c.Tr.BranchSyncOffBranchOnly, source.name)
		}
		flow := &branchSyncUpdateFlow{target: target, info: info, options: branchSyncUpdateDefaults(info)}
		return flow.menu()
	})
}

func branchSyncUpdateDefaults(info git_commands.BranchSyncInfo) git_commands.BranchFetchOptions {
	options := git_commands.BranchFetchOptions{Remote: info.UpstreamRemote, RemoteRef: info.UpstreamRef, TrackingRef: info.UpstreamTrackingRef}
	if options.RemoteRef == "" {
		options.RemoteRef = info.FullRef
	}
	if looksLikeBranchSyncRemoteURL(options.Remote) {
		options.Remote = ""
	}
	return options
}

func (flow *branchSyncUpdateFlow) menu() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	tr := flow.target.c.Tr
	fetchLabel := tr.BranchSyncFetchForReview
	if flow.options.Remote == "." {
		fetchLabel = tr.BranchSyncReviewLocalUpdate
	}
	items := []*types.MenuItem{
		{CommandID: "branch.update-selected.fetch", Verb: "fetch", Label: fetchLabel, OnPress: flow.beginFetch},
		{CommandID: "branch.update-selected.remote", Verb: "configure-fetch", Label: tr.BranchSyncChooseRemote, OpensMenu: true, OnPress: flow.promptRemote},
		{CommandID: "branch.update-selected.upstream", Verb: "configure-fetch", Label: tr.BranchSyncChooseUpstreamBranch, OpensMenu: true, OnPress: flow.promptUpstream},
		{CommandID: "branch.update-selected.tracking", Verb: "configure-fetch", Label: tr.BranchSyncChooseTrackingRef, OpensMenu: true, OnPress: flow.promptTrackingRef},
	}
	return flow.target.c.Menu(types.CreateMenuOptions{Title: tr.BranchSyncUpdateMenu, Prompt: flow.summary(), Items: items, OnCancel: flow.target.checkRepository})
}

func (flow *branchSyncUpdateFlow) summary() string {
	return fmt.Sprintf(flow.target.c.Tr.BranchSyncUpdateSummary, flow.info.Name, utils.ShortHash(flow.info.OID), flow.options.Remote, flow.options.RemoteRef, flow.options.TrackingRef)
}

func (flow *branchSyncUpdateFlow) promptRemote() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	flow.target.c.Prompt(types.PromptOpts{Title: flow.target.c.Tr.BranchSyncRemotePrompt, InitialContent: flow.options.Remote, PreserveWhitespace: true, AllowEditSuggestion: true, HandleClose: flow.target.checkRepository, HandleConfirm: func(value string) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		if looksLikeBranchSyncRemoteURL(value) {
			return errors.New("enter a configured remote name, not a URL")
		}
		if value != flow.options.Remote {
			flow.options.TrackingRef = ""
		}
		flow.options.Remote = value
		return flow.menu()
	}})
	return nil
}

func (flow *branchSyncUpdateFlow) promptUpstream() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	flow.target.c.Prompt(types.PromptOpts{Title: flow.target.c.Tr.BranchSyncUpstreamBranchPrompt, InitialContent: flow.options.RemoteRef, PreserveWhitespace: true, AllowEditSuggestion: true, HandleClose: flow.target.checkRepository, HandleConfirm: func(value string) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		if !strings.HasPrefix(value, "refs/heads/") {
			value = "refs/heads/" + value
		}
		if value != flow.options.RemoteRef {
			flow.options.TrackingRef = ""
		}
		flow.options.RemoteRef = value
		return flow.menu()
	}})
	return nil
}

func (flow *branchSyncUpdateFlow) promptTrackingRef() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	flow.target.c.Prompt(types.PromptOpts{Title: flow.target.c.Tr.BranchSyncTrackingRefPrompt, InitialContent: flow.options.TrackingRef, PreserveWhitespace: true, AllowEditSuggestion: true, HandleClose: flow.target.checkRepository, HandleConfirm: func(value string) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		flow.options.TrackingRef = value
		return flow.menu()
	}})
	return nil
}

func (flow *branchSyncUpdateFlow) beginFetch() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if flow.options.Remote == "" {
		return flow.promptRemote()
	}
	if flow.options.RemoteRef == "" {
		return flow.promptUpstream()
	}
	if flow.options.Remote == "." {
		return flow.prepareLocalUpdate()
	}
	if flow.options.TrackingRef == "" {
		return flow.promptTrackingRef()
	}
	return flow.prepareFetch()
}

func (flow *branchSyncUpdateFlow) prepareLocalUpdate() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	name, sourceOID, upstreamRef := flow.info.Name, flow.info.OID, flow.options.RemoteRef
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		plan, err := flow.target.git.Sync.PrepareLocalBranchUpdate(name, sourceOID, upstreamRef)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		if errors.Is(err, git_commands.ErrBranchSyncDiverged) {
			flow.target.c.OnUIThread(func() error { return flow.showDiverged(name) })
			return nil
		}
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			flow.update = &plan
			return flow.reviewUpdate()
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) prepareFetch() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	name, sourceOID, options := flow.info.Name, flow.info.OID, flow.options
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		plan, err := flow.target.git.Sync.PrepareBranchFetch(name, sourceOID, options)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			flow.fetch = &plan
			return flow.reviewFetch()
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) showDiverged(name string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.c.Menu(types.CreateMenuOptions{
		Title:    flow.target.c.Tr.BranchSyncUpdateReviewTitle,
		Prompt:   fmt.Sprintf(flow.target.c.Tr.BranchSyncUpdateDiverged, name),
		Items:    []*types.MenuItem{{CommandID: "branch.update-selected.diverged-cancel", Verb: "cancel", Label: flow.target.c.Tr.Cancel, OnPress: flow.target.checkRepository}},
		OnCancel: flow.target.checkRepository,
	})
}

func (flow *branchSyncUpdateFlow) reviewFetch() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	plan := *flow.fetch
	opts := plan.Options()
	prompt := fmt.Sprintf(flow.target.c.Tr.BranchSyncFetchReviewPrompt, opts.Remote, opts.RemoteRef, opts.TrackingRef, plan.Name(), utils.ShortHash(plan.SourceOID()))
	flow.target.c.Confirm(types.ConfirmOpts{Title: flow.target.c.Tr.BranchSyncFetchReviewTitle, Prompt: prompt, HandleConfirm: func() error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		return flow.executeFetch(plan)
	}, HandleClose: flow.target.checkRepository})
	return nil
}

func (flow *branchSyncUpdateFlow) executeFetch(plan git_commands.BranchFetchPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.withOperation(types.ItemOperationPulling, flow.target.c.Tr.BranchSyncFetchStatus, func(task gocui.Task) error {
		cmd, err := flow.target.git.Sync.BranchFetchCommand(plan, task)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		if err != nil {
			return err
		}
		if err := cmd.Run(); err != nil {
			return err
		}
		update, err := flow.target.git.Sync.PrepareFetchedBranchUpdate(plan)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		if errors.Is(err, git_commands.ErrBranchSyncDiverged) {
			flow.target.c.OnUIThread(func() error { return flow.showDiverged(plan.Name()) })
			return nil
		}
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			flow.update = &update
			return flow.reviewUpdate()
		})
		return nil
	})
}

func (flow *branchSyncUpdateFlow) reviewUpdate() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	plan := *flow.update
	if !plan.NeedsUpdate() {
		return flow.target.c.Menu(types.CreateMenuOptions{Title: flow.target.c.Tr.BranchSyncUpdateReviewTitle, Prompt: fmt.Sprintf(flow.target.c.Tr.BranchSyncAlreadyAhead, plan.Name(), utils.ShortHash(plan.OldOID()), utils.ShortHash(plan.TargetOID())), Items: []*types.MenuItem{{CommandID: "branch.update-selected.noop-cancel", Verb: "cancel", Label: flow.target.c.Tr.Cancel, OnPress: flow.target.checkRepository}}, OnCancel: flow.target.checkRepository})
	}
	prompt := fmt.Sprintf(flow.target.c.Tr.BranchSyncUpdateReviewPrompt, plan.Name(), utils.ShortHash(plan.OldOID()), utils.ShortHash(plan.TargetOID()))
	flow.target.c.Confirm(types.ConfirmOpts{Title: flow.target.c.Tr.BranchSyncUpdateReviewTitle, Prompt: prompt, HandleConfirm: func() error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		return flow.applyUpdate(plan)
	}, HandleClose: flow.target.checkRepository})
	return nil
}

func (flow *branchSyncUpdateFlow) applyUpdate(plan git_commands.BranchFastForwardPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.withOperation(types.ItemOperationPulling, flow.target.c.Tr.BranchSyncUpdateStatus, func(gocui.Task) error {
		err := flow.target.git.Sync.ApplyBranchFastForward(plan)
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error { return flow.reinspect() })
			return nil
		}
		return err
	})
}

func (flow *branchSyncUpdateFlow) reinspect() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.inspect(func(info git_commands.BranchSyncInfo) error {
		if info.HeadRef == info.FullRef {
			return fmt.Errorf(flow.target.c.Tr.BranchSyncOffBranchOnly, info.Name)
		}
		flow.info = info
		flow.fetch, flow.update = nil, nil
		return flow.menu()
	})
}
