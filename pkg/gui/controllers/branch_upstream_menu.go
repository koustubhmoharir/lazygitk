package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type capturedUpstreamMenu struct {
	controller       *BranchesController
	target           *branchSyncTarget
	branchContext    types.Context
	branch           *models.Branch
	info             git_commands.BranchSyncInfo
	upstreamRef      string
	upstreamOID      string
	upstreamDisabled *types.DisabledReason
	baseBranch       string
	upstreamFlow     *branchUpstreamFlow
}

// showCapturedUpstreamOptions resolves branch-dependent menu data before the
// menu opens. Callbacks retain the branch and repository captured at entry.
func (self *BranchesController) showCapturedUpstreamOptions(name string) error {
	target := newBranchSyncTarget(self.c, branchSyncSource{name: name, selected: true})
	captured := &capturedUpstreamMenu{
		controller: self, target: target, branchContext: self.context(),
		upstreamFlow: newBranchUpstreamFlow(self.c, name),
	}
	mainBranches := self.c.Model().MainBranches
	if err := target.checkRepository(); err != nil {
		return err
	}
	return self.c.WithWaitingStatus(self.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := target.checkRepository(); err != nil {
			return err
		}
		info, err := target.git.Sync.InspectBranchSync(name)
		if err != nil {
			return err
		}
		captured.info = info
		captured.branch = &models.Branch{
			Name: info.Name, CommitHash: info.OID, Head: info.HeadRef == info.FullRef,
			UpstreamRemote: info.UpstreamRemote,
			UpstreamBranch: strings.TrimPrefix(info.UpstreamRef, "refs/heads/"),
		}
		captured.upstreamRef = info.UpstreamTrackingRef
		if captured.upstreamRef == "" {
			captured.upstreamDisabled = &types.DisabledReason{Text: self.c.Tr.UpstreamNotSetError}
		} else {
			captured.upstreamOID, err = target.git.Branch.ResolveBranchActionCommit(captured.upstreamRef)
			if errors.Is(err, git_commands.ErrBranchActionChanged) {
				// A missing native tracking ref retains the legacy absent-upstream
				// explanation while leaving set/unset available for repair.
				captured.upstreamDisabled = &types.DisabledReason{Text: self.c.Tr.UpstreamNotSetError}
			} else if err != nil {
				// Preserve native failures on dependent choices without hiding the
				// upstream metadata repair actions.
				captured.upstreamDisabled = &types.DisabledReason{Text: err.Error()}
			}
		}
		if mainBranches != nil {
			captured.baseBranch, err = target.git.Loaders.BranchLoader.GetBaseBranch(captured.branch, mainBranches)
			if err != nil {
				return err
			}
		}
		if err := target.checkRepository(); err != nil {
			return err
		}
		target.c.OnUIThread(func() error {
			if err := target.checkRepository(); err != nil {
				return err
			}
			return captured.menu()
		})
		return nil
	})
}

func (captured *capturedUpstreamMenu) checkRepository() error {
	return captured.target.checkRepository()
}

