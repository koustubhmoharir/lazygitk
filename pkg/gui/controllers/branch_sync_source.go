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

// branchSyncCommitTips returns the local branches whose tip is the commit with
// the given full hash. The checked-out branch comes first when it is among
// them; the rest keep the order of branches. Detached pseudo-branches and
// branches without a name are ignored.
func branchSyncCommitTips(hash string, branches []*models.Branch) []*models.Branch {
	var checkedOut *models.Branch
	var others []*models.Branch
	for _, branch := range branches {
		if branch == nil || branch.DetachedHead || branch.Name == "" || branch.CommitHash != hash {
			continue
		}
		if branch.Head && checkedOut == nil {
			checkedOut = branch
		} else {
			others = append(others, branch)
		}
	}
	if checkedOut == nil {
		return others
	}
	return append([]*models.Branch{checkedOut}, others...)
}

// Resolve only the focused context: a remembered selection in a different
// pane must never redirect a global current-branch command. In the commits
// pane, tips are the ordered local branches at the single selected commit;
// they are empty for a range, no selection, or a commit with no branch tip.
func resolveBranchSyncSource(
	key types.ContextKey, selection []*models.Branch, history models.Ref, current *models.Branch, tips []*models.Branch,
) ([]branchSyncSource, branchSyncSourceProblem) {
	if key == context.LOCAL_COMMITS_CONTEXT_KEY && len(tips) > 0 {
		// Fork: a commit with branch tips targets those branches, each as if
		// selected in the Branches pane, even when one is the checked-out branch.
		sources := make([]branchSyncSource, 0, len(tips))
		for _, branch := range tips {
			sources = append(sources, branchSyncSource{name: branch.Name, selected: true})
		}
		return sources, branchSyncSourceOK
	}
	var branch *models.Branch
	selected := false
	switch key {
	case context.LOCAL_BRANCHES_CONTEXT_KEY:
		if len(selection) != 1 {
			return nil, branchSyncSelectOneLocalBranch
		}
		branch, selected = selection[0], true
	case context.SUB_COMMITS_CONTEXT_KEY:
		branch, _ = history.(*models.Branch)
		selected = true
	case context.REMOTE_BRANCHES_CONTEXT_KEY:
		return nil, branchSyncSelectOneLocalBranch
	default:
		if current == nil || !current.Head || current.DetachedHead || current.Name == "" {
			return nil, branchSyncNoCurrentLocalBranch
		}
		branch = current
	}
	if branch == nil || branch.DetachedHead || branch.Name == "" {
		return nil, branchSyncSelectOneLocalBranch
	}
	return []branchSyncSource{{name: branch.Name, selected: selected}}, branchSyncSourceOK
}
