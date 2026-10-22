package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestRebaseCommitPreflight(t *testing.T) {
	t.Run("unstaged tracked changes beside staged changes reject amend before commit", func(t *testing.T) {
		root := initHistoricalRepo(t)
		writeTestFile(t, filepath.Join(root, "one"), []byte("base one\n"), 0o644)
		writeTestFile(t, filepath.Join(root, "two"), []byte("base two\n"), 0o644)
		gitTest(t, root, "add", "one", "two")
		gitTest(t, root, "commit", "-qm", "base")
		head := gitTest(t, root, "rev-parse", "HEAD")
		writeTestFile(t, filepath.Join(root, "one"), []byte("staged one\n"), 0o644)
		gitTest(t, root, "add", "one")
		writeTestFile(t, filepath.Join(root, "two"), []byte("unstaged two\n"), 0o644)
		commands := selectedCommitRebaseCommands(t, root)
		commit := models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: head})
		indexTree := gitTest(t, root, "write-tree")

		err := commands.AmendTo([]*models.Commit{commit}, 0)

		assert.ErrorIs(t, err, ErrRebaseWouldRetainTrackedChanges)
		assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, indexTree, gitTest(t, root, "write-tree"), "the staged index must remain unchanged")
		one, readErr := os.ReadFile(filepath.Join(root, "one"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		two, readErr := os.ReadFile(filepath.Join(root, "two"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		assert.Equal(t, "staged one\n", string(one))
		assert.Equal(t, "unstaged two\n", string(two))
		assert.Empty(t, gitTest(t, root, "stash", "list"))
	})

	t.Run("staged and unstaged portions of one file reject amend before commit", func(t *testing.T) {
		root := initHistoricalRepo(t)
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("base\n"), 0o644)
		gitTest(t, root, "add", "tracked")
		gitTest(t, root, "commit", "-qm", "base")
		head := gitTest(t, root, "rev-parse", "HEAD")
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged\n"), 0o644)
		gitTest(t, root, "add", "tracked")
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged\n"), 0o644)
		commands := selectedCommitRebaseCommands(t, root)
		commit := models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: head})
		indexTree := gitTest(t, root, "write-tree")

		err := commands.AmendTo([]*models.Commit{commit}, 0)

		assert.ErrorIs(t, err, ErrRebaseWouldRetainTrackedChanges)
		assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, indexTree, gitTest(t, root, "write-tree"), "the staged index must remain unchanged")
		contents, readErr := os.ReadFile(filepath.Join(root, "tracked"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		assert.Equal(t, "unstaged\n", string(contents))
		assert.Empty(t, gitTest(t, root, "stash", "list"))
	})

	t.Run("real Git allows staged changes and harmless untracked files", func(t *testing.T) {
		root := initHistoricalRepo(t)
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("base\n"), 0o644)
		gitTest(t, root, "add", "tracked")
		gitTest(t, root, "commit", "-qm", "base")
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged\n"), 0o644)
		gitTest(t, root, "add", "tracked")
		writeTestFile(t, filepath.Join(root, "untracked"), []byte("untracked\n"), 0o644)
		osCommand := oscommands.NewDummyOSCommand()
		common := NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil)

		assert.NoError(t, common.requireCleanAfterIndexCommitForRebase())
	})

	t.Run("staged index and harmless untracked files allow the fixup flow", func(t *testing.T) {
		hash := "abc123"
		runner := oscommands.NewFakeRunner(t).
			ExpectGitArgs([]string{"diff", "--quiet", "--no-ext-diff", "--ignore-submodules=untracked", "--"}, "", nil).
			ExpectGitArgs([]string{"commit", "--fixup=" + hash}, "", nil).
			ExpectGitArgs([]string{"rev-parse", "--verify", "HEAD"}, "fixup123\n", nil).
			ExpectGitArgs([]string{"status", "--porcelain=v1", "-z", "--untracked-files=no", "--ignore-submodules=untracked"}, "", nil).
			ExpectFunc("interactive rebase", func(cmdObj *oscommands.CmdObj) bool {
				assert.Equal(t, []string{"git", "rebase", "--interactive", "--no-autostash", "--keep-empty", "--no-autosquash", "--rebase-merges", "--root"}, cmdObj.Args())
				return true
			}, "", nil)
		instance := buildRebaseCommands(commonDeps{runner: runner})
		commit := models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: hash})

		assert.NoError(t, instance.AmendTo([]*models.Commit{commit}, 0))
		runner.CheckForMissingCalls()
	})

	t.Run("diff query failures propagate as errors", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).ExpectGitArgs([]string{"diff", "--quiet", "--no-ext-diff", "--ignore-submodules=untracked", "--"}, "", exec.ErrNotFound)
		common := buildGitCommon(commonDeps{runner: runner})

		err := common.requireCleanAfterIndexCommitForRebase()

		assert.ErrorIs(t, err, exec.ErrNotFound)
		runner.CheckForMissingCalls()
	})
}
