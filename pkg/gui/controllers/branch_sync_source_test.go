package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestBranchSyncSource(t *testing.T) {
	current := &models.Branch{Name: "current", Head: true}
	selected := &models.Branch{Name: "selected"}
	for _, tc := range []struct {
		name      string
		key       types.ContextKey
		selection []*models.Branch
		history   models.Ref
		current   *models.Branch
		want      branchSyncSource
		problem   branchSyncSourceProblem
	}{
		{name: "selected not current", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{selected}, current: current, want: branchSyncSource{name: "selected", selected: true}},
		{name: "explicitly selected current", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{current}, current: current, want: branchSyncSource{name: "current", selected: true}},
		{name: "no selection never falls back", key: context.LOCAL_BRANCHES_CONTEXT_KEY, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "range never picks first", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{selected, current}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "nil selection", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{nil}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "detached pseudo branch", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{{Name: "abcdef", DetachedHead: true}}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "local branch history", key: context.SUB_COMMITS_CONTEXT_KEY, selection: []*models.Branch{current}, history: selected, current: current, want: branchSyncSource{name: "selected", selected: true}},
		{name: "remote branch history", key: context.SUB_COMMITS_CONTEXT_KEY, history: &models.RemoteBranch{}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "tag history", key: context.SUB_COMMITS_CONTEXT_KEY, history: &models.Tag{}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "commit history", key: context.SUB_COMMITS_CONTEXT_KEY, history: historicalTestCommit("abcdef"), current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "missing history", key: context.SUB_COMMITS_CONTEXT_KEY, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "typed nil history", key: context.SUB_COMMITS_CONTEXT_KEY, history: (*models.Branch)(nil), current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "remote branch pane", key: context.REMOTE_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{selected}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "files ignore remembered selection", key: context.FILES_CONTEXT_KEY, selection: []*models.Branch{selected}, history: selected, current: current, want: branchSyncSource{name: "current"}},
		{name: "no current", key: context.FILES_CONTEXT_KEY, selection: []*models.Branch{selected}, problem: branchSyncNoCurrentLocalBranch},
		{name: "detached current", key: context.FILES_CONTEXT_KEY, current: &models.Branch{Name: "abcdef", Head: true, DetachedHead: true}, problem: branchSyncNoCurrentLocalBranch},
		{name: "first branch is not current", key: context.FILES_CONTEXT_KEY, current: selected, problem: branchSyncNoCurrentLocalBranch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, problem := resolveBranchSyncSource(tc.key, tc.selection, tc.history, tc.current)
			assert.Equal(t, tc.problem, problem)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBranchSyncSourceCapturesName(t *testing.T) {
	branch := &models.Branch{Name: "before"}
	source, problem := resolveBranchSyncSource(context.LOCAL_BRANCHES_CONTEXT_KEY, []*models.Branch{branch}, nil, nil)
	assert.Equal(t, branchSyncSourceOK, problem)
	branch.Name = "after"
	assert.Equal(t, "before", source.name)
}
