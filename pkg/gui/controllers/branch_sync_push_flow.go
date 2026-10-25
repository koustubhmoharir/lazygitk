package controllers

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type branchSyncPushFlow struct {
	target *branchSyncTarget
	info   git_commands.BranchSyncInfo
	opts   git_commands.BranchPushOptions
	plan   *git_commands.BranchPushPlan
	// Stack review collects validated plans before any network push. Nil keeps
	// the ordinary one-branch execution path.
	acceptPlan    func(git_commands.BranchPushPlan) error
	reviewContext string
}

func (self *SyncController) pushBranchSource(source branchSyncSource) error {
	target := newBranchSyncTarget(self.c, source)
	return target.inspect(func(info git_commands.BranchSyncInfo) error {
		flow := newBranchSyncPushFlow(target, info)
		// Fork scope: pushing is one branch only. Wider-stack push preparation
		// remains deferred and must not be offered by the ordinary push menu.
		return flow.menu()
	})
}

func newBranchSyncPushFlow(target *branchSyncTarget, info git_commands.BranchSyncInfo) *branchSyncPushFlow {
	flow := &branchSyncPushFlow{target: target, info: info}
	flow.opts = git_commands.BranchPushOptions{Remote: info.PushRemote, DestinationRef: info.PushRef, RequireCurrent: !target.source.selected, Submodules: git_commands.BranchPushNoSubmoduleRecursion}
	if looksLikeBranchSyncRemoteURL(flow.opts.Remote) {
		flow.opts.Remote = ""
	}
	if !strings.HasPrefix(flow.opts.DestinationRef, "refs/heads/") || flow.opts.DestinationRef == "refs/heads/" {
		flow.opts.DestinationRef = "refs/heads/" + info.Name
	}
	return flow
}

func looksLikeBranchSyncRemoteURL(remote string) bool {
	return strings.Contains(remote, ":")
}

func (flow *branchSyncPushFlow) menu() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	tr := flow.target.c.Tr
	executeLabel := tr.BranchSyncPushThisBranch
	if flow.acceptPlan != nil {
		executeLabel = tr.BranchSyncStackPreparePush
	}
	items := []*types.MenuItem{
		{CommandID: "branch.push-selected.execute", Verb: "push", Label: executeLabel, OnPress: func() error {
			if flow.opts.Remote == "" {
				return flow.promptRemote(false)
			}
			return flow.prepare(false)
		}},
		{CommandID: "branch.push-selected.remote", Verb: "configure-push", Label: tr.BranchSyncChooseRemote, OpensMenu: true, OnPress: func() error { return flow.promptRemote(false) }},
		{CommandID: "branch.push-selected.destination", Verb: "configure-push", Label: tr.BranchSyncChooseDestination, OpensMenu: true, OnPress: flow.promptDestination},
		{CommandID: "branch.push-selected.set-upstream", Verb: "configure-push", Label: fmt.Sprintf("%s (%t)", tr.BranchSyncSetUpstream, flow.opts.SetUpstream), OnPress: func() error { flow.opts.SetUpstream = !flow.opts.SetUpstream; return flow.menu() }},
		{CommandID: "branch.push-selected.submodules", Verb: "configure-push", Label: fmt.Sprintf("%s (%t)", tr.BranchSyncCheckSubmodules, flow.opts.Submodules == git_commands.BranchPushCheckSubmodules), OnPress: func() error {
			if flow.opts.Submodules == git_commands.BranchPushCheckSubmodules {
				flow.opts.Submodules = git_commands.BranchPushNoSubmoduleRecursion
			} else {
				flow.opts.Submodules = git_commands.BranchPushCheckSubmodules
			}
			return flow.menu()
		}},
		{CommandID: "branch.push-selected.force-with-lease", Verb: "push", Label: tr.BranchSyncForceLease, OpensMenu: true, OnPress: flow.beginForce},
	}
	return flow.target.c.Menu(types.CreateMenuOptions{Title: tr.BranchSyncPushMenu, Prompt: flow.reviewContext + flow.menuSummary(), Items: items, OnCancel: flow.target.checkRepository})
}

