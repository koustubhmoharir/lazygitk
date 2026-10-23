package helpers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type capturedRebaseFlow struct {
	repository   *capturedActionRepository
	source       git_commands.BranchActionTarget
	base         git_commands.BranchActionTarget
	baseDisabled *types.DisabledReason
	markedBase   string
}

type capturedRebaseChoice struct {
	id, label   string
	insertBreak bool
	source      git_commands.BranchActionTarget
	disabled    *types.DisabledReason
	key         rune
	tooltip     string
}

// RebaseCurrentOntoCapturedRef retains a literal source and marked base across
// mode, confirmation, ignored-file review and terminal execution.
func RebaseCurrentOntoCapturedRef(c *HelperCommon, ref string) error {
	flow := &capturedRebaseFlow{
		repository: captureActionRepository(c), markedBase: c.Modes().MarkedBaseCommit.GetHash(),
	}
	mainBranches := c.Model().MainBranches
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		var err error
		flow.source, err = flow.captureSource(ref)
		if err != nil {
			return err
		}
		if flow.source.HeadRef() == "" {
			return errors.New("cannot rebase with a detached HEAD")
		}
		if flow.source.HeadOID() == "" {
			return errors.New("cannot rebase an unborn branch")
		}
		flow.baseDisabled = &types.DisabledReason{Text: c.Tr.CouldNotDetermineBaseBranch}
		if mainBranches != nil {
			current := &models.Branch{Name: strings.TrimPrefix(flow.source.HeadRef(), "refs/heads/"), CommitHash: flow.source.HeadOID(), Head: true}
			baseRef, err := flow.repository.git.Loaders.BranchLoader.GetBaseBranch(current, mainBranches)
			if err != nil {
				return err
			}
			if baseRef != "" {
				flow.base, err = flow.captureSource(baseRef)
				if err != nil {
					flow.baseDisabled = &types.DisabledReason{Text: err.Error()}
				} else if flow.base.HeadRef() != flow.source.HeadRef() || flow.base.HeadOID() != flow.source.HeadOID() {
					return git_commands.ErrBranchActionChanged
				} else {
					flow.baseDisabled = nil
				}
			}
		}
		c.OnUIThread(flow.menu)
		return nil
	})
}

func (flow *capturedRebaseFlow) captureSource(ref string) (git_commands.BranchActionTarget, error) {
	oid, err := flow.repository.git.Branch.ResolveBranchActionCommit(ref)
	if err != nil {
		return git_commands.BranchActionTarget{}, err
	}
	return flow.repository.git.Branch.PrepareBranchActionTarget(ref, oid)
}

func (flow *capturedRebaseFlow) checkReview() error {
	if err := flow.repository.checkRepository(); err != nil {
		return err
	}
	if flow.repository.c.Modes().MarkedBaseCommit.GetHash() != flow.markedBase {
		return git_commands.ErrBranchActionChanged
	}
	return nil
}

func (flow *capturedRebaseFlow) title() string {
	template := flow.repository.c.Tr.RebasingTitle
	if flow.markedBase != "" {
		template = flow.repository.c.Tr.RebasingFromBaseCommitTitle
	}
	return utils.ResolvePlaceholderString(template, map[string]string{"checkedOutBranch": strings.TrimPrefix(flow.source.HeadRef(), "refs/heads/")})
}

func (flow *capturedRebaseFlow) prompt(source git_commands.BranchActionTarget, mode string) string {
	base := flow.markedBase
	if base == "" {
		base = flow.repository.c.Tr.BranchActionRebaseDefaultRange
	}
	return fmt.Sprintf(flow.repository.c.Tr.BranchActionRebaseReviewPrompt,
		strings.TrimPrefix(source.HeadRef(), "refs/heads/"), source.HeadOID(), ShortBranchName(source.SourceRef()), source.SourceRef(), source.SourceOID(), mode, base)
}

