package git_commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// testCommit builds a commit with the given hash and parents, for the scope tests.
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

// descendantTestLines formats commits, newest first, as lines of the descendant walk,
// in the fields that parseHashAndParents reads, and returns them with the walk's commits.
func descendantTestLines(commits []*models.Commit) []string {
	lines := make([]string, 0, len(commits))
	for _, commit := range commits {
		lines = append(lines, fmt.Sprintf("%s\x001600000000\x00A\x00a@x\x00%s\x00\x00\x00msg",
			commit.Hash(), strings.Join(commit.Parents(), " ")))
	}
	return lines
}

// runDescendantWalk feeds lines, newest first, to a walk as loadCommitDescendants does.
// It returns the rows and the number of lines read, including the one that stopped it.
func runDescendantWalk(pool *utils.StringPool, lines []string, selected []string, reachable []string, limit bool) ([]*models.Commit, int) {
	walk := newDescendantWalk(selected, reachable)
	read := 0
	for _, line := range lines {
		read++
		if !walk.read(line) {
			break
		}
	}
	rows := walk.rows(limit, func(line string) *models.Commit {
		hash, parents, _ := parseHashAndParents(line)
		return testCommit(pool, hash, parents...)
	})
	return rows, read
}

func TestDescendantWalk(t *testing.T) {
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
	lines := descendantTestLines(history)

	tests := []struct {
		name      string
		selected  []string
		reachable []string
		expected  []string
	}{
		{
			name:      "a commit keeps itself and its descendants, not its ancestors",
			selected:  []string{"m1"},
			reachable: []string{"m1"},
			expected:  []string{"m", "f2", "f1", "m1"},
		},
		{
			name:      "a tip keeps only itself",
			selected:  []string{"f2"},
			reachable: []string{"f2"},
			expected:  []string{"m", "f2"},
		},
		{
			name:      "descendants of several selected commits are kept once each",
			selected:  []string{"m1", "o1"},
			reachable: []string{"m1", "o1"},
			expected:  []string{"m", "f2", "o1", "f1", "m1"},
		},
		{
			name:      "an unrelated commit with no selected ancestor is not kept",
			selected:  []string{"x"},
			reachable: []string{"x"},
			expected:  []string{"x"},
		},
		{
			name:      "the root keeps every commit that descends from it",
			selected:  []string{"base"},
			reachable: []string{"base"},
			expected:  []string{"m", "f2", "o1", "f1", "m1", "base"},
		},
		{
			name:      "an unreachable selection adds no rows",
			selected:  []string{"ghost"},
			reachable: []string{},
			expected:  []string{},
		},
		{
			name:      "an unreachable selection is ignored, and the reachable one is kept",
			selected:  []string{"ghost", "m1"},
			reachable: []string{"m1"},
			expected:  []string{"m", "f2", "f1", "m1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows, _ := runDescendantWalk(pool, lines, test.selected, test.reachable, false)
			assert.Equal(t, test.expected, hashesOf(rows))
		})
	}
}

// linearChainTestLines returns the lines of a linear history c0 (root) to c305, newest
// first, and the commits they are made from.
func linearChainTestLines(pool *utils.StringPool) ([]*models.Commit, []string) {
	const length = 306
	chain := make([]*models.Commit, 0, length)
	for i := length - 1; i >= 0; i-- {
		if i == 0 {
			chain = append(chain, testCommit(pool, "c0"))
		} else {
			chain = append(chain, testCommit(pool, fmt.Sprintf("c%d", i), fmt.Sprintf("c%d", i-1)))
		}
	}
	return chain, descendantTestLines(chain)
}

func TestDescendantWalkStopsEarly(t *testing.T) {
	pool := &utils.StringPool{}
	chain, lines := linearChainTestLines(pool)

	// c300 is 5 commits below the tip. Its descendants are c305 to c300, and the walk
	// stops at the line after c300, so 7 lines are read of 306.
	rows, read := runDescendantWalk(pool, lines, []string{"c300"}, []string{"c300"}, false)
	assert.Equal(t, hashesOf(chain[:6]), hashesOf(rows))
	assert.Equal(t, 7, read)

	// A root selection has to read the whole history.
	_, read = runDescendantWalk(pool, lines, []string{"c0"}, []string{"c0"}, false)
	assert.Equal(t, len(lines), read)

	// Without a reachable selection there is nothing to read.
	rows, read = runDescendantWalk(pool, lines, []string{"c300"}, []string{}, false)
	assert.Empty(t, rows)
	assert.Equal(t, 1, read)
}

func TestDescendantWalkUnrelatedPrefix(t *testing.T) {
	pool := &utils.StringPool{}

	// A long unrelated branch, newer than the selected commit in the walk, is read but
	// never kept. Newest first: u0 -> u1 -> ... -> u399, then a2 -> a1 -> base.
	var history []*models.Commit
	for i := 0; i < 400; i++ {
		var parents []string
		if i+1 < 400 {
			parents = []string{fmt.Sprintf("u%d", i+1)}
		}
		history = append(history, testCommit(pool, fmt.Sprintf("u%d", i), parents...))
	}
	history = append(history, testCommit(pool, "a2", "a1"), testCommit(pool, "a1", "base"), testCommit(pool, "base"))

	for _, limit := range []bool{false, true} {
		rows, _ := runDescendantWalk(pool, descendantTestLines(history), []string{"a1"}, []string{"a1"}, limit)
		assert.Equal(t, []string{"a2", "a1"}, hashesOf(rows))
	}
}

func TestDescendantWalkLimit(t *testing.T) {
	pool := &utils.StringPool{}
	chain, lines := linearChainTestLines(pool)

	// Limited, the rows are the newest commitScopeLimit descendants of the root: c305 to c6.
	limited, _ := runDescendantWalk(pool, lines, []string{"c0"}, []string{"c0"}, true)
	assert.Len(t, limited, commitScopeLimit)
	assert.Equal(t, hashesOf(chain[:commitScopeLimit]), hashesOf(limited))
	assert.Equal(t, "c305", limited[0].Hash())
	assert.Equal(t, "c6", limited[commitScopeLimit-1].Hash())

	// Unlimited, every commit is kept, the root included.
	unlimited, _ := runDescendantWalk(pool, lines, []string{"c0"}, []string{"c0"}, false)
	assert.Equal(t, hashesOf(chain), hashesOf(unlimited))
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

	// A descendant walk runs over --all and has no git limit: loadCommitDescendants
	// limits its rows.
	descendants := GetCommitsOptions{ScopeHashes: []string{"aaa"}, ScopeDescendants: true, RefName: "HEAD", Limit: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "date-order", all: true, limit: false},
		commitScopeLog(descendants, "HEAD", "default"))
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "author-date-order", all: true, limit: false},
		commitScopeLog(descendants, "HEAD", "author-date-order"))

	unscoped := GetCommitsOptions{RefName: "HEAD", All: true, Limit: true}
	assert.Equal(t, commitScopeLogArgs{revisions: []string{"HEAD"}, order: "default", all: true, limit: true},
		commitScopeLog(unscoped, "HEAD", "default"))
}
