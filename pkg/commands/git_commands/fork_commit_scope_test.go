package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// testCommit builds a commit with the given hash and parents, for keepCommitDescendants.
func testCommit(pool *utils.StringPool, hash string, parents ...string) *models.Commit {
	return models.NewCommit(pool, models.NewCommitOpts{Hash: hash, Parents: parents})
}

func hashesOf(commits []*models.Commit) []string {
	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		hashes = append(hashes, commit.Hash())
	}
	return hashes
}

func TestKeepCommitDescendants(t *testing.T) {
	pool := &utils.StringPool{}

	// A topo-order log of this history, newest first:
	//
	//	merge (m) has parents m1 and f2
	//	f2 -> f1 -> m1 -> base; o1 -> base; x is unrelated and has no parents
	history := []*models.Commit{
		testCommit(pool, "m", "m1", "f2"),
		testCommit(pool, "f2", "f1"),
		testCommit(pool, "o1", "base"),
		testCommit(pool, "f1", "m1"),
		testCommit(pool, "m1", "base"),
		testCommit(pool, "x"),
		testCommit(pool, "base"),
	}

	tests := []struct {
		name     string
		selected []string
		expected []string
	}{
		{
			name:     "a commit keeps itself and its descendants, not its ancestors",
			selected: []string{"m1"},
			expected: []string{"m", "f2", "f1", "m1"},
		},
		{
			name:     "a tip keeps only itself",
			selected: []string{"f2"},
			expected: []string{"m", "f2"},
		},
		{
			name:     "descendants of several selected commits are kept once each",
			selected: []string{"m1", "o1"},
			expected: []string{"m", "f2", "o1", "f1", "m1"},
		},
		{
			name:     "an unrelated commit with no selected ancestor is not kept",
			selected: []string{"x"},
			expected: []string{"x"},
		},
		{
			name:     "the root keeps every commit that descends from it",
			selected: []string{"base"},
			expected: []string{"m", "f2", "o1", "f1", "m1", "base"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selected := map[string]bool{}
			for _, hash := range test.selected {
				selected[hash] = true
			}
			kept := keepCommitDescendants(history, selected)
			assert.Equal(t, test.expected, hashesOf(kept))
		})
	}
}

func TestKeepCommitDescendantsOfNothing(t *testing.T) {
	pool := &utils.StringPool{}
	commits := []*models.Commit{testCommit(pool, "b", "a"), testCommit(pool, "a")}
	assert.Empty(t, keepCommitDescendants(commits, map[string]bool{}))
}

func TestCommitScopeLog(t *testing.T) {
	ancestors := GetCommitsOptions{ScopeHashes: []string{"aaa", "bbb"}, Limit: true, All: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"aaa", "bbb"}, order: "default", all: false, limit: true},
		commitScopeLog(ancestors, "HEAD", "default"))

	descendants := GetCommitsOptions{ScopeHashes: []string{"aaa"}, ScopeDescendants: true, Limit: true}
	assert.Equal(t, commitScopeLogArgs{order: "topo-order", all: true, limit: false},
		commitScopeLog(descendants, "HEAD", "default"))
	assert.Equal(t, commitScopeLogArgs{order: "author-date-order", all: true, limit: false},
		commitScopeLog(descendants, "HEAD", "author-date-order"))

	unscoped := GetCommitsOptions{RefName: "HEAD", All: true, Limit: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "default", all: true, limit: true},
		commitScopeLog(unscoped, "HEAD", "default"))
}