func (flow *branchSyncPushFlow) menuSummary() string {
	remote, destination := flow.opts.Remote, flow.opts.DestinationRef
	if remote == "" {
		remote = flow.target.c.Tr.BranchSyncChooseRemote
	}
	destination = strings.TrimPrefix(destination, "refs/heads/")
	return fmt.Sprintf(flow.target.c.Tr.BranchSyncPushSummary, flow.info.Name, utils.ShortHash(flow.info.OID), remote, destination, flow.opts.SetUpstream, flow.opts.Submodules == git_commands.BranchPushCheckSubmodules)
}

func (flow *branchSyncPushFlow) promptRemote(forceContinuation bool) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	title := flow.target.c.Tr.BranchSyncRemotePrompt
	if looksLikeBranchSyncRemoteURL(flow.info.PushRemote) {
		title = flow.target.c.Tr.BranchSyncNamedRemoteRequired
	}
	flow.target.c.Prompt(types.PromptOpts{Title: title, InitialContent: flow.opts.Remote, PreserveWhitespace: true, AllowEditSuggestion: true, HandleClose: flow.target.checkRepository, HandleConfirm: func(remote string) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		flow.opts.Remote = remote
		flow.opts.LeaseRef = ""
		return flow.prepare(forceContinuation)
	}})
	return nil
}

func (flow *branchSyncPushFlow) promptDestination() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	initial := strings.TrimPrefix(flow.opts.DestinationRef, "refs/heads/")
	flow.target.c.Prompt(types.PromptOpts{Title: flow.target.c.Tr.BranchSyncDestinationPrompt, InitialContent: initial, PreserveWhitespace: true, AllowEditSuggestion: true, HandleClose: flow.target.checkRepository, HandleConfirm: func(value string) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		if strings.HasPrefix(value, "refs/heads/") {
			flow.opts.DestinationRef = value
		} else {
			flow.opts.DestinationRef = "refs/heads/" + value
		}
		flow.opts.LeaseRef = ""
		return flow.prepare(false)
	}})
	return nil
}

func (flow *branchSyncPushFlow) prepare(force bool) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	opts := flow.opts
	opts.ForceWithLease = force
	if !force {
		opts.LeaseRef = ""
	}
	return flow.prepareOptions(opts, flow.info.OID)
}

func (flow *branchSyncPushFlow) prepareOptions(opts git_commands.BranchPushOptions, sourceOID string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		plan, err := flow.target.git.Sync.PrepareBranchPush(flow.target.source.name, sourceOID, opts)
		if errors.Is(err, git_commands.ErrBranchPushUpstreamAmbiguous) {
			flow.target.c.OnUIThread(func() error { return flow.reviewAmbiguousUpstream(opts, sourceOID) })
			return nil
		}
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			flow.target.c.OnUIThread(func() error {
				return flow.target.inspect(func(fresh git_commands.BranchSyncInfo) error {
					flow.info = fresh
					flow.opts.LeaseRef = ""
					return flow.menu()
				})
			})
			return nil
		}
		if errors.Is(err, git_commands.ErrBranchPushLeaseChoice) {
			flow.target.c.OnUIThread(func() error { return flow.promptLease(opts, sourceOID) })
			return nil
		}
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			flow.plan = &plan
			return flow.review()
		})
		return nil
	})
}

// Refusing a tracking write is not permission to push. The alternative drops
// only SetUpstream and prepares another plan for a new, explicit review.
func (flow *branchSyncPushFlow) reviewAmbiguousUpstream(opts git_commands.BranchPushOptions, sourceOID string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.c.Menu(types.CreateMenuOptions{
		Title: flow.target.c.Tr.BranchSyncUpstreamAmbiguousTitle,
		Prompt: fmt.Sprintf(flow.target.c.Tr.BranchSyncUpstreamAmbiguousPrompt,
			flow.target.source.name, utils.ShortHash(sourceOID), opts.Remote, opts.DestinationRef),
		HideCancel: true,
		OnCancel:   flow.target.checkRepository,
		Items: []*types.MenuItem{
			{CommandID: "branch.push-selected.upstream-cancel", Verb: "cancel", Label: flow.target.c.Tr.Cancel, OnPress: flow.target.checkRepository},
			{CommandID: "branch.push-selected.without-upstream", Verb: "configure-push", Label: flow.target.c.Tr.BranchSyncPushWithoutTracking, OpensMenu: true, OnPress: func() error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				opts.SetUpstream = false
				flow.opts.SetUpstream = false
				return flow.prepareOptions(opts, sourceOID)
			}},
		},
	})
}

