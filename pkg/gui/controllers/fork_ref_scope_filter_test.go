package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestCommitHasRefScope(t *testing.T) {
	const hash = "aaaa"
	pool := &utils.StringPool{}
	commitWith := func(extraInfo string, tags ...string) *models.Commit {
		return models.NewCommit(pool, models.NewCommitOpts{Hash: hash, ExtraInfo: extraInfo, Tags: tags})
	}
	noBranches := []*models.Branch{}
	origin := []*models.Remote{{Name: "origin", Branches: []*models.RemoteBranch{{Name: "feat", RemoteName: "origin"}}}}

	tests := []struct {
		name     string
		commit   *models.Commit
		branches []*models.Branch
		remotes  []*models.Remote
		expected bool
	}{
		{
			name:     "a bare HEAD decoration does not count",
			commit:   commitWith("(HEAD)"),
			expected: false,
		},
		{
			name:     "origin/HEAD is not a remote branch of the model",
			commit:   commitWith("(origin/HEAD)"),
			remotes:  origin,
			expected: false,
		},
		{
			name:     "a local branch at the commit",
			commit:   commitWith("(HEAD -> main)"),
			branches: []*models.Branch{{Name: "main", CommitHash: hash}},
			expected: true,
		},
		{
			name:     "a remote branch decorating the commit",
			commit:   commitWith("(origin/feat)"),
			remotes:  origin,
			expected: true,
		},
		{
			name:     "a remote branch among other decorations",
			commit:   commitWith("(HEAD -> main, origin/feat, tag: v1)"),
			remotes:  origin,
			expected: true,
		},
		{
			name:     "a tag alone",
			commit:   commitWith("(tag: v1.0)", "v1.0"),
			remotes:  origin,
			expected: true,
		},
		{
			name:     "a remote branch that the model does not know",
			commit:   commitWith("(origin/other)"),
			remotes:  origin,
			expected: false,
		},
		{
			name:     "no decoration",
			commit:   commitWith(""),
			branches: noBranches,
			remotes:  origin,
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, commitHasRefScope(test.commit, test.branches, test.remotes))
		})
	}
}

func TestCommitDecorationNames(t *testing.T) {
	assert.Equal(t, []string{"main", "origin/main", "tag: v1"}, commitDecorationNames("(HEAD -> main, origin/main, tag: v1)"))
	assert.Equal(t, []string{"HEAD"}, commitDecorationNames("(HEAD)"))
	assert.Equal(t, []string{"origin/feat"}, commitDecorationNames("(origin/feat)"))
	assert.Nil(t, commitDecorationNames(""))
}
