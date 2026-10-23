package git_commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func branchActionResetTarget(t *testing.T, root, ref, oid string) (*BranchCommands, BranchActionTarget) {
	t.Helper()
	c := transactionCommands(t, root)
	target, err := c.PrepareBranchActionTarget(ref, oid)
	resetNoError(t, err)
	return c, target
}

func TestBranchActionResetCommandModes(t *testing.T) {
	for _, mode := range []string{"soft", "mixed", "hard"} {
		t.Run(mode, func(t *testing.T) {
			root, source, current := transactionHistory(t)
			gitTest(t, root, "branch", "selected", source)
			gitTest(t, root, "tag", "selected", current)
			gitTest(t, root, "update-ref", "refs/vendor/unrelated", current)
			commands, target := branchActionResetTarget(t, root, "refs/heads/selected", source)
			tracked := filepath.Join(root, "tracked")
			writeTestFile(t, tracked, []byte("staged\x00bytes\n"), 0o644)
			gitTest(t, root, "add", "tracked")
			indexBefore, err := os.ReadFile(filepath.Join(root, ".git", "index"))
			resetNoError(t, err)
			writeTestFile(t, tracked, []byte("worktree\x00bytes\n"), 0o644)
			writeTestFile(t, filepath.Join(root, "untracked"), []byte("untracked bytes\n"), 0o600)
			// Command preparation and another fresh validation are read-only.
			cmd, err := commands.BranchActionResetCommand(target, mode)
			resetNoError(t, err)
			resetNoError(t, commands.CheckBranchActionTarget(target))
			assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
			assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
			assert.Equal(t, []byte("worktree\x00bytes\n"), mustRead(t, tracked))
			assert.Equal(t, []byte("untracked bytes\n"), mustRead(t, filepath.Join(root, "untracked")))

			resetNoError(t, cmd.Run())
			assert.Equal(t, source, gitTest(t, root, "rev-parse", "HEAD"))
			assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/heads/selected"))
			assert.Equal(t, current, gitTest(t, root, "rev-parse", "refs/tags/selected"))
			assert.Equal(t, current, gitTest(t, root, "rev-parse", "refs/vendor/unrelated"))
			switch mode {
			case "soft":
				assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
				assert.Equal(t, []byte("worktree\x00bytes\n"), mustRead(t, tracked))
			case "mixed":
				assert.Equal(t, gitTest(t, root, "rev-parse", source+"^{tree}"), gitTest(t, root, "write-tree"))
				assert.Equal(t, []byte("worktree\x00bytes\n"), mustRead(t, tracked))
			case "hard":
				assert.Equal(t, gitTest(t, root, "rev-parse", source+"^{tree}"), gitTest(t, root, "write-tree"))
				assert.Equal(t, []byte("one\n"), mustRead(t, tracked))
			}
			assert.Equal(t, []byte("untracked bytes\n"), mustRead(t, filepath.Join(root, "untracked")))
			assert.Empty(t, gitTest(t, root, "stash", "list"), "reset must not create a stash")
		})
	}
}

func TestBranchActionResetCommandRejectsInvalidAndStaleTargets(t *testing.T) {
	root, first, second := transactionHistory(t)
	gitTest(t, root, "branch", "reset-source", first)
	commands, target := branchActionResetTarget(t, root, "refs/heads/reset-source", first)
	for _, mode := range []string{"", "keep", "HARD", "--hard"} {
		_, err := commands.BranchActionResetCommand(target, mode)
		assert.Error(t, err)
	}
	_, err := commands.BranchActionResetCommand(BranchActionTarget{}, "hard")
	assert.Error(t, err)
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "refs/heads/master"))
	assert.NoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("builder baseline\n"), 0o644))
	gitTest(t, root, "add", "tracked")
	indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
	_, err = commands.BranchActionResetCommand(target, "hard")
	resetNoError(t, err)
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, "builder baseline\n", string(mustRead(t, filepath.Join(root, "tracked"))))
	gitTest(t, root, "update-ref", "refs/heads/reset-source", second, first)
	_, err = commands.BranchActionResetCommand(target, "hard")
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, "builder baseline\n", string(mustRead(t, filepath.Join(root, "tracked"))))

	// A second backend instance for the same repository must not accept the
	// first instance's captured target.
	assert.Error(t, transactionCommands(t, root).CheckBranchActionTarget(target))
	_, err = transactionCommands(t, root).BranchActionResetCommand(target, "hard")
	assert.Error(t, err)
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, "builder baseline\n", string(mustRead(t, filepath.Join(root, "tracked"))))

	// Changing current HEAD after review also refuses before creating a command.
	root, first, _ = transactionHistory(t)
	gitTest(t, root, "branch", "reset-source", first)
	commands, target = branchActionResetTarget(t, root, "refs/heads/reset-source", first)
	third := branchPlacementTestCommit(t, root, "third", "three\n", "third")
	_, err = commands.BranchActionResetCommand(target, "hard")
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assert.Equal(t, third, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, third, gitTest(t, root, "rev-parse", "refs/heads/master"))
}

