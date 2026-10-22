package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func selectedRebaseRequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func selectedCommitRebaseCommands(t *testing.T, root string) *RebaseCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	common := NewGitCommon(common.NewDummyCommon(), nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil)
	return NewRebaseCommands(common, nil, nil)
}

func selectedCommitRebaseCommit(t *testing.T, root, path, contents, message string) string {
	t.Helper()
	writeTestFile(t, filepath.Join(root, path), []byte(contents), 0o644)
	gitTest(t, root, "add", "--", path)
	gitTest(t, root, "commit", "-qm", message)
	return gitTest(t, root, "rev-parse", "HEAD")
}

func TestSelectedCommitRebasePrepareAndApply(t *testing.T) {
	root := initHistoricalRepo(t)
	base := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	head := selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target := selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	gitTest(t, root, "checkout", "-q", "topic")
	commands := selectedCommitRebaseCommands(t, root)

	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, target, plan.TargetOID())
	assert.Equal(t, "refs/heads/topic", plan.HeadRef())
	assert.Equal(t, head, plan.HeadOID())
	assert.False(t, plan.PreserveMerges())
	cmd, err := commands.SelectedCommitRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	args := cmd.GetCmd().Args
	assert.Contains(t, args, "-c")
	assert.Contains(t, args, "rebase.updateRefs=false")
	assert.Contains(t, args, "--no-autostash")
	assert.Contains(t, args, "--no-fork-point")
	assert.Contains(t, args, "--no-autosquash")
	assert.Contains(t, args, "--no-rebase-merges")
	selectedRebaseRequireNoError(t, cmd.Run())
	assert.Equal(t, target, gitTest(t, root, "merge-base", "HEAD", target))
	assert.Equal(t, base, gitTest(t, root, "merge-base", head, base))
}

