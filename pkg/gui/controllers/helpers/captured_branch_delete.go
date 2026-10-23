package helpers

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type RemoteBranchDeleteSource struct {
	Ref, Remote string
}

type capturedBranchDeleteMode uint8

const (
	capturedDeleteLocal capturedBranchDeleteMode = iota
	capturedDeleteRemote
	capturedDeleteBoth
)

type capturedBranchDeleteFlow struct {
	repository    *capturedActionRepository
	context       git_commands.BranchActionDeleteContext
	mainRefs      []string
	initialLocal  git_commands.BranchActionLocalDeletePlan
	choices       []git_commands.BranchActionRemoteDeleteChoices
	choiceErrors  []error
	remoteHints   []string
	local         bool
	started       bool
	remoteTargets []git_commands.BranchActionRemoteDeleteSelection
	manual        []bool
}

type capturedBranchDeleteReview struct {
	flow       *capturedBranchDeleteFlow
	mode       capturedBranchDeleteMode
	localPlan  git_commands.BranchActionLocalDeletePlan
	remotePlan git_commands.BranchActionRemoteDeletePlan
}

func DeleteSelectedLocalBranches(c *HelperCommon, refs []string) error {
	return startCapturedBranchDelete(c, slices.Clone(refs), nil, true)
}

func DeleteSelectedRemoteBranches(c *HelperCommon, sources []RemoteBranchDeleteSource) error {
	return startCapturedBranchDelete(c, nil, slices.Clone(sources), false)
}

func startCapturedBranchDelete(c *HelperCommon, localRefs []string, sources []RemoteBranchDeleteSource, local bool) error {
	flow := &capturedBranchDeleteFlow{repository: captureActionRepository(c), local: local}
	refs := localRefs
	if !local {
		refs = make([]string, len(sources))
		flow.remoteHints = make([]string, len(sources))
		for i, source := range sources {
			refs[i], flow.remoteHints[i] = source.Ref, source.Remote
		}
	}
	if local && len(refs) == 0 || !local && len(sources) == 0 {
		return fmt.Errorf("select at least one branch")
	}
	if refs == nil {
		refs = []string{}
	}
	if local {
		branches := c.Model().MainBranches
		if branches != nil {
			flow.mainRefs = slices.Clone(branches.Get())
		}
	}
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		targets := make([]git_commands.BranchActionTarget, 0, len(refs))
		for _, ref := range refs {
			oid, err := flow.repository.git.Branch.ResolveBranchActionCommit(ref)
			if err != nil {
				return err
			}
			target, err := flow.repository.git.Branch.PrepareBranchActionTarget(ref, oid)
			if err != nil {
				return err
			}
			targets = append(targets, target)
		}
		var err error
		flow.context, err = flow.repository.git.Branch.PrepareBranchActionDeleteContext(targets)
		if err != nil {
			return err
		}
		if local {
			flow.remoteHints = make([]string, len(targets))
			flow.initialLocal, err = flow.repository.git.Branch.PrepareBranchActionLocalDelete(targets, flow.mainRefs, nil)
			if err != nil {
				return err
			}
			flow.choices = make([]git_commands.BranchActionRemoteDeleteChoices, len(targets))
			flow.choiceErrors = make([]error, len(targets))
			for i, target := range targets {
				flow.choices[i], flow.choiceErrors[i] = flow.repository.git.Sync.InspectBranchActionLocalDeleteUpstreams(target)
				flow.remoteHints[i] = flow.choices[i].Remote()
			}
		} else {
			flow.choices = make([]git_commands.BranchActionRemoteDeleteChoices, len(targets))
			flow.choiceErrors = make([]error, len(targets))
			for i, target := range targets {
				flow.choices[i], flow.choiceErrors[i] = flow.repository.git.Sync.InspectBranchActionRemoteDeleteChoices(target, flow.remoteHints[i])
			}
		}
		if err := flow.checkNative(); err != nil {
			return err
		}
		c.OnUIThread(flow.menu)
		return nil
	})
}

func (flow *capturedBranchDeleteFlow) check() error {
	if err := flow.repository.checkRepository(); err != nil {
		return err
	}
	if flow.started {
		return git_commands.ErrBranchActionChanged
	}
	return nil
}

func (flow *capturedBranchDeleteFlow) checkNative() error {
	if err := flow.check(); err != nil {
		return err
	}
	return flow.repository.git.Branch.CheckBranchActionDeleteContext(flow.context)
}