func (flow *capturedRebaseFlow) menu() error {
	if err := flow.checkReview(); err != nil {
		return err
	}
	c := flow.repository.c
	var selfDisabled *types.DisabledReason
	if flow.source.SourceRef() == flow.source.HeadRef() {
		selfDisabled = &types.DisabledReason{Text: c.Tr.CantRebaseOntoSelf}
	}
	label := ShortBranchName(flow.source.SourceRef())
	baseLabel := c.Tr.CouldNotDetermineBaseBranch
	if flow.baseDisabled == nil {
		baseLabel = ShortBranchName(flow.base.SourceRef())
	}
	choices := []capturedRebaseChoice{
		{
			id: "branch.rebase-current-onto-selected.simple", label: utils.ResolvePlaceholderString(c.Tr.SimpleRebase, map[string]string{"ref": label}),
			source: flow.source, disabled: selfDisabled, key: 's',
		},
		{
			id: "branch.rebase-current-onto-selected.interactive", label: utils.ResolvePlaceholderString(c.Tr.InteractiveRebase, map[string]string{"ref": label}),
			source: flow.source, disabled: selfDisabled, key: 'i', insertBreak: true, tooltip: c.Tr.InteractiveRebaseTooltip,
		},
		{
			id: "branch.rebase-current-onto-base", label: utils.ResolvePlaceholderString(c.Tr.RebaseOntoBaseBranch, map[string]string{"baseBranch": baseLabel}),
			source: flow.base, disabled: flow.baseDisabled, key: 'b', tooltip: c.Tr.RebaseOntoBaseBranchTooltip,
		},
	}
	items := make([]*types.MenuItem, 0, len(choices))
	for _, choice := range choices {
		verb := "rebase"
		if choice.insertBreak {
			verb = "interactive-rebase"
		}
		items = append(items, &types.MenuItem{
			CommandID: choice.id, Verb: verb, Label: choice.label, Keys: menuKey(choice.key),
			DisabledReason: choice.disabled, Tooltip: choice.tooltip, OpensMenu: true,
			OnPress: func() error { return flow.prepare(choice) },
		})
	}
	return c.Menu(types.CreateMenuOptions{Title: flow.title(), Prompt: flow.prompt(flow.source, c.Tr.RebaseBranch), Items: items, OnCancel: flow.repository.checkRepository})
}

func (flow *capturedRebaseFlow) prepare(choice capturedRebaseChoice) error {
	if err := flow.checkReview(); err != nil {
		return err
	}
	if choice.disabled != nil {
		return errors.New(choice.disabled.Text)
	}
	c := flow.repository.c
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		plan, err := flow.repository.git.Rebase.PrepareBranchActionRebase(choice.source, flow.markedBase, choice.insertBreak)
		if err != nil {
			return err
		}
		c.OnUIThread(func() error {
			if err := flow.checkReview(); err != nil {
				return err
			}
			review := &capturedRebaseReview{flow: flow, plan: plan, choice: choice}
			c.Confirm(types.ConfirmOpts{
				Title: flow.title(), Prompt: flow.prompt(choice.source, choice.label),
				HandleConfirm: review.reviewIgnored, HandleClose: flow.repository.checkRepository,
			})
			return nil
		})
		return nil
	})
}

type capturedRebaseReview struct {
	flow    *capturedRebaseFlow
	plan    git_commands.BranchActionRebasePlan
	choice  capturedRebaseChoice
	started bool
}

func (review *capturedRebaseReview) check() error {
	if err := review.flow.checkReview(); err != nil {
		return err
	}
	if review.started {
		return git_commands.ErrBranchActionChanged
	}
	return nil
}

func (review *capturedRebaseReview) reviewIgnored() error {
	if err := review.check(); err != nil {
		return err
	}
	if len(review.plan.IgnoredFiles()) == 0 {
		return review.execute(review.plan)
	}
	c := review.flow.repository.c
	items := []*types.MenuItem{
		{CommandID: review.choice.id + ".cancel-risk", Verb: "cancel", Label: c.Tr.Cancel, OnPress: review.flow.repository.checkRepository},
		{
			CommandID: review.choice.id + ".approve-ignored-overwrite", Verb: "rebase", Label: c.Tr.SelectedRebaseApproveIgnored, Keys: menuKey('p'),
			OnPress: func() error { return review.execute(review.plan.WithIgnoredFilesApproved()) },
		},
	}
	for _, path := range review.plan.IgnoredFiles() {
		quoted := fmt.Sprintf("%q", path)
		items = append(items, &types.MenuItem{
			CommandID: review.choice.id + ".inspect-ignored-path", Verb: "inspect", Label: quoted, Tooltip: quoted,
			OpensMenu: true, OnPress: review.reviewIgnored,
		})
	}
	return c.Menu(types.CreateMenuOptions{
		Title: c.Tr.SelectedRebaseIgnoredTitle, Prompt: c.Tr.SelectedRebaseIgnoredPrompt,
		HideCancel: true, Items: items, OnCancel: review.flow.repository.checkRepository,
	})
}

func capturedRebaseRefreshOptions(success bool) types.RefreshOptions {
	selection := types.KeepCommitSelectionByHash
	if success {
		selection = types.SelectHeadCommit
	}
	return types.RefreshOptions{
		Scope:          []types.RefreshableView{types.BRANCHES, types.COMMITS, types.SUB_COMMITS, types.FILES, types.REFLOG, types.WORKTREES, types.BISECT_INFO},
		BatchUIUpdates: true, BranchSelection: types.KeepBranchSelectionByName, CommitSelection: selection,
	}
}
