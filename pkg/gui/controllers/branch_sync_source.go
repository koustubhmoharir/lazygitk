package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// A source captures a literal local branch name, not a live model pointer.
// Selected distinguishes explicit branch-bearing contexts from a global
// current-branch action. Execution must still inspect fresh native Git state.
type branchSyncSource struct {
	name     string
	selected bool
}

type branchSyncSourceProblem uint8

const (
	branchSyncSourceOK branchSyncSourceProblem = iota
	branchSyncSelectOneLocalBranch
	branchSyncNoCurrentLocalBranch
)

// Resolve only the focused context: a remembered selection in a different
// pane must never redirect a global current-branch command.
func resolveBranchSyncSource(
	key types.ContextKey, selection []*models.Branch, history models.Ref, current *models.Branch,
) (branchSyncSource, branchSyncSourceProblem) {
	var branch *models.Branch
	selected := false
	switch key {
	case context.LOCAL_BRANCHES_CONTEXT_KEY:
		if len(selection) != 1 {
			return branchSyncSource{}, branchSyncSelectOneLocalBranch
		}
		branch, selected = selection[0], true
	case context.SUB_COMMITS_CONTEXT_KEY:
		branch, _ = history.(*models.Branch)
		selected = true
	case context.REMOTE_BRANCHES_CONTEXT_KEY:
		return branchSyncSource{}, branchSyncSelectOneLocalBranch
	default:
		if current == nil || !current.Head || current.DetachedHead || current.Name == "" {
			return branchSyncSource{}, branchSyncNoCurrentLocalBranch
		}
		branch = current
	}
	if branch == nil || branch.DetachedHead || branch.Name == "" {
		return branchSyncSource{}, branchSyncSelectOneLocalBranch
	}
	return branchSyncSource{name: branch.Name, selected: selected}, branchSyncSourceOK
}