func (flow *capturedBranchDeleteFlow) prompt() string {
	lines := make([]string, 0, len(flow.context.Targets()))
	for _, target := range flow.context.Targets() {
		lines = append(lines, fmt.Sprintf("%s (%s)", target.SourceRef(), target.SourceOID()))
	}
	head := fmt.Sprintf(flow.repository.c.Tr.BranchActionDeleteSources, strings.Join(lines, "\n"))
	target := flow.context.Targets()[0]
	headName := target.HeadRef()
	if headName == "" {
		headName = flow.repository.c.Tr.BranchActionDetachedHead
	}
	if target.HeadOID() == "" {
		headName = flow.repository.c.Tr.BranchActionUnbornHead
	}
	head += fmt.Sprintf(flow.repository.c.Tr.BranchActionDeleteHead, headName, target.HeadOID())
	return head
}

func (flow *capturedBranchDeleteFlow) menu() error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	localDisabled := flow.localHasReservation()
	var remoteDisabled *types.DisabledReason
	if len(flow.context.Remotes()) == 0 {
		remoteDisabled = &types.DisabledReason{Text: c.Tr.BranchActionDeleteNoRemote}
	}
	items := []*types.MenuItem{
		{CommandID: "branch.delete-local", Verb: "delete", Label: lo.Ternary(len(flow.context.Targets()) == 1, c.Tr.DeleteLocalBranch, c.Tr.DeleteLocalBranches), Keys: menuKey('c'), DisabledReason: localDisabled, OnPress: func() error { return flow.choose(capturedDeleteLocal) }},
		{CommandID: "remote-branch.delete", Verb: "delete", Label: lo.Ternary(len(flow.context.Targets()) == 1, c.Tr.DeleteRemoteBranch, c.Tr.DeleteRemoteBranches), Keys: menuKey('r'), DisabledReason: remoteDisabled, OnPress: func() error { return flow.choose(capturedDeleteRemote) }},
		{CommandID: "branch.delete-local-and-remote", Verb: "delete", Label: lo.Ternary(len(flow.context.Targets()) == 1, c.Tr.DeleteLocalAndRemoteBranch, c.Tr.DeleteLocalAndRemoteBranches), Keys: menuKey('b'), DisabledReason: combineDisabled(localDisabled, remoteDisabled), OnPress: func() error { return flow.choose(capturedDeleteBoth) }},
	}
	if !flow.local {
		return flow.choose(capturedDeleteRemote)
	}
	name := ShortBranchName(flow.context.Targets()[0].SourceRef())
	title := c.Tr.DeleteBranchTitle
	if len(flow.context.Targets()) > 1 {
		title = c.Tr.DeleteBranchesTitle
	}
	title = utils.ResolvePlaceholderString(title, map[string]string{"selectedBranchName": name})
	return c.Menu(types.CreateMenuOptions{Title: title, Prompt: flow.prompt(), Items: items, OnCancel: flow.check})
}

func (flow *capturedBranchDeleteFlow) localHasReservation() *types.DisabledReason {
	for _, source := range flow.initialLocal.Sources() {
		if source.Info.Reservation.Kind != "" {
			return &types.DisabledReason{Text: fmt.Sprintf(flow.repository.c.Tr.BranchActionDeleteReserved, source.Info.FullRef, source.Info.Reservation.Kind, source.Info.Reservation.GitDir)}
		}
	}
	return nil
}

func combineDisabled(a, b *types.DisabledReason) *types.DisabledReason {
	if a != nil {
		return a
	}
	return b
}

func (flow *capturedBranchDeleteFlow) choose(mode capturedBranchDeleteMode) error {
	if err := flow.check(); err != nil {
		return err
	}
	if mode == capturedDeleteLocal || mode == capturedDeleteBoth {
		if reason := flow.localHasReservation(); reason != nil {
			return fmt.Errorf("%s", reason.Text)
		}
	}
	if mode == capturedDeleteRemote || mode == capturedDeleteBoth {
		if len(flow.context.Remotes()) == 0 {
			return errors.New(flow.repository.c.Tr.BranchActionDeleteNoRemote)
		}
		return flow.chooseRemote(mode, 0)
	}
	return flow.prepareLocal(mode)
}

func (flow *capturedBranchDeleteFlow) prepareLocal(mode capturedBranchDeleteMode) error {
	c := flow.repository.c
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.checkNative(); err != nil {
			return err
		}
		plan, err := flow.repository.git.Branch.PrepareBranchActionLocalDelete(flow.context.Targets(), flow.mainRefs, nil)
		if err != nil {
			return err
		}
		for _, source := range plan.Sources() {
			if source.Info.Reservation.Kind != "" {
				return fmt.Errorf(flow.repository.c.Tr.BranchActionDeleteReserved, source.Info.FullRef, source.Info.Reservation.Kind, source.Info.Reservation.GitDir)
			}
		}
		review := &capturedBranchDeleteReview{flow: flow, mode: mode, localPlan: plan}
		c.OnUIThread(func() error {
			if err := review.check(); err != nil {
				return err
			}
			unmerged := unmergedLocalSources(plan)
			if len(unmerged) == 0 {
				return review.execute()
			}
			prompt := flow.prompt() + "\n\n" + fmt.Sprintf(c.Tr.BranchActionDeleteForcePrompt, strings.Join(quotedSources(plan), "\n"))
			c.Confirm(types.ConfirmOpts{Title: c.Tr.ForceDeleteBranchTitle, Prompt: prompt, HandleClose: flow.check, HandleConfirm: func() error {
				if err := review.check(); err != nil {
					return err
				}
				review.localPlan = review.localPlan.WithForceApproved()
				return review.execute()
			}})
			return nil
		})
		return nil
	})
}

