package git_commands

import (
	"fmt"
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

// keepAncestors runs the ancestors filter over commits in walk order, as
// loadCommitAncestors runs it over the lines of the log.
func keepAncestors(commits []*models.Commit, selected []string, limit bool) []*models.Commit {
	filter := newCommitAncestorFilter(selected, limit)
	kept := []*models.Commit{}
	for _, commit := range commits {
		if filter.done() {
			break
		}
		if filter.keep(commit) {
			kept = append(kept, commit)
		}
	}
	return kept
}

func TestKeepCommitAncestors(t *testing.T) {
	pool := &utils.StringPool{}

	// The unfiltered walk of two diverged branches, children first, with the
	// commits of both branches interleaved:
	//
	//	o has parent a1; m is a merge of a3 and b2
	//	a1 -> base, a2 -> a1, a3 -> a2; b1 -> base, b2 -> b1, b3 -> b2
	walk := []*models.Commit{
		testCommit(pool, "o", "a1"),
		testCommit(pool, "b3", "b2"),
		testCommit(pool, "m", "a3", "b2"),
		testCommit(pool, "a3", "a2"),
		testCommit(pool, "b2", "b1"),
		testCommit(pool, "a2", "a1"),
		testCommit(pool, "b1", "base"),
		testCommit(pool, "a1", "base"),
		testCommit(pool, "base"),
	}

	tests := []struct {
		name     string
		selected []string
		expected []string
	}{
		{
			name:     "selected commits on two diverged branches keep the unfiltered order",
			selected: []string{"a3", "b3"},
			expected: []string{"b3", "a3", "b2", "a2", "b1", "a1", "base"},
		},
		{
			name:     "a merge keeps the ancestors of both of its parents",
			selected: []string{"m"},
			expected: []string{"m", "a3", "b2", "a2", "b1", "a1", "base"},
		},
		{
			name:     "a root commit ends the walk and is the last kept commit",
			selected: []string{"a1"},
			expected: []string{"a1", "base"},
		},
		{
			name:     "a selected commit that is an ancestor of another selected one is kept once",
			selected: []string{"a3", "a1"},
			expected: []string{"a3", "a2", "a1", "base"},
		},
		{
			name:     "a selected commit with a parent outside the scope keeps only its own ancestors",
			selected: []string{"o"},
			expected: []string{"o", "a1", "base"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kept := keepAncestors(walk, test.selected, false)
			assert.Equal(t, test.expected, hashesOf(kept))
		})
	}
}

func TestCommitAncestorFilterDone(t *testing.T) {
	pool := &utils.StringPool{}
	a1 := testCommit(pool, "a1", "base")
	base := testCommit(pool, "base")

	// The walk is done once the last wanted commit is kept, so it does not read on
	// past the root of the selection.
	filter := newCommitAncestorFilter([]string{"a1"}, false)
	assert.False(t, filter.done())
	assert.True(t, filter.keep(a1))
	assert.False(t, filter.done())
	assert.True(t, filter.keep(base))
	assert.True(t, filter.done())
}

func TestKeepCommitAncestorsLimit(t *testing.T) {
	pool := &utils.StringPool{}

	// A linear history of commits c0 (root) to c349, newest first, with the
	// selection at the newest commit. Every commit is an ancestor.
	const length = 350
	chain := make([]*models.Commit, 0, length)
	for i := length - 1; i >= 0; i-- {
		if i == 0 {
			chain = append(chain, testCommit(pool, "c0"))
		} else {
			chain = append(chain, testCommit(pool, fmt.Sprintf("c%d", i), fmt.Sprintf("c%d", i-1)))
		}
	}
	selected := []string{fmt.Sprintf("c%d", length-1)}

	// The limit keeps the newest commitScopeLimit ancestors, and nothing older.
	limited := keepAncestors(chain, selected, true)
	assert.Len(t, limited, commitScopeLimit)
	assert.Equal(t, hashesOf(chain[:commitScopeLimit]), hashesOf(limited))

	unlimited := keepAncestors(chain, selected, false)
	assert.Len(t, unlimited, length)
}

func TestCommitScopeLog(t *testing.T) {
	ancestors := GetCommitsOptions{ScopeHashes: []string{"aaa", "bbb"}, RefName: "HEAD", Limit: true, All: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "date-order", all: true, limit: false},
		commitScopeLog(ancestors, "HEAD", "default"))
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "topo-order", all: true, limit: false},
		commitScopeLog(ancestors, "HEAD", "topo-order"))

	// Without --all, a hash scope walks the unfiltered revisions only: its selected
	// commits came from that walk. A branch scope adds its tips.
	notAll := GetCommitsOptions{ScopeHashes: []string{"aaa"}, RefName: "HEAD", Limit: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "topo-order", all: false, limit: false},
		commitScopeLog(notAll, "HEAD", "topo-order"))
	branch := GetCommitsOptions{ScopeHashes: []string{"aaa", "bbb"}, ScopeTips: []string{"aaa", "bbb"}, RefName: "HEAD"}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD", "aaa", "bbb"}, order: "topo-order", all: false, limit: false},
		commitScopeLog(branch, "HEAD", "topo-order"))
	branch.All = true
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "topo-order", all: true, limit: false},
		commitScopeLog(branch, "HEAD", "topo-order"))

	descendants := GetCommitsOptions{ScopeHashes: []string{"aaa"}, ScopeDescendants: true, RefName: "HEAD", Limit: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "date-order", all: true, limit: false},
		commitScopeLog(descendants, "HEAD", "default"))
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "author-date-order", all: true, limit: false},
		commitScopeLog(descendants, "HEAD", "author-date-order"))

	unscoped := GetCommitsOptions{RefName: "HEAD", All: true, Limit: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "default", all: true, limit: true},
		commitScopeLog(unscoped, "HEAD", "default"))
}