func (flow *branchSyncPushFlow) review() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	plan := *flow.plan
	opts := plan.Options()
	title := flow.target.c.Tr.BranchSyncReviewTitle
	prompt := flow.planReviewPrompt(plan)
	if opts.ForceWithLease {
		title = flow.target.c.Tr.BranchSyncLeaseConfirmTitle
		return flow.target.c.Menu(types.CreateMenuOptions{Title: title, Prompt: prompt, HideCancel: true, Items: []*types.MenuItem{
			{CommandID: "branch.push-selected.force-cancel", Verb: "cancel", Label: flow.target.c.Tr.Cancel, OnPress: func() error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				flow.opts.LeaseRef = ""
				return nil
			}},
			{CommandID: "branch.push-selected.force-confirm", Verb: "push", Label: flow.target.c.Tr.BranchSyncForceConfirm, OnPress: func() error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				return flow.execute(plan)
			}},
		}})
	}
	// Fork (roadmap 11): for an ordinary push the push menu's summary is the
	// review, so the "Push this branch" choice pushes without a second
	// confirmation. A confirmation remains when the stacked-push collection
	// flow needs it, or when setting upstream would replace different tracking.
	if flow.acceptPlan == nil && !branchPushReplacesTracking(plan) {
		return flow.execute(plan)
	}
	flow.target.c.Confirm(types.ConfirmOpts{Title: title, Prompt: prompt, HandleConfirm: func() error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		return flow.execute(plan)
	}, HandleClose: flow.target.checkRepository})
	return nil
}

// branchPushReplacesTracking reports whether setting upstream would replace
// existing tracking that is not exactly the new remote and destination.
func branchPushReplacesTracking(plan git_commands.BranchPushPlan) bool {
	opts := plan.Options()
	if !opts.SetUpstream {
		return false
	}
	previousMerges := plan.PreviousMergeRefs()
	if plan.PreviousRemote() == "" && len(previousMerges) == 0 {
		return false
	}
	sameMerge := len(previousMerges) == 1 && previousMerges[0] == opts.DestinationRef
	return plan.PreviousRemote() != opts.Remote || !sameMerge
}

func (flow *branchSyncPushFlow) upstreamReplacementSummary(plan git_commands.BranchPushPlan) string {
	remote := fmt.Sprintf("%q", plan.PreviousRemote())
	if strings.ContainsRune(plan.PreviousRemote(), ':') {
		remote = safeBranchSyncURL(plan.PreviousRemote())
	}
	return fmt.Sprintf(flow.target.c.Tr.BranchSyncUpstreamReplacement, remote, plan.PreviousMergeRefs())
}

func (flow *branchSyncPushFlow) beginForce() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if flow.opts.Remote == "" {
		return flow.promptRemote(true)
	}
	return flow.prepare(true)
}

func (flow *branchSyncPushFlow) promptLease(opts git_commands.BranchPushOptions, sourceOID string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	flow.target.c.Prompt(types.PromptOpts{Title: flow.target.c.Tr.BranchSyncLeasePrompt, InitialContent: opts.LeaseRef, PreserveWhitespace: true, AllowEditSuggestion: true, HandleClose: flow.target.checkRepository, HandleConfirm: func(ref string) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		opts.LeaseRef = ref
		return flow.prepareOptions(opts, sourceOID)
	}})
	return nil
}