func TestSelectedCommitRebaseChangedSourceAndUnprepared(t *testing.T) {
	root := initHistoricalRepo(t)
	target := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	branch := gitTest(t, root, "branch", "--show-current")
	first := selectedCommitRebaseCommit(t, root, "topic", "first\n", "first")
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	_, err = commands.SelectedCommitRebaseCommand(SelectedCommitRebasePlan{})
	if err == nil {
		t.Fatal("expected unprepared plan to fail")
	}
	gitTest(t, root, "branch", "same-tip")
	gitTest(t, root, "checkout", "-q", "same-tip")
	_, err = commands.SelectedCommitRebaseCommand(plan)
	assert.ErrorIs(t, err, ErrSelectedCommitRebaseChanged)
	gitTest(t, root, "checkout", "-q", branch)
	selectedCommitRebaseCommit(t, root, "next", "second\n", "second")
	_, err = commands.SelectedCommitRebaseCommand(plan)
	assert.ErrorIs(t, err, ErrSelectedCommitRebaseChanged)
	assert.NotEqual(t, first, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestSelectedCommitRebaseRejectsDirtyDetachedAndOperationStates(t *testing.T) {
	root := initHistoricalRepo(t)
	target := selectedCommitRebaseCommit(t, root, "tracked", "one\n", "one")
	commands := selectedCommitRebaseCommands(t, root)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
	_, err := commands.PrepareSelectedCommitRebase(target, false)
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	gitTest(t, root, "checkout", "--", "tracked")
	branch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "--detach", "-q")
	_, err = commands.PrepareSelectedCommitRebase(target, false)
	assert.Error(t, err)
	gitTest(t, root, "checkout", "-q", branch)
	if err := os.Mkdir(filepath.Join(root, ".git", "rebase-merge"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = commands.PrepareSelectedCommitRebase(target, false)
	assert.ErrorContains(t, err, "cannot start a rebase while a rebase is in progress")
}

func TestSelectedCommitRebaseContinuationWithStagedResolution(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "shared", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "side")
	interior := selectedCommitRebaseCommit(t, root, "replayed", "first side commit\n", "side first")
	selectedCommitRebaseCommit(t, root, "shared", "side\n", "side second")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target := selectedCommitRebaseCommit(t, root, "shared", "main\n", "main")
	gitTest(t, root, "config", "rebase.updateRefs", "true")
	// An interior auxiliary ref at a replayed commit must survive conflict continuation.
	gitTest(t, root, "branch", "interior", interior)
	gitTest(t, root, "checkout", "-q", "side")
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	cmd, err := commands.SelectedCommitRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	err = cmd.Run()
	if err == nil {
		t.Fatal("expected divergent edits to conflict")
	}
	writeTestFile(t, filepath.Join(root, "shared"), []byte("resolved\n"), 0o644)
	gitTest(t, root, "add", "shared")
	// The production continue wrapper invokes lazygit as its editor. In this
	// package the executable is the test binary, not the application's daemon;
	// exercise the same continuation command with a deterministic editor instead.
	selectedRebaseRequireNoError(t, commands.GenericMergeOrRebaseActionCmdObj("rebase", "continue").
		AddEnvVars("GIT_EDITOR=true").Run())
	assert.Equal(t, interior, gitTest(t, root, "rev-parse", "refs/heads/interior"))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", "rebase.updateRefs"))
	selectedCommitRebaseCommit(t, root, "shared", "side again\n", "side again")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target2 := selectedCommitRebaseCommit(t, root, "shared", "main again\n", "main again")
	gitTest(t, root, "checkout", "-q", "side")
	beforeAbort := gitTest(t, root, "rev-parse", "HEAD")
	plan, err = commands.PrepareSelectedCommitRebase(target2, false)
	selectedRebaseRequireNoError(t, err)
	cmd, err = commands.SelectedCommitRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	if err := cmd.Run(); err == nil {
		t.Fatal("expected abort fixture to conflict")
	}
	selectedRebaseRequireNoError(t, commands.AbortRebase())
	assert.Equal(t, beforeAbort, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestSelectedCommitRebaseAncestorNoopAndDescendant(t *testing.T) {
	for _, targetPosition := range []string{"root", "ancestor", "same", "descendant"} {
		t.Run(targetPosition, func(t *testing.T) {
			root := initHistoricalRepo(t)
			rootOID := selectedCommitRebaseCommit(t, root, "root", "root\n", "root")
			first := selectedCommitRebaseCommit(t, root, "one", "one\n", "one")
			second := selectedCommitRebaseCommit(t, root, "two", "two\n", "two")
			target := first
			if targetPosition == "root" {
				target = rootOID
			}
			if targetPosition == "same" {
				target = second
			}
			if targetPosition == "descendant" {
				gitTest(t, root, "branch", "source", second)
				target = selectedCommitRebaseCommit(t, root, "three", "three\n", "three")
				gitTest(t, root, "checkout", "-q", "source")
			}
			headBefore := gitTest(t, root, "rev-parse", "HEAD")
			commands := selectedCommitRebaseCommands(t, root)
			plan, err := commands.PrepareSelectedCommitRebase(target, false)
			selectedRebaseRequireNoError(t, err)
			cmd, err := commands.SelectedCommitRebaseCommand(plan)
			selectedRebaseRequireNoError(t, err)
			selectedRebaseRequireNoError(t, cmd.Run())
			if targetPosition == "descendant" {
				assert.Equal(t, target, gitTest(t, root, "rev-parse", "HEAD"))
			} else {
				assert.Equal(t, headBefore, gitTest(t, root, "rev-parse", "HEAD"), "ancestor and current-tip targets are no-ops")
			}
		})
	}
}

func TestSelectedCommitRebaseUntrackedFiles(t *testing.T) {
	t.Run("unrelated file is preserved", func(t *testing.T) {
		root := initHistoricalRepo(t)
		selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
		baseBranch := gitTest(t, root, "branch", "--show-current")
		gitTest(t, root, "checkout", "-qb", "topic")
		target := selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
		gitTest(t, root, "checkout", "-q", baseBranch)
		selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
		gitTest(t, root, "checkout", "-q", "topic")
		untracked := filepath.Join(root, "untracked")
		writeTestFile(t, untracked, []byte("keep these bytes\n"), 0o600)
		commands := selectedCommitRebaseCommands(t, root)
		plan, err := commands.PrepareSelectedCommitRebase(target, false)
		selectedRebaseRequireNoError(t, err)
		cmd, err := commands.SelectedCommitRebaseCommand(plan)
		selectedRebaseRequireNoError(t, err)
		selectedRebaseRequireNoError(t, cmd.Run())
		bytes, err := os.ReadFile(untracked)
		selectedRebaseRequireNoError(t, err)
		assert.Equal(t, "keep these bytes\n", string(bytes))
	})

	t.Run("target collision remains intact and branch does not move", func(t *testing.T) {
		root := initHistoricalRepo(t)
		selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
		baseBranch := gitTest(t, root, "branch", "--show-current")
		gitTest(t, root, "checkout", "-qb", "topic")
		selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
		gitTest(t, root, "checkout", "-q", baseBranch)
		target := selectedCommitRebaseCommit(t, root, "collision", "target content\n", "target collision")
		gitTest(t, root, "checkout", "-q", "topic")
		collision := filepath.Join(root, "collision")
		writeTestFile(t, collision, []byte("user content\n"), 0o600)
		before := gitTest(t, root, "rev-parse", "HEAD")
		indexBefore := gitTest(t, root, "write-tree")
		commands := selectedCommitRebaseCommands(t, root)
		plan, err := commands.PrepareSelectedCommitRebase(target, false)
		selectedRebaseRequireNoError(t, err)
		cmd, err := commands.SelectedCommitRebaseCommand(plan)
		selectedRebaseRequireNoError(t, err)
		assert.Error(t, cmd.Run(), "Git should reject overwriting an untracked target collision")
		bytes, err := os.ReadFile(collision)
		selectedRebaseRequireNoError(t, err)
		assert.Equal(t, "user content\n", string(bytes))
		assert.Equal(t, before, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, indexBefore, gitTest(t, root, "write-tree"))
	})
}

func TestSelectedCommitRebaseDetachedGitErrorsAreNotChanged(t *testing.T) {
	root := initHistoricalRepo(t)
	target := selectedCommitRebaseCommit(t, root, "tracked", "one\n", "one")
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	gitTest(t, root, "checkout", "--detach", "-q")
	_, err = commands.SelectedCommitRebaseCommand(plan)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrSelectedCommitRebaseChanged)
}

func TestSelectedCommitRebaseLiteralTargetsAndUnbornSource(t *testing.T) {
	root := initHistoricalRepo(t)
	base := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	branch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "target")
	target := selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	gitTest(t, root, "checkout", "-q", branch)
	head := selectedCommitRebaseCommit(t, root, "source", "source\n", "source")
	gitTest(t, root, "branch", target, head)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	cmd, err := commands.SelectedCommitRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	selectedRebaseRequireNoError(t, cmd.Run())
	assert.Equal(t, target, gitTest(t, root, "merge-base", "HEAD", "refs/heads/target"))
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "refs/heads/"+target))
	for _, invalid := range []string{"HEAD", "--root", base[:8], ""} {
		_, err := commands.PrepareSelectedCommitRebase(invalid, false)
		assert.Error(t, err)
	}
	gitTest(t, root, "checkout", "--orphan", "unborn")
	gitTest(t, root, "rm", "-rf", ".")
	_, err = commands.PrepareSelectedCommitRebase(base, false)
	assert.ErrorContains(t, err, "unborn")
}

