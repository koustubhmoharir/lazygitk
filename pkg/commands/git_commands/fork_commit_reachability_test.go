package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func reachabilityTestCommands(t *testing.T, root string) *CommitCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	gitCommon := NewGitCommon(common.NewDummyCommon(), nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil)
	return NewCommitCommands(gitCommon)
}

func TestIsCommitInCommitsLog(t *testing.T) {
	root := initHistoricalRepo(t)
	base := branchPlacementTestCommit(t, root, "tracked", "base\n", "base")
	head := branchPlacementTestCommit(t, root, "tracked", "head\n", "head")

	// side is a branch that HEAD does not reach, and it has a tag.
	gitTest(t, root, "checkout", "-q", "-b", "side", base)
	side := branchPlacementTestCommit(t, root, "side", "side\n", "side")
	gitTest(t, root, "tag", "side-tag", side)
	gitTest(t, root, "checkout", "-q", "master")

	// kept is reached by a custom ref only, outside the usual namespaces.
	kept := gitTest(t, root, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "kept")
	gitTest(t, root, "update-ref", "refs/keep/only", kept)

	// dangling is a commit that no ref reaches.
	dangling := gitTest(t, root, "commit-tree", "HEAD^{tree}", "-p", base, "-m", "dangling")

	// lost was HEAD, and the reset moves HEAD away from it. The reflog still names it,
	// but no ref reaches it any more.
	lost := head
	gitTest(t, root, "reset", "-q", "--hard", base)

	scenarios := []struct {
		name       string
		hash       string
		wholeGraph bool
		expected   bool
	}{
		{name: "HEAD itself", hash: gitTest(t, root, "rev-parse", "HEAD"), wholeGraph: false, expected: true},
		{name: "ancestor of HEAD without whole graph", hash: base, wholeGraph: false, expected: true},
		{name: "ancestor of HEAD with whole graph", hash: base, wholeGraph: true, expected: true},
		{name: "branch tip not reached by HEAD, without whole graph", hash: side, wholeGraph: false, expected: false},
		{name: "branch tip not reached by HEAD, with whole graph", hash: side, wholeGraph: true, expected: true},
		{name: "commit only a custom ref reaches, with whole graph", hash: kept, wholeGraph: true, expected: true},
		{name: "commit only a custom ref reaches, without whole graph", hash: kept, wholeGraph: false, expected: false},
		{name: "commit no ref reaches", hash: dangling, wholeGraph: true, expected: false},
		{name: "commit only the reflog names", hash: lost, wholeGraph: true, expected: false},
		{name: "commit only the reflog names, without whole graph", hash: lost, wholeGraph: false, expected: false},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			inLog, err := reachabilityTestCommands(t, root).IsCommitInCommitsLog(s.hash, s.wholeGraph)
			assert.NoError(t, err)
			assert.Equal(t, s.expected, inLog)
		})
	}
}

func TestIsCommitInCommitsLogErrors(t *testing.T) {
	root := initHistoricalRepo(t)
	branchPlacementTestCommit(t, root, "tracked", "base\n", "base")
	commands := reachabilityTestCommands(t, root)

	_, err := commands.IsCommitInCommitsLog("not-a-hash", true)
	assert.Error(t, err)

	missing := "0123456789012345678901234567890123456789"
	_, err = commands.IsCommitInCommitsLog(missing, true)
	assert.Error(t, err)
}
