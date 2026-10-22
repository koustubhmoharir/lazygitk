package git_commands

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestRequireCleanRebaseStartWithRealGit(t *testing.T) {
	root := initHistoricalRepo(t)
	gitTest(t, root, "config", "status.showUntrackedFiles", "no")
	osCommand := oscommands.NewDummyOSCommand()
	common := NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil)

	assert.NoError(t, common.requireCleanRebaseStart())

	writeTestFile(t, root+"/.gitignore", []byte("/ignored\n"), 0o600)
	gitTest(t, root, "add", ".gitignore")
	gitTest(t, root, "commit", "-qm", "ignore generated file")
	writeTestFile(t, root+"/ignored", []byte("ignored\n"), 0o600)
	assert.NoError(t, common.requireCleanRebaseStart(), "ignored files should not block a rebase")

	writeTestFile(t, root+"/untracked", []byte("new\n"), 0o600)
	assert.NoError(t, common.requireCleanRebaseStart(), "normal untracked files should not block a rebase")
	gitTest(t, root, "clean", "-fd")

	writeTestFile(t, root+"/tracked", []byte("initial\n"), 0o600)
	gitTest(t, root, "add", "tracked")
	gitTest(t, root, "commit", "-qm", "initial")
	writeTestFile(t, root+"/tracked", []byte("changed\n"), 0o600)
	assert.ErrorIs(t, common.requireCleanRebaseStart(), ErrRebaseRequiresClean)
	gitTest(t, root, "checkout", "--", "tracked")

	writeTestFile(t, root+"/tracked", []byte("staged\n"), 0o600)
	gitTest(t, root, "add", "tracked")
	assert.ErrorIs(t, common.requireCleanRebaseStart(), ErrRebaseRequiresClean)
}

func TestRequireCleanRebaseStartDetectsDirtySubmoduleDespiteIgnoreConfig(t *testing.T) {
	child := initHistoricalRepo(t)
	writeTestFile(t, child+"/tracked", []byte("initial\n"), 0o600)
	gitTest(t, child, "add", "tracked")
	gitTest(t, child, "commit", "-qm", "initial")

	root := initHistoricalRepo(t)
	gitTest(t, root, "-c", "protocol.file.allow=always", "submodule", "add", child, "child")
	gitTest(t, root, "commit", "-qm", "add submodule")
	gitTest(t, root, "config", "submodule.child.ignore", "all")
	writeTestFile(t, root+"/child/untracked", []byte("new child file\n"), 0o600)
	osCommand := oscommands.NewDummyOSCommand()
	common := NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil)
	assert.NoError(t, common.requireCleanRebaseStart(), "untracked submodule content should not block a rebase")
	gitTest(t, root+"/child", "clean", "-fd")
	writeTestFile(t, root+"/child/tracked", []byte("modified\n"), 0o600)

	assert.ErrorIs(t, common.requireCleanRebaseStart(), ErrRebaseRequiresClean)
}

func TestPrepareInteractiveRebaseCommandCleanPolicy(t *testing.T) {
	statusArgs := []string{"status", "--porcelain=v1", "-z", "--untracked-files=no", "--ignore-submodules=untracked"}
	rebaseArgs := []string{"rebase", "--interactive", "--no-autostash", "--keep-empty", "--no-autosquash", "--rebase-merges", "base"}

	t.Run("dirty status rejects before constructing rebase command", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).ExpectGitArgs(statusArgs, " M tracked\x00", nil)
		instance := buildRebaseCommands(commonDeps{runner: runner})

		cmdObj, err := instance.PrepareInteractiveRebaseCommand(PrepareInteractiveRebaseCommandOpts{baseHashOrRoot: "base"})

		assert.Nil(t, cmdObj)
		assert.ErrorIs(t, err, ErrRebaseRequiresClean)
		runner.CheckForMissingCalls()
	})

	t.Run("reword editor returns the clean-check error without a command", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).ExpectGitArgs(statusArgs, " M tracked\x00", nil)
		instance := buildRebaseCommands(commonDeps{runner: runner})
		commits := []*models.Commit{models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: "abc123"})}

		cmdObj, err := instance.RewordCommitInEditor(commits, 0)

		assert.Nil(t, cmdObj)
		assert.ErrorIs(t, err, ErrRebaseRequiresClean)
		runner.CheckForMissingCalls()
	})

	t.Run("clean status starts with explicit no-autostash and disables lock retries", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).
			ExpectGitArgs(statusArgs, "", nil).
			ExpectFunc("rebase command", func(cmdObj *oscommands.CmdObj) bool {
				assert.Equal(t, rebaseArgs, cmdObj.Args()[1:])
				assert.True(t, cmdObj.LockRetriesDisabled())
				return true
			}, "", nil)
		instance := buildRebaseCommands(commonDeps{runner: runner})

		assert.NoError(t, instance.RebaseBranch("base"))
		runner.CheckForMissingCalls()
	})

	t.Run("continuation does not run clean check", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).ExpectGitArgs([]string{"rebase", "--continue"}, "", nil)
		instance := buildRebaseCommands(commonDeps{runner: runner})

		assert.NoError(t, instance.ContinueRebase())
		runner.CheckForMissingCalls()
	})

	t.Run("status errors propagate with context", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).ExpectGitArgs(statusArgs, "", assert.AnError)
		common := buildGitCommon(commonDeps{runner: runner})

		err := common.requireCleanRebaseStart()

		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "checking for changes before rebase"))
		runner.CheckForMissingCalls()
	})
}

func TestRebaseRecoveryCommandsDoNotRetry(t *testing.T) {
	instance := buildRebaseCommands(commonDeps{})
	for _, step := range []string{"continue", "skip", "abort"} {
		cmd := instance.GenericMergeOrRebaseActionCmdObj("rebase", step)
		assert.True(t, cmd.LockRetriesDisabled())
		assert.Equal(t, []string{"git", "rebase", "--" + step}, cmd.Args())
	}
	assert.False(t, instance.GenericMergeOrRebaseActionCmdObj("merge", "continue").LockRetriesDisabled())
}
