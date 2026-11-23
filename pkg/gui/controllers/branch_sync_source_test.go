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
		tips      []*models.Branch
		want      []branchSyncSource
		problem   branchSyncSourceProblem
	}{
		{name: "selected not current", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{selected}, current: current, want: []branchSyncSource{branchSyncSource{name: "selected", selected: true}}},
		{name: "explicitly selected current", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{current}, current: current, want: []branchSyncSource{branchSyncSource{name: "current", selected: true}}},
		{name: "no selection never falls back", key: context.LOCAL_BRANCHES_CONTEXT_KEY, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "range never picks first", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{selected, current}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "nil selection", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{nil}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "detached pseudo branch", key: context.LOCAL_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{{Name: "abcdef", DetachedHead: true}}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "local branch history", key: context.SUB_COMMITS_CONTEXT_KEY, selection: []*models.Branch{current}, history: selected, current: current, want: []branchSyncSource{branchSyncSource{name: "selected", selected: true}}},
		{name: "remote branch history", key: context.SUB_COMMITS_CONTEXT_KEY, history: &models.RemoteBranch{}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "tag history", key: context.SUB_COMMITS_CONTEXT_KEY, history: &models.Tag{}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "commit history", key: context.SUB_COMMITS_CONTEXT_KEY, history: historicalTestCommit("abcdef"), current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "missing history", key: context.SUB_COMMITS_CONTEXT_KEY, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "typed nil history", key: context.SUB_COMMITS_CONTEXT_KEY, history: (*models.Branch)(nil), current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "remote branch pane", key: context.REMOTE_BRANCHES_CONTEXT_KEY, selection: []*models.Branch{selected}, current: current, problem: branchSyncSelectOneLocalBranch},
		{name: "files ignore remembered selection", key: context.FILES_CONTEXT_KEY, selection: []*models.Branch{selected}, history: selected, current: current, want: []branchSyncSource{branchSyncSource{name: "current"}}},
		{name: "no current", key: context.FILES_CONTEXT_KEY, selection: []*models.Branch{selected}, problem: branchSyncNoCurrentLocalBranch},
		{name: "detached current", key: context.FILES_CONTEXT_KEY, current: &models.Branch{Name: "abcdef", Head: true, DetachedHead: true}, problem: branchSyncNoCurrentLocalBranch},
		{name: "first branch is not current", key: context.FILES_CONTEXT_KEY, current: selected, problem: branchSyncNoCurrentLocalBranch},
		{name: "commits without tips use current", key: context.LOCAL_COMMITS_CONTEXT_KEY, selection: []*models.Branch{selected}, current: current, want: []branchSyncSource{{name: "current"}}},
		{name: "commits no current and no tips", key: context.LOCAL_COMMITS_CONTEXT_KEY, current: nil, problem: branchSyncNoCurrentLocalBranch},
		{name: "commits one non-current tip", key: context.LOCAL_COMMITS_CONTEXT_KEY, current: current, tips: []*models.Branch{selected}, want: []branchSyncSource{{name: "selected", selected: true}}},
		{name: "commits one checked-out tip is selected", key: context.LOCAL_COMMITS_CONTEXT_KEY, current: current, tips: []*models.Branch{current}, want: []branchSyncSource{{name: "current", selected: true}}},
		{name: "commits several tips keep order", key: context.LOCAL_COMMITS_CONTEXT_KEY, current: current, tips: []*models.Branch{current, selected, {Name: "third"}}, want: []branchSyncSource{{name: "current", selected: true}, {name: "selected", selected: true}, {name: "third", selected: true}}},
		{name: "commits tips ignore remembered branch", key: context.LOCAL_COMMITS_CONTEXT_KEY, selection: []*models.Branch{selected}, current: current, tips: []*models.Branch{selected}, want: []branchSyncSource{{name: "selected", selected: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, problem := resolveBranchSyncSource(tc.key, tc.selection, tc.history, tc.current, tc.tips)
			assert.Equal(t, tc.problem, problem)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBranchSyncSourceCapturesName(t *testing.T) {
	branch := &models.Branch{Name: "before"}
	sources, problem := resolveBranchSyncSource(context.LOCAL_BRANCHES_CONTEXT_KEY, []*models.Branch{branch}, nil, nil, nil)
	assert.Equal(t, branchSyncSourceOK, problem)
	branch.Name = "after"
	assert.Equal(t, "before", sources[0].name)
}

func TestBranchSyncCommitTips(t *testing.T) {
	const hash = "aaaa"
	checkedOut := &models.Branch{Name: "main", Head: true, CommitHash: hash}
	first := &models.Branch{Name: "first", CommitHash: hash}
	second := &models.Branch{Name: "second", CommitHash: hash}
	elsewhere := &models.Branch{Name: "elsewhere", CommitHash: "bbbb"}
	detached := &models.Branch{Name: "abcdef", CommitHash: hash, DetachedHead: true, Head: true}
	unnamed := &models.Branch{Name: "", CommitHash: hash}
	for _, tc := range []struct {
		name     string
		branches []*models.Branch
		want     []*models.Branch
	}{
		{name: "no branches", want: nil},
		{name: "no tip at commit", branches: []*models.Branch{elsewhere}, want: nil},
		{name: "checked-out first, then model order", branches: []*models.Branch{first, checkedOut, second}, want: []*models.Branch{checkedOut, first, second}},
		{name: "no checked-out tip keeps model order", branches: []*models.Branch{second, elsewhere, first}, want: []*models.Branch{second, first}},
		{name: "detached and unnamed ignored", branches: []*models.Branch{nil, detached, unnamed, first}, want: []*models.Branch{first}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, branchSyncCommitTips(hash, tc.branches))
		})
	}
}
