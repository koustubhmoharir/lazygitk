package git_commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchRenamePreservesSelectedBranchStateAndMovesConfig(t *testing.T) {
	root, old, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	gitTest(t, root, "config", "branch.selected.remote", "origin")
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/main")
	gitTest(t, root, "config", "branch.selected.description", "topic")
	gitTest(t, root, "config", "branch.other.description", "keep")
	gitTest(t, root, "config", "branch.DEST.description", "case-sensitive separate subsection")
	gitTest(t, root, "checkout", "-q", "master")
	if err := os.WriteFile(filepath.Join(root, "tracked"), []byte("staged bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(root, "tracked"), []byte("unstaged\x00bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "untracked.bin"), []byte{0, 1, 255}, 0o600); err != nil {
		t.Fatal(err)
	}
	headPath := filepath.Join(root, ".git", "HEAD")
	indexPath := filepath.Join(root, ".git", "index")
	reflogPath := filepath.Join(root, ".git", "logs", "HEAD")
	trackedPath := filepath.Join(root, "tracked")
	beforeHead := mustRead(t, headPath)
	beforeIndex := mustRead(t, indexPath)
	beforeReflog := mustRead(t, reflogPath)
	beforeTracked := mustRead(t, trackedPath)
	beforeUntracked := mustRead(t, filepath.Join(root, "untracked.bin"))
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchRename("selected", old, "topic/renamed")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, "selected", plan.OldName())
	assert.Equal(t, "topic/renamed", plan.NewName())
	assert.Equal(t, old, plan.SourceOID())
	assert.NoError(t, commands.ApplyBranchRename(plan))
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/topic/renamed"))
	assertBranchRenameGitFails(t, root, "show-ref", "--verify", "--quiet", "refs/heads/selected")
	assert.Equal(t, beforeHead, mustRead(t, headPath))
	assert.Equal(t, beforeIndex, mustRead(t, indexPath))
	assert.Equal(t, beforeReflog, mustRead(t, reflogPath))
	assert.Equal(t, beforeTracked, mustRead(t, trackedPath))
	assert.Equal(t, beforeUntracked, mustRead(t, filepath.Join(root, "untracked.bin")))
	assert.Equal(t, "origin", gitTest(t, root, "config", "branch.topic/renamed.remote"))
	assert.Equal(t, "refs/heads/main", gitTest(t, root, "config", "branch.topic/renamed.merge"))
	assert.Equal(t, "topic", gitTest(t, root, "config", "branch.topic/renamed.description"))
	assert.Equal(t, "keep", gitTest(t, root, "config", "branch.other.description"))
	assert.Equal(t, "case-sensitive separate subsection", gitTest(t, root, "config", "branch.DEST.description"))
}

func TestBranchRenameCurrentBranchChangesOnlySymbolicHeadName(t *testing.T) {
	root, _, _ := transactionHistory(t)
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	gitTest(t, root, "checkout", "-q", "-b", "current-rename")
	gitTest(t, root, "config", "branch.current-rename.description", "current")
	headoid := gitTest(t, root, "rev-parse", "HEAD")
	index := gitTest(t, root, "write-tree")
	status := gitTest(t, root, "status", "--porcelain=v1", "-z")
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchRename("current-rename", headoid, "renamed-current")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.NoError(t, commands.ApplyBranchRename(plan))
	assert.Equal(t, "refs/heads/renamed-current", gitTest(t, root, "symbolic-ref", "HEAD"))
	assert.Equal(t, headoid, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, index, gitTest(t, root, "write-tree"))
	assert.Equal(t, status, gitTest(t, root, "status", "--porcelain=v1", "-z"))
	assert.Equal(t, headoid, gitTest(t, root, "rev-parse", "refs/heads/renamed-current"))
}

func TestBranchRenameRejectsDestinationsAndRevalidates(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	commands := branchSyncTestCommands(t, root)
	for _, name := range []string{"", "bad name", "bad..name", "-option", "@{-1}"} {
		_, err := commands.PrepareBranchRename("selected", old, name)
		assert.Error(t, err, name)
	}
	gitTest(t, root, "branch", "exists", old)
	_, err := commands.PrepareBranchRename("selected", old, "exists")
	assert.Error(t, err)
	gitTest(t, root, "branch", "-D", "exists")
	gitTest(t, root, "config", "branch.dest.description", "owned")
	_, err = commands.PrepareBranchRename("selected", old, "dest")
	assert.Error(t, err)
	gitTest(t, root, "config", "--unset", "branch.dest.description")
	gitTest(t, root, "config", "branch.DEST.description", "separate case-sensitive subsection")
	plan, err := commands.PrepareBranchRename("selected", old, "dest")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	gitTest(t, root, "update-ref", "refs/heads/selected", target, old)
	err = commands.ApplyBranchRename(plan)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/selected"))
	assertBranchRenameGitFails(t, root, "show-ref", "--verify", "--quiet", "refs/heads/dest")
	plan, err = commands.PrepareBranchRename("selected", target, "dest")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	gitTest(t, root, "config", "branch.dest.description", "appeared")
	err = commands.ApplyBranchRename(plan)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	assert.Equal(t, "appeared", gitTest(t, root, "config", "branch.dest.description"))
	foreign, err := branchSyncTestCommands(t, root).PrepareBranchRename("selected", target, "another")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = commands.ApplyBranchRename(foreign)
	assert.Error(t, err)
}