func unmergedLocalSources(plan git_commands.BranchActionLocalDeletePlan) []string {
	var refs []string
	for _, source := range plan.Sources() {
		if !source.Merged {
			refs = append(refs, source.Info.FullRef)
		}
	}
	return refs
}

func quotedSources(plan git_commands.BranchActionLocalDeletePlan) []string {
	var lines []string
	for _, source := range plan.Sources() {
		if !source.Merged {
			lines = append(lines, fmt.Sprintf("%q (%s)", source.Info.FullRef, source.Target.SourceOID()))
		}
	}
	return lines
}

func (review *capturedBranchDeleteReview) check() error {
	return review.flow.check()
}

func (flow *capturedBranchDeleteFlow) chooseRemote(mode capturedBranchDeleteMode, index int) error {
	if err := flow.check(); err != nil {
		return err
	}
	if index >= len(flow.context.Targets()) {
		return flow.prepareRemote(mode)
	}
	if index == 0 {
		flow.remoteTargets = make([]git_commands.BranchActionRemoteDeleteSelection, len(flow.context.Targets()))
		flow.manual = make([]bool, len(flow.context.Targets()))
	}
	choices := flow.choices[index]
	refs := choices.Refs()
	if flow.choiceErrors[index] != nil || choices.Remote() == "." || len(refs) == 0 {
		return flow.manualRemote(mode, index, flow.choiceErrors[index])
	}
	if len(refs) == 1 && strings.HasPrefix(refs[0], "refs/heads/") {
		flow.remoteTargets[index] = git_commands.BranchActionRemoteDeleteSelection{Target: flow.context.Targets()[index], Remote: choices.Remote(), RemoteRef: refs[0]}
		return flow.chooseRemote(mode, index+1)
	}
	items := make([]*types.MenuItem, 0, len(refs)+1)
	items = append(items, &types.MenuItem{CommandID: "branch.delete.cancel-remote-ref", Verb: "cancel", Label: flow.repository.c.Tr.Cancel, OnPress: flow.check})
	for _, ref := range refs {
		branch := strings.HasPrefix(ref, "refs/heads/")
		var disabled *types.DisabledReason
		if !branch {
			disabled = &types.DisabledReason{Text: flow.repository.c.Tr.BranchActionDeleteNotBranch}
		}
		items = append(items, &types.MenuItem{CommandID: "branch.delete.choose-remote-ref", Verb: "choose", Label: ref, DisabledReason: disabled, OpensMenu: true, OnPress: func() error {
			if err := flow.check(); err != nil {
				return err
			}
			flow.remoteTargets[index] = git_commands.BranchActionRemoteDeleteSelection{Target: flow.context.Targets()[index], Remote: choices.Remote(), RemoteRef: ref}
			return flow.chooseRemote(mode, index+1)
		}})
	}
	return flow.repository.c.Menu(types.CreateMenuOptions{Title: flow.repository.c.Tr.BranchActionDeleteChooseRemoteRef, Prompt: fmt.Sprintf("%s\n%s", flow.prompt(), choices.Remote()), Items: items, HideCancel: true, OnCancel: flow.check})
}

func (flow *capturedBranchDeleteFlow) manualRemote(mode capturedBranchDeleteMode, index int, reason error) error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	prompt := c.Tr.BranchActionDeleteLocalUpstream
	if reason != nil {
		prompt = reason.Error()
	}
	return c.Menu(types.CreateMenuOptions{Title: c.Tr.BranchActionDeleteManualTitle, Prompt: flow.prompt() + "\n\n" + prompt, HideCancel: true, OnCancel: flow.check, Items: []*types.MenuItem{
		{CommandID: "branch.delete.cancel-remote-manual", Verb: "cancel", Label: c.Tr.Cancel, OnPress: flow.check},
		{CommandID: "branch.delete.choose-remote-ref", Verb: "choose", Label: c.Tr.BranchActionDeleteManualAction, OpensMenu: true, OnPress: func() error { return flow.promptRemoteName(mode, index) }},
	}})
}

