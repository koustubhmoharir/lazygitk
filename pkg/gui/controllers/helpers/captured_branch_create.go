package helpers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type capturedBranchCreateFlow struct {
	repository *capturedActionRepository
	from       string
	label      string
	suggested  string
	target     git_commands.BranchActionTarget
	name       string
	plans      map[git_commands.BranchActionCreateTracking]git_commands.BranchActionCreatePlan
	planErrors map[git_commands.BranchActionCreateTracking]error
	submitted  bool
	started    bool
	cancelled  bool
}

// newCapturedBranch creates and checks out a branch from the reference selected
// when the action began. The repository is captured before suggestion lookup or
// any prompt callback can run.
func (self *RefsHelper) newCapturedBranch(from, label, suggested string) error {
	flow := &capturedBranchCreateFlow{repository: captureActionRepository(self.c), from: from, label: label, suggested: suggested}
	if flow.suggested == "" {
		var err error
		flow.suggested, err = self.getSuggestedBranchName()
		if err != nil {
			return err
		}
	}
	return flow.inspect()
}

// NewBranchWithNameFromRef creates and checks out a branch with a name already
// entered by the caller, while retaining the captured source and repository
// guards used by the interactive branch creation flow.
func (self *RefsHelper) NewBranchWithNameFromRef(from, label, name string) error {
	flow := &capturedBranchCreateFlow{
		repository: captureActionRepository(self.c), from: from, label: label,
		name: name, suggested: name, submitted: true,
	}
	return flow.inspect()
}

func (flow *capturedBranchCreateFlow) check() error {
	if err := flow.repository.checkRepository(); err != nil {
		return err
	}
	if flow.started || flow.cancelled {
		return git_commands.ErrBranchActionChanged
	}
	return nil
}

func (flow *capturedBranchCreateFlow) inspect() error {
	return flow.repository.c.WithWaitingStatus(flow.repository.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		oid, err := flow.repository.git.Branch.ResolveBranchActionCommit(flow.from)
		if err != nil {
			return err
		}
		flow.target, err = flow.repository.git.Branch.PrepareBranchActionTarget(flow.from, oid)
		if err != nil {
			return err
		}
		flow.repository.c.OnUIThread(func() error {
			if flow.submitted {
				return flow.review(flow.name)
			}
			return flow.prompt()
		})
		return nil
	})
}

func (flow *capturedBranchCreateFlow) prompt() error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	title := utils.ResolvePlaceholderString(c.Tr.NewBranchNameBranchOff, map[string]string{"branchName": flow.label})
	c.Prompt(types.PromptOpts{
		Title: title, InitialContent: flow.suggested, PreserveWhitespace: true,
		HandleClose: flow.cancel,
		HandleConfirm: func(name string) error {
			if err := flow.check(); err != nil {
				return err
			}
			if flow.submitted {
				return git_commands.ErrBranchActionChanged
			}
			flow.submitted = true
			return flow.review(name)
		},
	})
	return nil
}

func (flow *capturedBranchCreateFlow) review(name string) error {
	return flow.repository.c.WithWaitingStatus(flow.repository.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		if err := flow.repository.git.Branch.CheckBranchActionTarget(flow.target); err != nil {
			return err
		}
		flow.name = name
		flow.plans = make(map[git_commands.BranchActionCreateTracking]git_commands.BranchActionCreatePlan, 3)
		flow.planErrors = make(map[git_commands.BranchActionCreateTracking]error, 3)
		for _, policy := range []git_commands.BranchActionCreateTracking{
			git_commands.BranchActionCreateNoTracking,
			git_commands.BranchActionCreateConfiguredTracking,
			git_commands.BranchActionCreateDirectTracking,
		} {
			plan, err := flow.repository.git.Branch.PrepareBranchActionCreate(flow.target, name, git_commands.BranchActionCreateOptions{Tracking: policy, Checkout: true})
			if err != nil {
				if policy == git_commands.BranchActionCreateNoTracking {
					return err
				}
				flow.planErrors[policy] = err
				continue
			}
			flow.plans[policy] = plan
		}
		flow.repository.c.OnUIThread(func() error { return flow.policyMenu() })
		return nil
	})
}