func TestBranchActionResetCommandDetachedUnbornAndMerge(t *testing.T) {
	t.Run("detached", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		gitTest(t, root, "branch", "reset-source", first)
		gitTest(t, root, "checkout", "--detach", "-q", second)
		commands, target := branchActionResetTarget(t, root, "refs/heads/reset-source", first)
		cmd, err := commands.BranchActionResetCommand(target, "mixed")
		resetNoError(t, err)
		resetNoError(t, cmd.Run())
		assert.Empty(t, gitTest(t, root, "branch", "--show-current"))
		assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"))
	})
	t.Run("unborn", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "branch", "reset-source", first)
		gitTest(t, root, "checkout", "--orphan", "unborn")
		gitTest(t, root, "rm", "-qf", "tracked")
		commands, target := branchActionResetTarget(t, root, "refs/heads/reset-source", first)
		cmd, err := commands.BranchActionResetCommand(target, "hard")
		resetNoError(t, err)
		resetNoError(t, cmd.Run())
		assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, "unborn", gitTest(t, root, "branch", "--show-current"))
	})
	for _, mode := range []string{"mixed", "hard"} {
		for _, mergeKind := range []string{"clean", "unmerged"} {
			t.Run("merge-"+mergeKind+"-"+mode, func(t *testing.T) {
				root, first, _ := transactionHistory(t)
				gitTest(t, root, "branch", "reset-source", first)
				gitTest(t, root, "checkout", "-q", "-b", "side", first)
				if mergeKind == "clean" {
					branchPlacementTestCommit(t, root, "side-file", "side\n", "side")
				} else {
					branchPlacementTestCommit(t, root, "tracked", "side conflicting bytes\n", "side")
				}
				gitTest(t, root, "checkout", "-q", "master")
				if mergeKind == "clean" {
					gitTest(t, root, "merge", "--no-commit", "--no-ff", "side")
				} else {
					branchPlacementTestCommit(t, root, "tracked", "master conflicting bytes\n", "master")
					if output, err := exec.Command("git", "-C", root, "merge", "side").CombinedOutput(); err == nil {
						t.Fatalf("expected unmerged merge conflict, output: %s", output)
					}
					assert.FileExists(t, filepath.Join(root, ".git", "MERGE_HEAD"))
				}
				headBefore := gitTest(t, root, "rev-parse", "HEAD")
				refsBefore := []string{gitTest(t, root, "rev-parse", "refs/heads/side"), gitTest(t, root, "rev-parse", "refs/heads/reset-source")}
				commands, target := branchActionResetTarget(t, root, "refs/heads/reset-source", first)
				cmd, err := commands.BranchActionResetCommand(target, mode)
				resetNoError(t, err)
				resetNoError(t, cmd.Run())
				_, statErr := os.Stat(filepath.Join(root, ".git", "MERGE_HEAD"))
				assert.True(t, errors.Is(statErr, os.ErrNotExist), "reset should abort native merge state")
				assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"))
				assert.NotEmpty(t, headBefore)
				assert.Equal(t, refsBefore[0], gitTest(t, root, "rev-parse", "refs/heads/side"))
				assert.Equal(t, refsBefore[1], gitTest(t, root, "rev-parse", "refs/heads/reset-source"))
				assert.Empty(t, gitTest(t, root, "stash", "list"), "reset must not create a stash")
			})
		}
	}
}

func TestBranchActionResetCommandPreservesIndexLockFailure(t *testing.T) {
	root, first, second := transactionHistory(t)
	gitTest(t, root, "branch", "reset-source", first)
	commands, target := branchActionResetTarget(t, root, "refs/heads/reset-source", first)
	lock := filepath.Join(root, ".git", "index.lock")
	resetNoError(t, os.WriteFile(lock, []byte("native lock owner\n"), 0o600))
	defer os.Remove(lock)
	cmd, err := commands.BranchActionResetCommand(target, "hard")
	resetNoError(t, err)
	err = cmd.Run()
	if err == nil {
		t.Fatal("expected reset to fail with index.lock")
	}
	assert.Contains(t, err.Error(), "index.lock")
	assert.Equal(t, []byte("native lock owner\n"), mustRead(t, lock))
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, "two\n", string(mustRead(t, filepath.Join(root, "tracked"))))
	assert.Empty(t, gitTest(t, root, "stash", "list"))
}

