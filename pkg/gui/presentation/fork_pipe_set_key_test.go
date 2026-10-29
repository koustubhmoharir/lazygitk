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
