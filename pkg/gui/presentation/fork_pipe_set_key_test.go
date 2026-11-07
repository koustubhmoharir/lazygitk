package presentation

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func pipeSetKeyTestCommits(pool *utils.StringPool, hashesAndParents ...[]string) []*models.Commit {
	commits := make([]*models.Commit, 0, len(hashesAndParents))
	for _, entry := range hashesAndParents {
		commits = append(commits, models.NewCommit(pool, models.NewCommitOpts{Hash: entry[0], Parents: entry[1:]}))
	}
	return commits
}

// A filtered list starts with the same commit and has the same length as the
// unfiltered one, but holds other commits: it must get its own pipes.
func TestPipeSetsOfListsWithTheSameHeadAndLength(t *testing.T) {
	pool := &utils.StringPool{}
	unfiltered := pipeSetKeyTestCommits(pool,
		[]string{"pskHead", "pskMid", "pskSide"},
		[]string{"pskSide", "pskBase"},
		[]string{"pskMid", "pskBase"},
	)
	filtered := pipeSetKeyTestCommits(pool,
		[]string{"pskHead", "pskMid", "pskSide"},
		[]string{"pskMid", "pskBase"},
		[]string{"pskBase"},
	)

	unfilteredPipes := loadPipesets(unfiltered)
	filteredPipes := loadPipesets(filtered)

	assert.NotEqual(t, commitListKey(unfiltered), commitListKey(filtered))
	assert.NotSame(t, &unfilteredPipes[0], &filteredPipes[0])
	assert.Equal(t, loadPipesets(filtered), filteredPipes)
}

func TestPipeSetsAreReusedForTheSameList(t *testing.T) {
	pool := &utils.StringPool{}
	list := pipeSetKeyTestCommits(pool, []string{"pskReuseHead", "pskReuseBase"}, []string{"pskReuseBase"})
	again := pipeSetKeyTestCommits(pool, []string{"pskReuseHead", "pskReuseBase"}, []string{"pskReuseBase"})

	assert.Same(t, &loadPipesets(list)[0], &loadPipesets(again)[0])
}

func TestDropPipeSetCacheComputesTheGraphAgain(t *testing.T) {
	pool := &utils.StringPool{}
	list := pipeSetKeyTestCommits(pool, []string{"pskDropHead", "pskDropBase"}, []string{"pskDropBase"})

	before := loadPipesets(list)
	DropPipeSetCache()
	after := loadPipesets(list)

	assert.NotSame(t, &before[0], &after[0])
	assert.Equal(t, before, after)
}

// The memo is tested by replacing a memoized digest: a lookup that returns the
// replaced value did not hash the list again.
func TestCommitListKeyIsComputedOncePerList(t *testing.T) {
	DropPipeSetCache()
	pool := &utils.StringPool{}
	list := pipeSetKeyTestCommits(pool, []string{"pskMemoOnce"}, []string{"pskMemoBase"})

	first := commitListKey(list)
	assert.Len(t, commitListMemo, 1)
	commitListMemo[0].key = "replaced"

	assert.Equal(t, "replaced", commitListKey(list))
	assert.NotEqual(t, "replaced", first)
}

func TestCommitListKeyOfEqualListsIsEqual(t *testing.T) {
	DropPipeSetCache()
	pool := &utils.StringPool{}
	list := pipeSetKeyTestCommits(pool, []string{"pskEqualA"}, []string{"pskEqualBase"})
	again := pipeSetKeyTestCommits(pool, []string{"pskEqualA"}, []string{"pskEqualBase"})

	assert.Equal(t, commitListKey(list), commitListKey(again))
}

// A subslice starts at the same commit as the list but has another length: it must
// not get the list's entry.
func TestCommitListKeyOfSubsliceWithOtherLengthIsDifferent(t *testing.T) {
	DropPipeSetCache()
	pool := &utils.StringPool{}
	list := pipeSetKeyTestCommits(pool,
		[]string{"pskSubA"}, []string{"pskSubB"}, []string{"pskSubC"})

	full := commitListKey(list)
	prefix := commitListKey(list[:2])

	assert.NotEqual(t, full, prefix)
	assert.Len(t, commitListMemo, 2)
}

func TestDropPipeSetCacheClearsTheListMemo(t *testing.T) {
	DropPipeSetCache()
	pool := &utils.StringPool{}
	list := pipeSetKeyTestCommits(pool, []string{"pskClearMemo"})
	commitListKey(list)
	assert.Len(t, commitListMemo, 1)

	DropPipeSetCache()

	assert.Empty(t, commitListMemo)
}

func TestCommitListMemoIsBounded(t *testing.T) {
	DropPipeSetCache()
	pool := &utils.StringPool{}
	lists := make([][]*models.Commit, 0, commitListMemoSize+1)
	for i := range commitListMemoSize + 1 {
		list := pipeSetKeyTestCommits(pool, []string{"pskBound" + string(rune('A'+i))})
		lists = append(lists, list)
		commitListKey(list)
	}

	assert.Len(t, commitListMemo, commitListMemoSize)
	for _, entry := range commitListMemo {
		assert.NotSame(t, &lists[0][0], &entry.commits[0])
	}
	assert.Same(t, &lists[1][0], &commitListMemo[0].commits[0])
	assert.Same(t, &lists[commitListMemoSize][0], &commitListMemo[commitListMemoSize-1].commits[0])
}