func TestSelectedCommitRebaseTopologyAndOtherRefs(t *testing.T) {
	for _, preserve := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "preserve-merges"}[preserve], func(t *testing.T) {
			root := initHistoricalRepo(t)
			base := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
			branch := gitTest(t, root, "branch", "--show-current")
			interior := selectedCommitRebaseCommit(t, root, "source", "source\n", "source")
			gitTest(t, root, "branch", "interior", interior)
			gitTest(t, root, "checkout", "-qb", "side", base)
			selectedCommitRebaseCommit(t, root, "side", "side\n", "side")
			gitTest(t, root, "checkout", "-q", branch)
			gitTest(t, root, "merge", "--no-ff", "-qm", "merge side", "side")
			gitTest(t, root, "checkout", "-qb", "target", base)
			target := selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
			gitTest(t, root, "checkout", "-q", branch)
			gitTest(t, root, "config", "rebase.updateRefs", "true")
			// Explicit choices must override the opposite repository default.
			gitTest(t, root, "config", "rebase.rebaseMerges", map[bool]string{false: "true", true: "false"}[preserve])
			commands := selectedCommitRebaseCommands(t, root)
			plan, err := commands.PrepareSelectedCommitRebase(target, preserve)
			selectedRebaseRequireNoError(t, err)
			cmd, err := commands.SelectedCommitRebaseCommand(plan)
			selectedRebaseRequireNoError(t, err)
			assert.True(t, cmd.LockRetriesDisabled())
			selectedRebaseRequireNoError(t, cmd.Run())
			assert.Equal(t, interior, gitTest(t, root, "rev-parse", "refs/heads/interior"))
			assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/target"))
			assert.Equal(t, target, gitTest(t, root, "merge-base", "HEAD", target))
			if preserve {
				assert.Equal(t, "1", gitTest(t, root, "rev-list", "--count", "--merges", "HEAD"))
			} else {
				assert.Equal(t, "0", gitTest(t, root, "rev-list", "--count", "--merges", "HEAD"))
			}
			assert.Equal(t, "true", gitTest(t, root, "config", "rebase.updateRefs"))
		})
	}
}

func TestSelectedCommitRebaseSHA256WhenSupported(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "-C", root, "init", "-q", "--object-format=sha256").CombinedOutput(); err != nil {
		t.Skipf("Git does not support SHA-256 repositories: %s", output)
	}
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	base := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	selectedCommitRebaseCommit(t, root, "next", "next\n", "next")
	head := gitTest(t, root, "rev-parse", "HEAD")
	assert.Len(t, base, 64)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(base, false)
	selectedRebaseRequireNoError(t, err)
	cmd, err := commands.SelectedCommitRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	selectedRebaseRequireNoError(t, cmd.Run())
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
}