func (captured *capturedUpstreamMenu) menu() error {
	if err := captured.checkRepository(); err != nil {
		return err
	}
	c := captured.controller.c
	info, branch := captured.info, captured.branch
	upstreamLabel := c.Tr.UpstreamGenericName
	if captured.upstreamRef != "" {
		upstreamLabel = helpers.ShortBranchName(captured.upstreamRef)
	}
	upstreamReview := c.Tr.UpstreamGenericName
	if captured.upstreamOID != "" {
		upstreamReview = fmt.Sprintf("%q at %s", captured.upstreamRef, captured.upstreamOID)
	} else if captured.upstreamDisabled != nil {
		upstreamReview = captured.upstreamDisabled.Text
	}
	prompt := fmt.Sprintf(c.Tr.BranchUpstreamMenuPrompt, info.Name, info.OID, upstreamReview)

	baseLabel := captured.baseBranch
	var baseDisabled *types.DisabledReason
	if baseLabel == "" {
		baseLabel = c.Tr.CouldNotDetermineBaseBranch
		baseDisabled = &types.DisabledReason{Text: c.Tr.CouldNotDetermineBaseBranch}
	} else {
		baseLabel = helpers.ShortBranchName(baseLabel)
	}
	baseDivergenceLabel := utils.ResolvePlaceholderString(c.Tr.ViewDivergenceFromBaseBranch, map[string]string{"baseBranch": baseLabel})
	viewUpstream := &types.MenuItem{
		CommandID: "branch.view-upstream-divergence", Verb: "view-divergence",
		LabelColumns: []string{c.Tr.ViewDivergenceFromUpstream}, DisabledReason: captured.upstreamDisabled,
		OnPress: func() error {
			if err := captured.checkRepository(); err != nil {
				return err
			}
			return c.Helpers().SubCommits.ViewSubCommits(helpers.ViewSubCommitsOpts{
				Ref: branch, TitleRef: fmt.Sprintf("%s <-> %s", branch.RefName(), upstreamLabel),
				RefToShowDivergenceFrom: captured.upstreamRef, Context: captured.branchContext, ShowBranchHeads: false,
			})
		},
	}
	viewBase := &types.MenuItem{
		CommandID: "branch.view-base-divergence", Verb: "view-divergence",
		LabelColumns: []string{baseDivergenceLabel}, Keys: menuKey('b'), DisabledReason: baseDisabled,
		OnPress: func() error {
			if err := captured.checkRepository(); err != nil {
				return err
			}
			return c.Helpers().SubCommits.ViewSubCommits(helpers.ViewSubCommitsOpts{
				Ref: branch, TitleRef: fmt.Sprintf("%s <-> %s", branch.RefName(), baseLabel),
				RefToShowDivergenceFrom: captured.baseBranch, Context: captured.branchContext, ShowBranchHeads: false,
			})
		},
	}
	unsetUpstream := &types.MenuItem{
		CommandID: "branch.unset-upstream", Verb: "unset-upstream",
		LabelColumns: []string{c.Tr.UnsetUpstream}, Keys: menuKey('u'), OnPress: captured.upstreamFlow.unset,
	}
	if info.UpstreamRemote == "" || info.UpstreamRef == "" {
		unsetUpstream.DisabledReason = &types.DisabledReason{Text: c.Tr.UpstreamNotSetError}
	}
	setUpstream := &types.MenuItem{
		CommandID: "branch.set-upstream", Verb: "set-upstream",
		LabelColumns: []string{c.Tr.SetUpstream}, Keys: menuKey('s'), OnPress: captured.upstreamFlow.set,
	}
	resetLabel := utils.ResolvePlaceholderString(c.Tr.ViewUpstreamResetOptions, map[string]string{"upstream": upstreamLabel})
	resetTooltip := utils.ResolvePlaceholderString(c.Tr.ViewUpstreamResetOptionsTooltip, map[string]string{"upstream": upstreamLabel})
	rebaseLabel := utils.ResolvePlaceholderString(c.Tr.ViewUpstreamRebaseOptions, map[string]string{"upstream": upstreamLabel})
	rebaseTooltip := utils.ResolvePlaceholderString(c.Tr.ViewUpstreamRebaseOptionsTooltip, map[string]string{"upstream": upstreamLabel})
	resetUpstream := &types.MenuItem{
		CommandID: "branch.reset-selected-upstream-menu", Verb: "reset", LabelColumns: []string{resetLabel},
		OpensMenu: true, Tooltip: resetTooltip, Keys: menuKey('g'), DisabledReason: captured.upstreamDisabled,
		OnPress: func() error {
			return captured.review(func() error {
				return helpers.ResetCurrentToCapturedSource(c.HelperCommon, upstreamLabel, captured.upstreamRef)
			})
		},
	}
	rebaseUpstream := &types.MenuItem{
		CommandID: "branch.rebase-current-onto-selected-upstream", Verb: "rebase", LabelColumns: []string{rebaseLabel},
		OpensMenu: true, Tooltip: rebaseTooltip, Keys: menuKey('r'), DisabledReason: captured.upstreamDisabled,
		OnPress: func() error {
			return captured.review(func() error { return c.Helpers().MergeAndRebase.RebaseOntoRef(captured.upstreamRef) })
		},
	}
	return c.Menu(types.CreateMenuOptions{
		Title: c.Tr.BranchUpstreamOptionsTitle, Prompt: prompt,
		Items:    []*types.MenuItem{viewUpstream, viewBase, unsetUpstream, setUpstream, resetUpstream, rebaseUpstream},
		OnCancel: captured.checkRepository,
	})
}

// Reinspect exact source, checked-out HEAD, tracking ref and upstream OID
// before opening either mutating submenu.
func (captured *capturedUpstreamMenu) review(open func() error) error {
	if err := captured.checkRepository(); err != nil {
		return err
	}
	target := captured.target
	return target.c.WithWaitingStatus(target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := captured.checkRepository(); err != nil {
			return err
		}
		fresh, err := target.git.Sync.InspectBranchSync(captured.info.Name)
		if err != nil {
			return err
		}
		if fresh.OID != captured.info.OID || fresh.HeadRef != captured.info.HeadRef || fresh.HeadOID != captured.info.HeadOID || fresh.UpstreamTrackingRef != captured.upstreamRef || captured.upstreamRef == "" {
			return git_commands.ErrBranchActionChanged
		}
		freshOID, err := target.git.Branch.ResolveBranchActionCommit(captured.upstreamRef)
		if err != nil {
			return err
		}
		if freshOID != captured.upstreamOID {
			return git_commands.ErrBranchActionChanged
		}
		target.c.OnUIThread(func() error {
			if err := captured.checkRepository(); err != nil {
				return err
			}
			return open()
		})
		return nil
	})
}
