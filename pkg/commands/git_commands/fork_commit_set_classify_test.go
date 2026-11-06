package git_commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifySelectedCommits(t *testing.T) {
	root := initHistoricalRepo(t)
	base := branchPlacementTestCommit(t, root, "tracked", "base\n", "base")
	head := branchPlacementTestCommit(t, root, "tracked", "head\n", "head")

	// side is a branch that HEAD does not reach.
	gitTest(t, root, "checkout", "-q", "-b", "side", base)
	side := branchPlacementTestCommit(t, root, "side", "side\n", "side")
	gitTest(t, root, "checkout", "-q", "master")

	// head is amended, so the old commit is reached by no ref and not by HEAD.
	gitTest(t, root, "commit", "-q", "--amend", "-m", "head amended")
	rewritten := head

	// A hash that never existed in this repository.
	fabricated := strings.Repeat("ab", 20)

	commands := reachabilityTestCommands(t, root)
	missing, unreachable, err := commands.ClassifySelectedCommits([]string{base, side, rewritten, fabricated})
	assert.NoError(t, err)
	assert.Equal(t, []string{fabricated}, missing, "an object that does not exist is missing")
	assert.Equal(t, []string{rewritten}, unreachable, "a commit that only the amended history replaced is unreachable")

	// base is reached by HEAD and side by its branch: neither is reported, so a
	// commit hidden by a filter or the limit is not noticed.
	missing, unreachable, err = commands.ClassifySelectedCommits([]string{base, side})
	assert.NoError(t, err)
	assert.Empty(t, missing)
	assert.Empty(t, unreachable)
}

func TestClassifySelectedCommitsBailsOutOnManyAbsentHashes(t *testing.T) {
	root := initHistoricalRepo(t)
	branchPlacementTestCommit(t, root, "tracked", "base\n", "base")

	hashes := make([]string, 0, maxClassifiedCommitSetHashes+1)
	for i := range maxClassifiedCommitSetHashes + 1 {
		hashes = append(hashes, fmt.Sprintf("%040x", i+1))
	}
	missing, unreachable, err := reachabilityTestCommands(t, root).ClassifySelectedCommits(hashes)
	assert.NoError(t, err)
	assert.Nil(t, missing, "past the limit nothing is classified")
	assert.Nil(t, unreachable)
}
