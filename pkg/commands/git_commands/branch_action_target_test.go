package git_commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchActionTargetResolvesLiteralRefsAndCommitOIDs(t *testing.T) {
	root, first, second := transactionHistory(t)
	gitTest(t, root, "update-ref", "refs/vendor/custom", first)
	gitTest(t, root, "update-ref", "refs/vendor/parent/child", first)
	commands := transactionCommands(t, root)
	got, err := commands.ResolveBranchActionCommit("refs/vendor/custom")
	assert.NoError(t, err)
	assert.Equal(t, first, got)
	got, err = commands.ResolveBranchActionCommit("refs/heads/master")
	assert.NoError(t, err)
	assert.Equal(t, second, got)
	got, err = commands.ResolveBranchActionCommit(strings.ToUpper(first))
	assert.NoError(t, err)
	assert.Equal(t, first, got)
	for _, bad := range []string{"master", "custom", "HEAD", "HEAD~1", "refs/heads/master^{}", "refs/heads/master/child", "refs/tags/master", "refs/heads/missing", "refs/vendor/parent"} {
		t.Run(bad, func(t *testing.T) { _, err := commands.ResolveBranchActionCommit(bad); assert.Error(t, err) })
	}
	gitTest(t, root, "symbolic-ref", "refs/vendor/alias", "refs/heads/master")
	gitTest(t, root, "symbolic-ref", "refs/vendor/dangling", "refs/heads/missing")
	for _, bad := range []string{"refs/vendor/alias", "refs/vendor/dangling"} {
		_, err := commands.ResolveBranchActionCommit(bad)
		assert.Error(t, err)
		assert.False(t, errors.Is(err, ErrBranchActionChanged))
	}
	blob := gitTest(t, root, "hash-object", "-w", "--stdin")
	_, err = commands.ResolveBranchActionCommit(blob)
	assert.Error(t, err, "a full blob OID is not a commit")
	_, err = commands.ResolveBranchActionCommit(second[:10])
	assert.Error(t, err, "abbreviated OIDs are not accepted")
}

func TestBranchActionTargetPeelsTagRefsToCommits(t *testing.T) {
	root, first, _ := transactionHistory(t)
	gitTest(t, root, "tag", "lightweight-commit", first)
	gitTest(t, root, "tag", "-a", "annotated-commit", "-m", "annotated", first)
	gitTest(t, root, "tag", "-a", "nested-inner", "-m", "inner", first)
	gitTest(t, root, "tag", "-a", "nested-outer", "-m", "outer", "nested-inner")
	blob := gitTest(t, root, "hash-object", "-w", "--stdin")
	gitTest(t, root, "tag", "-a", "blob-target", "-m", "blob", blob)

	commands := transactionCommands(t, root)
	for _, name := range []string{"lightweight-commit", "annotated-commit", "nested-outer"} {
		t.Run(name, func(t *testing.T) {
			got, err := commands.ResolveBranchActionCommit("refs/tags/" + name)
			assert.NoError(t, err)
			assert.Equal(t, first, got)
		})
	}
	_, err := commands.ResolveBranchActionCommit("refs/tags/blob-target")
	assert.Error(t, err, "a tag pointing to a blob is not a commit")
}

func TestBranchActionTargetRejectsMovedTagWithoutChangingRepositoryState(t *testing.T) {
	root, first, second := transactionHistory(t)
	gitTest(t, root, "tag", "-a", "selected", "-m", "selected", first)
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/tags/selected", first)
	assert.NoError(t, err)
	assert.Equal(t, first, target.SourceOID())

	gitTest(t, root, "tag", "-f", "-a", "selected", "-m", "moved", second)
	indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
	headBefore := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	statusBefore := gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all")
	_, err = commands.ResolveBranchActionCommit("refs/tags/selected")
	assert.NoError(t, err)
	assert.ErrorIs(t, commands.CheckBranchActionTarget(target), ErrBranchActionChanged)
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, statusBefore, gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all"))
}

func TestBranchActionTargetCapturesHeadAndChecksStalenessReadOnly(t *testing.T) {
	root, first, second := transactionHistory(t)
	gitTest(t, root, "branch", "selected", first)
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/selected", first)
	assert.NoError(t, err)
	_, err = commands.PrepareBranchActionTarget("refs/heads/selected", second)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assert.Equal(t, "refs/heads/selected", target.SourceRef())
	assert.Equal(t, first, target.SourceOID())
	assert.Equal(t, "refs/heads/master", target.HeadRef())
	assert.Equal(t, second, target.HeadOID())
	assert.NoError(t, commands.CheckBranchActionTarget(target))
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
	gitTest(t, root, "update-ref", "refs/heads/selected", second, first)
	assert.ErrorIs(t, commands.CheckBranchActionTarget(target), ErrBranchActionChanged)

	gitTest(t, root, "update-ref", "refs/heads/selected", first, second)
	target, err = commands.PrepareBranchActionTarget("refs/heads/selected", first)
	assert.NoError(t, err)
	branchPlacementTestCommit(t, root, "third", "third\n", "third")
	assert.ErrorIs(t, commands.CheckBranchActionTarget(target), ErrBranchActionChanged)

	target, err = commands.PrepareBranchActionTarget("refs/heads/selected", first)
	assert.NoError(t, err)
	gitTest(t, root, "branch", "other", second)
	gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/other")
	assert.ErrorIs(t, commands.CheckBranchActionTarget(target), ErrBranchActionChanged)
	gitTest(t, root, "update-ref", "-d", "refs/heads/selected", first)
	err = commands.CheckBranchActionTarget(target)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
}