func (flow *branchSyncPushFlow) execute(plan git_commands.BranchPushPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	run := func(task gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		cmd, err := flow.target.git.Sync.BranchPushCommand(plan, task)
		if errors.Is(err, git_commands.ErrBranchPushUpstreamAmbiguous) {
			opts := plan.Options()
			if opts.ForceWithLease {
				// Preserve the reference the user actually reviewed, including an
				// automatically resolved lease; the new review reads its fresh tip.
				opts.LeaseRef = plan.LeaseRef()
			}
			flow.target.c.OnUIThread(func() error { return flow.reviewAmbiguousUpstream(opts, plan.SourceOID()) })
			return nil
		}
		if errors.Is(err, git_commands.ErrBranchSyncChanged) {
			if flow.acceptPlan != nil {
				// A wider review is tied to its captured member tips. Do not
				// silently substitute another snapshot partway through it.
				return err
			}
			flow.target.c.OnUIThread(func() error {
				return flow.target.inspect(func(info git_commands.BranchSyncInfo) error {
					flow.info = info
					flow.opts.LeaseRef = ""
					return flow.menu()
				})
			})
			return nil
		}
		if errors.Is(err, git_commands.ErrBranchPushMultipleURLs) {
			urls := plan.ConfiguredPushURLs()
			redacted := make([]string, len(urls))
			for i, raw := range urls {
				redacted[i] = safeBranchSyncURL(raw)
			}
			flow.target.c.OnUIThread(func() error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				items := []*types.MenuItem{{CommandID: "branch.push-selected.multiple-urls-cancel", Verb: "cancel", Label: flow.target.c.Tr.Cancel, OnPress: flow.target.checkRepository}}
				for i, display := range redacted {
					items = append(items, &types.MenuItem{CommandID: fmt.Sprintf("branch.push-selected.destination-%d", i), Verb: "push-destination", Label: display, OnPress: func() error { return nil }})
				}
				items = append(items, &types.MenuItem{CommandID: "branch.push-selected.multiple-urls-approve", Verb: "push", Label: flow.target.c.Tr.BranchSyncApproveMultipleURLs, OnPress: func() error {
					if err := flow.target.checkRepository(); err != nil {
						return err
					}
					return flow.execute(plan.WithMultipleURLsApproved())
				}})
				return flow.target.c.Menu(types.CreateMenuOptions{Title: flow.target.c.Tr.BranchSyncMultipleURLsTitle, Prompt: flow.multipleURLsPrompt(plan), HideCancel: true, Items: items})
			})
			return nil
		}
		if err != nil {
			return err
		}
		if flow.acceptPlan != nil {
			flow.target.c.OnUIThread(func() error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				return flow.acceptPlan(plan)
			})
			return nil
		}
		return cmd.Run()
	}
	if flow.acceptPlan != nil {
		// Collection is read-only. Do not hold a one-branch push flag across
		// the callback into final batch approval; execution claims all flags.
		return flow.target.c.WithWaitingStatus(fmt.Sprintf(flow.target.c.Tr.BranchSyncStackReviewStatus, plan.Name()), run)
	}
	return flow.target.withOperation(types.ItemOperationPushing, flow.target.c.Tr.BranchSyncPushStatus, run)
}

func (flow *branchSyncPushFlow) multipleURLsPrompt(plan git_commands.BranchPushPlan) string {
	return flow.planReviewPrompt(plan) + "\n\n" + flow.target.c.Tr.BranchSyncMultipleURLsPrompt
}

func (flow *branchSyncPushFlow) planReviewPrompt(plan git_commands.BranchPushPlan) string {
	options := plan.Options()
	prompt := flow.reviewContext + fmt.Sprintf(flow.target.c.Tr.BranchSyncReviewPrompt, plan.Name(), plan.SourceOID(), options.Remote, options.DestinationRef, options.SetUpstream, options.Submodules == git_commands.BranchPushCheckSubmodules)
	if options.SetUpstream {
		prompt += flow.upstreamReplacementSummary(plan)
	}
	if options.ForceWithLease {
		prompt += fmt.Sprintf(flow.target.c.Tr.BranchSyncLeaseConfirmPrompt, plan.Name(), plan.SourceOID(), options.DestinationRef, plan.LeaseRef(), plan.LeaseOID())
	}
	return prompt
}

func safeBranchSyncURL(raw string) string {
	quoted := fmt.Sprintf("%q", strings.ToValidUTF8(raw, "�"))
	u, err := url.Parse(raw)
	if err == nil && ((u.Scheme == "" && !strings.ContainsAny(raw, "@:")) || (len(u.Scheme) == 1 && len(raw) > 2 && raw[1:3] == ":\\")) {
		return quoted
	}
	if err != nil || u.Scheme == "" || u.Host == "" {
		return `"[unrecognized remote URL]"`
	}
	if u.User != nil {
		u.User = url.User("[redacted]")
	}
	u.RawQuery, u.Fragment, u.RawFragment = "", "", ""
	return fmt.Sprintf("%q", u.String())
}
