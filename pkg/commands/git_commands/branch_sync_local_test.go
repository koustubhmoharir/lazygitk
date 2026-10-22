package git_commands

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalBranchSyncAdvancesSelectedOffBranchOnly(t *testing.T) {
	root, _, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected")
	selected := gitTest(t, root, "rev-parse", "refs/heads/selected")
	target := branchPlacementTestCommit(t, root, "next", "next\n", "next")
	gitTest(t, root, "branch", "upstream", target)
	gitTest(t, root, "checkout", "-q", "master")
	current := gitTest(t, root, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "binary-untracked"), []byte{0, 1, 2, 255}, 0o600)
	gitTest(t, root, "add", "tracked")
	index := gitTest(t, root, "write-tree")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged\n"), 0o644)
	rawHead := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	rawIndex := mustRead(t, filepath.Join(root, ".git", "index"))
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareLocalBranchUpdate("selected", selected, "refs/heads/upstream")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, plan.NeedsUpdate())
	assert.Equal(t, target, plan.TargetOID())
	if err := commands.ApplyBranchFastForward(plan); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, rawHead, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, rawIndex, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, index, gitTest(t, root, "write-tree"))
	assert.Equal(t, "unstaged\n", string(mustRead(t, filepath.Join(root, "tracked"))))
	assert.Equal(t, []byte{0, 1, 2, 255}, mustRead(t, filepath.Join(root, "binary-untracked")))
}

func TestLocalBranchSyncNoRewindAndRejectsDivergenceAndCurrent(t *testing.T) {
	root, _, _ := transactionHistory(t)
	base := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected", base)
	commands := branchSyncTestCommands(t, root)
	noop, err := commands.PrepareLocalBranchUpdate("selected", base, "refs/heads/master")
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, noop.NeedsUpdate())
	forward := branchPlacementTestCommit(t, root, "forward", "forward\n", "forward")
	gitTest(t, root, "branch", "upstream", forward)
	fast, err := commands.PrepareLocalBranchUpdate("selected", base, "refs/heads/upstream")
	if err != nil {
		t.Fatal(err)
	}
	if err := commands.ApplyBranchFastForward(fast); err != nil {
		t.Fatal(err)
	}
	ahead, err := commands.PrepareLocalBranchUpdate("selected", forward, "refs/heads/master")
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ahead.NeedsUpdate(), "an upstream behind selected must never rewind it")
	gitTest(t, root, "branch", "diverged", base)
	gitTest(t, root, "checkout", "-q", "diverged")
	branchPlacementTestCommit(t, root, "diverged", "diverged\n", "diverged")
	gitTest(t, root, "checkout", "-q", "master")
	_, err = commands.PrepareLocalBranchUpdate("selected", forward, "refs/heads/diverged")
	assert.ErrorIs(t, err, ErrBranchSyncDiverged)
	_, err = commands.PrepareLocalBranchUpdate("master", base, "refs/heads/upstream")
	assert.Error(t, err, "current branch cannot use ref-only update")
}

func TestLocalBranchSyncRejectsBadExactUpstreamRefs(t *testing.T) {
	root, _, _ := transactionHistory(t)
	base := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected", base)
	gitTest(t, root, "symbolic-ref", "refs/heads/alias", "refs/heads/master")
	gitTest(t, root, "symbolic-ref", "refs/heads/dangling", "refs/heads/missing")
	commands := branchSyncTestCommands(t, root)
	for _, ref := range []string{"master", "refs/tags/master", "refs/heads/", "refs/heads/missing", "refs/heads/alias", "refs/heads/dangling", "refs/heads/blob", "refs/heads/../master"} {
		t.Run(ref, func(t *testing.T) {
			_, err := commands.PrepareLocalBranchUpdate("selected", base, ref)
			assert.Error(t, err)
		})
	}
}

func TestLocalBranchSyncCapturesUpstreamOIDAndChecksReservations(t *testing.T) {
	root, _, _ := transactionHistory(t)
	base := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected", base)
	first := branchPlacementTestCommit(t, root, "first", "first\n", "first")
	gitTest(t, root, "branch", "upstream", first)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareLocalBranchUpdate("selected", base, "refs/heads/upstream")
	if err != nil {
		t.Fatal(err)
	}
	second := branchPlacementTestCommit(t, root, "second", "second\n", "second")
	gitTest(t, root, "update-ref", "refs/heads/upstream", second)
	if err := commands.ApplyBranchFastForward(plan); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/selected"), "approval applies the captured OID")

	root2, _, _ := transactionHistory(t)
	base2 := gitTest(t, root2, "rev-parse", "HEAD")
	gitTest(t, root2, "branch", "selected", base2)
	worktree := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root2, "worktree", "add", "-q", "-b", "occupied", worktree)
	occupiedTip := gitTest(t, root2, "rev-parse", "refs/heads/occupied")
	forward := branchPlacementTestCommit(t, root2, "occupied-next", "next\n", "next")
	gitTest(t, root2, "branch", "upstream", forward)
	_, err = branchSyncTestCommands(t, root2).PrepareLocalBranchUpdate("occupied", occupiedTip, "refs/heads/upstream")
	assert.Error(t, err, "a branch in a linked worktree cannot be advanced")
}