func TestBranchActionTargetRejectsForeignAndZeroTargets(t *testing.T) {
	root, _, second := transactionHistory(t)
	commands := transactionCommands(t, root)
	zero := BranchActionTarget{}
	assert.Error(t, commands.CheckBranchActionTarget(zero))
	prepared, err := commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.NoError(t, err)
	otherRoot, _, _ := transactionHistory(t)
	assert.Error(t, transactionCommands(t, otherRoot).CheckBranchActionTarget(prepared))
	assert.Error(t, transactionCommands(t, root).CheckBranchActionTarget(prepared), "another backend for the same filesystem is a different owner")
	_, err = commands.PrepareBranchActionTarget("refs/heads/master", "")
	assert.Error(t, err)
}

func TestBranchActionTargetSupportsDetachedUnbornAndInProgressMerge(t *testing.T) {
	root, first, second := transactionHistory(t)
	commands := transactionCommands(t, root)
	gitTest(t, root, "checkout", "--detach", "-q", second)
	target, err := commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.NoError(t, err)
	assert.Equal(t, "", target.HeadRef())
	assert.Equal(t, second, target.HeadOID())
	assert.NoError(t, commands.CheckBranchActionTarget(target))

	gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/unborn")
	target, err = commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.NoError(t, err)
	assert.Equal(t, "refs/heads/unborn", target.HeadRef())
	assert.Empty(t, target.HeadOID())
	assert.NoError(t, commands.CheckBranchActionTarget(target))

	gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/master")
	gitTest(t, root, "checkout", "-q", "-b", "side", first)
	branchPlacementTestCommit(t, root, "side-file", "side\n", "side")
	gitTest(t, root, "checkout", "-q", "master")
	gitTest(t, root, "merge", "--no-commit", "--no-ff", "side")
	// The active merge is allowed through read-only inspection; a caller may
	// explicitly choose reset/abort. Snapshot repository state across both calls.
	indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
	headBefore := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	mergeHeadBefore := mustRead(t, filepath.Join(root, ".git", "MERGE_HEAD"))
	statusBefore := gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all")
	configBefore := mustRead(t, filepath.Join(root, ".git", "config"))
	target, err = commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.NoError(t, err)
	assert.NoError(t, commands.CheckBranchActionTarget(target))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, mergeHeadBefore, mustRead(t, filepath.Join(root, ".git", "MERGE_HEAD")))
	assert.Equal(t, statusBefore, gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all"))
	assert.Equal(t, configBefore, mustRead(t, filepath.Join(root, ".git", "config")))
}

func TestBranchActionTargetInspectionsPreserveDirtyIndexAndWorktree(t *testing.T) {
	root, _, second := transactionHistory(t)
	commands := transactionCommands(t, root)
	tracked := filepath.Join(root, "tracked")
	writeTestFile(t, tracked, []byte("staged bytes\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
	writeTestFile(t, tracked, []byte("unstaged bytes\n"), 0o644)
	untracked := filepath.Join(root, "harmless-untracked")
	writeTestFile(t, untracked, []byte{0, 1, 2, 255}, 0o600)
	headBefore := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	target, err := commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.NoError(t, err)
	assert.NoError(t, commands.CheckBranchActionTarget(target))
	_, err = commands.ResolveBranchActionCommit("refs/heads/master")
	assert.NoError(t, err)
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, []byte("unstaged bytes\n"), mustRead(t, tracked))
	assert.Equal(t, []byte{0, 1, 2, 255}, mustRead(t, untracked))
}

func TestBranchActionTargetPreservesNativeConfigQueryErrors(t *testing.T) {
	root, _, second := transactionHistory(t)
	commands := transactionCommands(t, root)
	configPath := filepath.Join(root, ".git", "config")
	original := mustRead(t, configPath)
	defer func() { writeTestFile(t, configPath, original, 0o644) }()
	if err := os.WriteFile(configPath, append(append([]byte(nil), original...), []byte("\n[broken\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrBranchActionChanged)
	assert.Contains(t, err.Error(), "bad config")
}

func TestBranchActionTargetSHA256(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "-C", root, "init", "-q", "-b", "master", "--object-format=sha256").CombinedOutput(); err != nil {
		t.Skipf("Git does not support SHA-256 repositories: %v: %s", err, output)
	}
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("one\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	gitTest(t, root, "commit", "-qm", "one")
	first := gitTest(t, root, "rev-parse", "HEAD")
	commands := transactionCommands(t, root)
	assert.Len(t, first, 64)
	resolved, err := commands.ResolveBranchActionCommit(strings.ToUpper(first))
	assert.NoError(t, err)
	assert.Equal(t, first, resolved)
	target, err := commands.PrepareBranchActionTarget("refs/heads/master", first)
	assert.NoError(t, err)
	assert.NoError(t, commands.CheckBranchActionTarget(target))
	branchPlacementTestCommit(t, root, "next", "two\n", "two")
	assert.ErrorIs(t, commands.CheckBranchActionTarget(target), ErrBranchActionChanged)
}