func (flow *capturedBranchCreateFlow) policyMenu() error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	headRef := flow.target.HeadRef()
	if headRef == "" {
		headRef = c.Tr.BranchActionDetachedHead
	}
	if flow.target.HeadOID() == "" {
		headRef = c.Tr.BranchActionUnbornHead
	}
	prompt := fmt.Sprintf(c.Tr.BranchActionCreateReview, flow.name, flow.target.SourceRef(), flow.target.SourceOID(), headRef, flow.target.HeadOID())
	configured := git_commands.BranchActionCreateConfiguredTracking
	noTracking := git_commands.BranchActionCreateNoTracking
	first, second := configured, noTracking
	if flow.name != "" && flow.name != flow.suggested {
		first, second = noTracking, configured
	}
	items := []*types.MenuItem{
		flow.policyItem(prompt, first),
		flow.policyItem(prompt, second),
	}
	trackingDisabled := (*types.DisabledReason)(nil)
	if git_commands.IsFullObjectID(flow.target.SourceRef()) || strings.HasPrefix(flow.target.SourceRef(), "refs/tags/") || strings.HasPrefix(flow.target.SourceRef(), "refs/stash") || flow.target.SourceRef() == "HEAD" {
		trackingDisabled = &types.DisabledReason{Text: c.Tr.BranchActionCreateDirectUnavailable}
	}
	if err := flow.planErrors[git_commands.BranchActionCreateDirectTracking]; err != nil {
		trackingDisabled = &types.DisabledReason{Text: err.Error()}
	}
	items = append(items, &types.MenuItem{CommandID: "branch.create.direct-tracking", Verb: "create", Label: c.Tr.BranchActionCreateDirectTracking, DisabledReason: trackingDisabled, OnPress: func() error { return flow.execute(git_commands.BranchActionCreateDirectTracking) }})
	items = append(items, &types.MenuItem{CommandID: "branch.create.cancel", Verb: "cancel", Label: c.Tr.Cancel, OnPress: flow.cancel})
	return c.Menu(types.CreateMenuOptions{Title: c.Tr.BranchActionCreateTitle, Prompt: prompt, Items: items, HideCancel: true, OnCancel: flow.cancel})
}

func (flow *capturedBranchCreateFlow) policyItem(prompt string, policy git_commands.BranchActionCreateTracking) *types.MenuItem {
	c := flow.repository.c
	label := c.Tr.BranchActionCreateNoTracking
	id := "branch.create.no-tracking"
	if policy == git_commands.BranchActionCreateConfiguredTracking {
		label, id = c.Tr.BranchActionCreateConfiguredTracking, "branch.create.configured-tracking"
	}
	var disabled *types.DisabledReason
	if err := flow.planErrors[policy]; err != nil {
		disabled = &types.DisabledReason{Text: err.Error()}
	}
	return &types.MenuItem{CommandID: id, Verb: "create", Label: label, Tooltip: prompt, DisabledReason: disabled, OnPress: func() error { return flow.execute(policy) }}
}

func (flow *capturedBranchCreateFlow) cancel() error {
	if err := flow.repository.checkRepository(); err != nil {
		return err
	}
	if flow.started {
		return git_commands.ErrBranchActionChanged
	}
	if flow.cancelled {
		return nil
	}
	flow.cancelled = true
	return nil
}

func (flow *capturedBranchCreateFlow) execute(policy git_commands.BranchActionCreateTracking) error {
	if err := flow.check(); err != nil {
		return err
	}
	if _, ok := flow.plans[policy]; !ok {
		if err := flow.planErrors[policy]; err != nil {
			return err
		}
		return git_commands.ErrBranchActionChanged
	}
	flow.started = true
	repository := flow.repository
	c := repository.c
	return c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.CreatingBranchStatus}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		if err := repository.git.Branch.CheckBranchActionTarget(flow.target); err != nil {
			return flow.finish(err)
		}
		plan := flow.plans[policy]
		cmd, err := repository.git.Branch.BranchActionCreateCommand(plan)
		if err != nil {
			return flow.finish(err)
		}
		c.LogAction(c.Tr.Actions.CreateBranch)
		c.LogCommand(fmt.Sprintf("Creating branch %q from %s at %s", flow.name, flow.target.SourceRef(), flow.target.SourceOID()), false)
		if err = repository.checkRepository(); err == nil {
			err = cmd.Run()
		}
		return flow.finish(err)
	})
}

func (flow *capturedBranchCreateFlow) finish(result error) error {
	repository := flow.repository
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	selection := types.RefreshOptions{BranchSelection: types.KeepBranchSelectionByName, CommitSelection: types.KeepCommitSelectionByHash}
	if result == nil {
		selection.BranchSelection = types.SelectCheckedOutBranch
		selection.CommitSelection = types.SelectHeadCommit
		selection.SelectTopReflogCommit = true
	}
	repository.c.OnUIThread(func() error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		selection.Scope = []types.RefreshableView{types.COMMITS, types.BRANCHES, types.FILES, types.WORKTREES, types.REFLOG}
		selection.BatchUIUpdates = true
		selection.Then = func() error {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			if result == nil && repository.c.Context().Current() != repository.c.Contexts().Branches {
				repository.c.Context().Push(repository.c.Contexts().Branches, types.OnFocusOpts{})
			}
			return nil
		}
		repository.c.Refresh(selection)
		return nil
	})
	return result
}
