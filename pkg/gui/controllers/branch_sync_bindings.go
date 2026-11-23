package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Branch-bearing contexts own the selected target, even when their selection
// is invalid. Never fall back to HEAD for a remote branch or non-local history.
// Fork: the commits pane is branch-bearing only when the selected commit has
// local branch tips; otherwise its keys act on the checked-out branch.
func (self *SyncController) branchSyncSelectedContext() bool {
	side := self.c.Context().CurrentSide()
	if side == nil {
		return false
	}
	switch side.GetKey() {
	case context.LOCAL_BRANCHES_CONTEXT_KEY, context.REMOTE_BRANCHES_CONTEXT_KEY, context.SUB_COMMITS_CONTEXT_KEY:
		return true
	case context.LOCAL_COMMITS_CONTEXT_KEY:
		return len(self.selectedCommitTips()) > 0
	default:
		return false
	}
}

// Fork: selectedCommitTips returns the local branches whose tip is the single
// selected commit, in the order branchSyncCommitTips gives. A range or no
// selection has no tips.
func (self *SyncController) selectedCommitTips() []*models.Branch {
	commits, start, end := self.c.Contexts().LocalCommits.GetSelectedItems()
	if len(commits) != 1 || start != end || commits[0] == nil {
		return nil
	}
	return branchSyncCommitTips(commits[0].Hash(), self.c.Model().Branches)
}

// focusedBranchSyncSources returns the branch sources the focused pane targets:
// normally one; in the commits pane, every local branch tip of the selected
// commit (more than one needs a choice from the user).
func (self *SyncController) focusedBranchSyncSources() ([]branchSyncSource, branchSyncSourceProblem) {
	var current *models.Branch
	for _, branch := range self.c.Model().Branches {
		if branch != nil && branch.Head && !branch.DetachedHead {
			current = branch
			break
		}
	}
	var key types.ContextKey
	var selection []*models.Branch
	var history models.Ref
	var tips []*models.Branch
	if side := self.c.Context().CurrentSide(); side != nil {
		key = side.GetKey()
		switch key {
		case context.LOCAL_BRANCHES_CONTEXT_KEY:
			selection, _, _ = self.c.Contexts().Branches.GetSelectedItems()
		case context.SUB_COMMITS_CONTEXT_KEY:
			history = self.c.Contexts().SubCommits.GetRef()
		case context.LOCAL_COMMITS_CONTEXT_KEY:
			tips = self.selectedCommitTips()
		}
	}
	return resolveBranchSyncSource(key, selection, history, current, tips)
}

func (self *SyncController) branchSyncCommandID(verb string) string {
	if self.branchSyncSelectedContext() {
		return "branch." + verb + "-selected"
	}
	return "branch." + verb + "-current"
}

func (self *SyncController) branchSyncDescription(push bool) string {
	if self.branchSyncSelectedContext() {
		if push {
			return self.c.Tr.BranchSyncPushSelected
		}
		return self.c.Tr.BranchSyncPullSelected
	}
	if push {
		return self.c.Tr.BranchSyncPushCurrent
	}
	return self.c.Tr.BranchSyncPullCurrent
}

func (self *SyncController) branchSyncSourceDisabledReason(problem branchSyncSourceProblem) *types.DisabledReason {
	switch problem {
	case branchSyncSelectOneLocalBranch:
		return &types.DisabledReason{Text: self.c.Tr.BranchSyncSelectOneLocalBranch}
	case branchSyncNoCurrentLocalBranch:
		return &types.DisabledReason{Text: self.c.Tr.BranchSyncNoCurrentLocalBranch}
	default:
		return nil
	}
}

// branchSyncOperationDisabledReason reports a push or pull already running for
// the named local branch.
func (self *SyncController) branchSyncOperationDisabledReason(name string) *types.DisabledReason {
	for _, item := range []types.HasUrn{
		syncOperationItem(self.c.Git(), name),
		&models.Branch{Name: name},
	} {
		if self.c.State().GetItemOperation(item) != types.ItemOperationNone {
			return &types.DisabledReason{Text: self.c.Tr.CantPullOrPushSameBranchTwice}
		}
	}
	return nil
}

func (self *SyncController) getDisabledReasonForBranchSync() *types.DisabledReason {
	sources, problem := self.focusedBranchSyncSources()
	if disabled := self.branchSyncSourceDisabledReason(problem); disabled != nil {
		return disabled
	}
	if len(sources) != 1 {
		// Fork: a choice of several branch tips disables its own items instead.
		return nil
	}
	return self.branchSyncOperationDisabledReason(sources[0].name)
}
