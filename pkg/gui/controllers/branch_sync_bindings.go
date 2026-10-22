package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Branch-bearing contexts own the selected target, even when their selection
// is invalid. Never fall back to HEAD for a remote branch or non-local history.
func (self *SyncController) branchSyncSelectedContext() bool {
	side := self.c.Context().CurrentSide()
	if side == nil {
		return false
	}
	switch side.GetKey() {
	case context.LOCAL_BRANCHES_CONTEXT_KEY, context.REMOTE_BRANCHES_CONTEXT_KEY, context.SUB_COMMITS_CONTEXT_KEY:
		return true
	default:
		return false
	}
}

func (self *SyncController) focusedBranchSyncSource() (branchSyncSource, branchSyncSourceProblem) {
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
	if side := self.c.Context().CurrentSide(); side != nil {
		key = side.GetKey()
		switch key {
		case context.LOCAL_BRANCHES_CONTEXT_KEY:
			selection, _, _ = self.c.Contexts().Branches.GetSelectedItems()
		case context.SUB_COMMITS_CONTEXT_KEY:
			history = self.c.Contexts().SubCommits.GetRef()
		}
	}
	return resolveBranchSyncSource(key, selection, history, current)
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

func (self *SyncController) getDisabledReasonForBranchSync() *types.DisabledReason {
	source, problem := self.focusedBranchSyncSource()
	if disabled := self.branchSyncSourceDisabledReason(problem); disabled != nil {
		return disabled
	}
	for _, item := range []types.HasUrn{
		syncOperationItem(self.c.Git(), source.name),
		&models.Branch{Name: source.name},
	} {
		if self.c.State().GetItemOperation(item) != types.ItemOperationNone {
			return &types.DisabledReason{Text: self.c.Tr.CantPullOrPushSameBranchTwice}
		}
	}
	return nil
}