func TestBranchRenameRefAppearsAndSymbolicRefsAreRefused(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchRename("selected", old, "target")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	gitTest(t, root, "update-ref", "refs/heads/target", target)
	assert.ErrorIs(t, commands.ApplyBranchRename(plan), ErrBranchSyncChanged)
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/target"))
	gitTest(t, root, "symbolic-ref", "refs/heads/source-alias", "refs/heads/selected")
	_, err = commands.PrepareBranchRename("source-alias", old, "new-name")
	assert.Error(t, err)
	gitTest(t, root, "symbolic-ref", "refs/heads/symbolic-destination", "refs/heads/selected")
	_, err = commands.PrepareBranchRename("selected", old, "symbolic-destination")
	assert.Error(t, err)
	gitTest(t, root, "symbolic-ref", "refs/heads/dangling-destination", "refs/heads/missing")
	_, err = commands.PrepareBranchRename("selected", old, "dangling-destination")
	assert.Error(t, err)
}

func TestBranchRenameRefusesOperationReservation(t *testing.T) {
	root, old, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	gitDir := gitTest(t, root, "rev-parse", "--absolute-git-dir")
	rebaseDir := filepath.Join(gitDir, "rebase-merge")
	if err := os.Mkdir(rebaseDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rebaseDir, "head-name"), []byte("refs/heads/selected\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := branchSyncTestCommands(t, root).PrepareBranchRename("selected", old, "renamed")
	assert.Error(t, err)
	commands := branchSyncTestCommands(t, root)
	noop, err := commands.PrepareBranchRename("selected", old, "selected")
	assert.NoError(t, err)
	if err == nil {
		assert.False(t, noop.NeedsUpdate())
		assert.NoError(t, commands.ApplyBranchRename(noop))
	}
}

func TestBranchRenameAllowsLinkedWorktreeReservation(t *testing.T) {
	root, old, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", linked, "selected")
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchRename("selected", old, "renamed")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	linkedGitDir := gitTest(t, linked, "rev-parse", "--absolute-git-dir")
	headPath := filepath.Join(linkedGitDir, "HEAD")
	indexPath := filepath.Join(linkedGitDir, "index")
	beforeHead := mustRead(t, headPath)
	beforeIndex := mustRead(t, indexPath)
	beforeTracked := mustRead(t, filepath.Join(linked, "tracked"))
	assert.NoError(t, commands.ApplyBranchRename(plan))
	assert.Equal(t, []byte("ref: refs/heads/selected\n"), beforeHead)
	assert.Equal(t, []byte("ref: refs/heads/renamed\n"), mustRead(t, headPath))
	assert.Equal(t, beforeIndex, mustRead(t, indexPath))
	assert.Equal(t, beforeTracked, mustRead(t, filepath.Join(linked, "tracked")))
	assert.Equal(t, old, gitTest(t, linked, "rev-parse", "HEAD"))
}

func TestBranchRenameSameNameIsZeroWriteNoop(t *testing.T) {
	root, old, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	before := gitTest(t, root, "config", "--local", "--null", "--list")
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchRename("selected", old, "selected")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.False(t, plan.NeedsUpdate())
	assert.NoError(t, commands.ApplyBranchRename(plan))
	assert.Equal(t, before, gitTest(t, root, "config", "--local", "--null", "--list"))
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/selected"))
}

func TestBranchRenameAllowsUnicodeAndDottedLiteralNames(t *testing.T) {
	root, old, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchRename("selected", old, "team.v2/é")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.NoError(t, commands.ApplyBranchRename(plan))
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/team.v2/é"))
}

func TestBranchRenameApplyPropagatesNativePartialFailure(t *testing.T) {
	root, old, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	gitTest(t, root, "config", "branch.selected.description", "owned source config")
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchRename("selected", old, "renamed")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	commonDir := gitTest(t, root, "rev-parse", "--git-common-dir")
	lock := filepath.Join(root, commonDir, "config.lock")
	lockBytes := []byte("lock content preserved\n")
	if err := os.WriteFile(lock, lockBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, commonDir, "config")
	configBefore := mustRead(t, configPath)
	err = commands.ApplyBranchRename(plan)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrBranchSyncChanged))
	assert.Equal(t, lockBytes, mustRead(t, lock))
	assert.Equal(t, configBefore, mustRead(t, configPath))
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/renamed"))
	assertBranchRenameGitFails(t, root, "show-ref", "--verify", "--quiet", "refs/heads/selected")
	assert.Equal(t, "refs/heads/master", gitTest(t, root, "symbolic-ref", "HEAD"))
}

func assertBranchRenameGitFails(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	assert.Error(t, cmd.Run())
}