func (flow *capturedBranchDeleteFlow) promptRemoteName(mode capturedBranchDeleteMode, index int) error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	initial := ""
	if slices.Contains(flow.context.Remotes(), flow.remoteHints[index]) {
		initial = flow.remoteHints[index]
	}
	c.Prompt(types.PromptOpts{Title: c.Tr.BranchActionDeleteRemoteNameTitle, InitialContent: initial, PreserveWhitespace: true, HandleClose: flow.check, HandleConfirm: func(remote string) error {
		if err := flow.check(); err != nil {
			return err
		}
		if !slices.Contains(flow.context.Remotes(), remote) {
			return fmt.Errorf("remote %q is not configured", remote)
		}
		return flow.promptRemoteRef(mode, index, remote)
	}})
	return nil
}

func (flow *capturedBranchDeleteFlow) promptRemoteRef(mode capturedBranchDeleteMode, index int, remote string) error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	c.Prompt(types.PromptOpts{Title: c.Tr.BranchActionDeleteRemoteRefTitle, PreserveWhitespace: true, HandleClose: flow.check, HandleConfirm: func(ref string) error {
		if err := flow.check(); err != nil {
			return err
		}
		if !strings.HasPrefix(ref, "refs/heads/") {
			return errors.New(c.Tr.BranchActionDeleteNotBranch)
		}
		flow.remoteTargets[index] = git_commands.BranchActionRemoteDeleteSelection{Target: flow.context.Targets()[index], Remote: remote, RemoteRef: ref}
		flow.manual[index] = true
		return flow.chooseRemote(mode, index+1)
	}})
	return nil
}

func (flow *capturedBranchDeleteFlow) prepareRemote(mode capturedBranchDeleteMode) error {
	c := flow.repository.c
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.checkNative(); err != nil {
			return err
		}
		for i, choices := range flow.choices {
			if !flow.manual[i] && flow.choiceErrors[i] == nil && choices.Remote() != "." {
				if err := flow.repository.git.Sync.CheckBranchActionRemoteDeleteChoices(choices); err != nil {
					return err
				}
			}
		}
		remotePlan, err := flow.repository.git.Sync.PrepareBranchActionRemoteDelete(flow.remoteTargets)
		if err != nil {
			return err
		}
		var localPlan git_commands.BranchActionLocalDeletePlan
		if mode == capturedDeleteBoth {
			localPlan, err = flow.repository.git.Branch.PrepareBranchActionLocalDelete(flow.context.Targets(), flow.mainRefs, remotePlan.TrackingRefs())
			if err != nil {
				return err
			}
			for _, source := range localPlan.Sources() {
				if source.Info.Reservation.Kind != "" {
					return fmt.Errorf(c.Tr.BranchActionDeleteReserved, source.Info.FullRef, source.Info.Reservation.Kind, source.Info.Reservation.GitDir)
				}
			}
		}
		review := &capturedBranchDeleteReview{flow: flow, mode: mode, localPlan: localPlan, remotePlan: remotePlan}
		c.OnUIThread(func() error {
			if err := review.check(); err != nil {
				return err
			}
			prompt := flow.prompt() + fmt.Sprintf(c.Tr.BranchActionDeleteRemotePrompt, flow.remotePromptLines())
			unmerged := unmergedLocalSources(localPlan)
			if mode == capturedDeleteBoth && len(unmerged) > 0 {
				prompt += "\n\n" + fmt.Sprintf(c.Tr.BranchActionDeleteForcePrompt, strings.Join(quotedSources(localPlan), "\n"))
			}
			c.Confirm(types.ConfirmOpts{Title: flow.deleteTitle(mode), Prompt: prompt, HandleClose: flow.check, HandleConfirm: func() error {
				if err := review.check(); err != nil {
					return err
				}
				review.remotePlan = review.remotePlan.WithDeletionApproved()
				if mode == capturedDeleteBoth && len(unmerged) > 0 {
					review.localPlan = review.localPlan.WithForceApproved()
				}
				return review.execute()
			}})
			return nil
		})
		return nil
	})
}

func (flow *capturedBranchDeleteFlow) remotePromptLines() string {
	lines := make([]string, 0, len(flow.remoteTargets))
	for _, selection := range flow.remoteTargets {
		lines = append(lines, fmt.Sprintf("%q: %s -> %s", selection.Target.SourceRef(), selection.Remote, selection.RemoteRef))
	}
	return strings.Join(lines, "\n")
}

func (flow *capturedBranchDeleteFlow) deleteTitle(mode capturedBranchDeleteMode) string {
	c := flow.repository.c
	if len(flow.context.Targets()) > 1 {
		return c.Tr.DeleteBranchesTitle
	}
	if mode == capturedDeleteBoth {
		return c.Tr.DeleteLocalAndRemoteBranch
	}
	if mode == capturedDeleteLocal {
		return c.Tr.DeleteLocalBranch
	}
	return c.Tr.DeleteRemoteBranch
}