func TestBranchActionResetCommandRespectsOtherWorktreeReservation(t *testing.T) {
	t.Run("selected source occupied elsewhere is read-only", func(t *testing.T) {
		root, source, current := transactionHistory(t)
		linked := filepath.Join(t.TempDir(), "linked")
		gitTest(t, root, "branch", "reset-source", source)
		gitTest(t, root, "worktree", "add", "-q", linked, "reset-source")
		linkedGitDir := gitTest(t, linked, "rev-parse", "--absolute-git-dir")
		linkedIndex := mustRead(t, filepath.Join(linkedGitDir, "index"))
		linkedFile := mustRead(t, filepath.Join(linked, "tracked"))
		commands, target := branchActionResetTarget(t, root, "refs/heads/reset-source", source)
		cmd, err := commands.BranchActionResetCommand(target, "mixed")
		resetNoError(t, err)
		resetNoError(t, cmd.Run())
		assert.Equal(t, source, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, source, gitTest(t, linked, "rev-parse", "HEAD"))
		assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/heads/reset-source"))
		assert.Equal(t, []byte("two\n"), mustRead(t, filepath.Join(root, "tracked")), "mixed reset preserves current worktree bytes")
		assert.NotEqual(t, current, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, linkedIndex, mustRead(t, filepath.Join(linkedGitDir, "index")))
		assert.Equal(t, linkedFile, mustRead(t, filepath.Join(linked, "tracked")))
	})

	t.Run("tip movement is refused before command construction", func(t *testing.T) {
		root, source, current := transactionHistory(t)
		linked := filepath.Join(t.TempDir(), "linked")
		gitTest(t, root, "branch", "reset-source", source)
		gitTest(t, root, "worktree", "add", "--force", "-q", linked, "master")
		commands, target := branchActionResetTarget(t, root, "refs/heads/reset-source", source)
		mainIndex := mustRead(t, filepath.Join(root, ".git", "index"))
		linkedGitDir := gitTest(t, linked, "rev-parse", "--absolute-git-dir")
		linkedIndex := mustRead(t, filepath.Join(linkedGitDir, "index"))
		mainFile := mustRead(t, filepath.Join(root, "tracked"))
		linkedFile := mustRead(t, filepath.Join(linked, "tracked"))
		_, err := commands.BranchActionResetCommand(target, "mixed")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "another worktree")
		assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, current, gitTest(t, linked, "rev-parse", "HEAD"))
		assert.Equal(t, current, gitTest(t, root, "rev-parse", "refs/heads/master"))
		assert.Equal(t, mainIndex, mustRead(t, filepath.Join(root, ".git", "index")))
		assert.Equal(t, linkedIndex, mustRead(t, filepath.Join(linkedGitDir, "index")))
		assert.Equal(t, mainFile, mustRead(t, filepath.Join(root, "tracked")))
		assert.Equal(t, linkedFile, mustRead(t, filepath.Join(linked, "tracked")))
	})

	t.Run("same-tip hard reset may affect only current worktree files", func(t *testing.T) {
		root, _, current := transactionHistory(t)
		linked := filepath.Join(t.TempDir(), "linked")
		gitTest(t, root, "worktree", "add", "--force", "-q", linked, "master")
		commands, target := branchActionResetTarget(t, root, "refs/heads/master", current)
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("current dirty bytes\n"), 0o644)
		linkedFile := mustRead(t, filepath.Join(linked, "tracked"))
		mainTree := gitTest(t, root, "write-tree")
		linkedGitDir := gitTest(t, linked, "rev-parse", "--absolute-git-dir")
		linkedIndex := mustRead(t, filepath.Join(linkedGitDir, "index"))
		cmd, err := commands.BranchActionResetCommand(target, "hard")
		resetNoError(t, err)
		resetNoError(t, cmd.Run())
		assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, current, gitTest(t, linked, "rev-parse", "HEAD"))
		assert.Equal(t, current, gitTest(t, root, "rev-parse", "refs/heads/master"))
		assert.Equal(t, []byte("two\n"), mustRead(t, filepath.Join(root, "tracked")))
		assert.Equal(t, linkedFile, mustRead(t, filepath.Join(linked, "tracked")))
		assert.Equal(t, mainTree, gitTest(t, root, "write-tree"))
		assert.Equal(t, linkedIndex, mustRead(t, filepath.Join(linkedGitDir, "index")))
	})
}

func resetNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
