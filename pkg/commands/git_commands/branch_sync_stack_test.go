package git_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchSyncStackNativeWalkAndStableSharedTips(t *testing.T) {
	root, rootOID := branchSyncTestRepo(t)
	first := branchPlacementTestCommit(t, root, "tracked", "first\n", "first")
	selected := branchPlacementTestCommit(t, root, "tracked", "selected\n", "selected")
	gitTest(t, root, "branch", "selected")
	gitTest(t, root, "update-ref", "refs/heads/topic/z", first)
	gitTest(t, root, "update-ref", "refs/heads/topic/a", first)
	gitTest(t, root, "update-ref", "refs/heads/main", first)
	gitTest(t, root, "update-ref", "refs/heads/side", rootOID)
	got, err := branchSyncTestCommands(t, root).InspectBranchStack("selected", selected, []string{"refs/heads/main"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []BranchStackMember{{Name: "master", OID: selected}}, got)
}

func TestBranchSyncStackEmptyMainRefsAndStaleSource(t *testing.T) {
	root, rootOID := branchSyncTestRepo(t)
	selected := branchPlacementTestCommit(t, root, "tracked", "next\n", "next")
	gitTest(t, root, "branch", "selected")
	gitTest(t, root, "branch", "root-tip", rootOID)
	got, err := branchSyncTestCommands(t, root).InspectBranchStack("selected", selected, nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []BranchStackMember{{Name: "master", OID: selected}, {Name: "root-tip", OID: rootOID}}, got)
	_, err = branchSyncTestCommands(t, root).InspectBranchStack("selected", rootOID, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
}

func TestBranchSyncStackRequiresExactDirectMainRefs(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "selected")
	for _, ref := range []string{"master", "refs/heads/missing", "refs/heads/alias", "refs/heads/master^{}"} {
		if ref == "refs/heads/alias" {
			gitTest(t, root, "symbolic-ref", ref, "refs/heads/master")
		}
		_, err := branchSyncTestCommands(t, root).InspectBranchStack("selected", oid, []string{ref})
		assert.Error(t, err, "main ref %q", ref)
	}
}

func TestBranchSyncStackNativeDAGAndStableSharedTips(t *testing.T) {
	root, base := branchSyncTestRepo(t)
	mainTip := branchPlacementTestCommit(t, root, "tracked", "main\n", "main")
	gitTest(t, root, "branch", "mainline", mainTip)
	lower := branchPlacementTestCommit(t, root, "tracked", "lower\n", "lower")
	shared := branchPlacementTestCommit(t, root, "tracked", "shared\n", "shared")
	selected := branchPlacementTestCommit(t, root, "tracked", "selected\n", "selected")
	gitTest(t, root, "branch", "selected")
	gitTest(t, root, "update-ref", "refs/heads/master", mainTip, selected)
	gitTest(t, root, "update-ref", "refs/heads/topic/z", shared)
	gitTest(t, root, "update-ref", "refs/heads/topic/a", shared)
	gitTest(t, root, "update-ref", "refs/heads/lower", lower)
	gitTest(t, root, "checkout", "-q", mainTip)
	side := branchPlacementTestCommit(t, root, "tracked", "side\n", "side")
	gitTest(t, root, "branch", "side", side)
	gitTest(t, root, "update-ref", "refs/heads/merged", mainTip)
	got, err := branchSyncTestCommands(t, root).InspectBranchStack("selected", selected, []string{"refs/heads/mainline"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []BranchStackMember{{Name: "topic/a", OID: shared}, {Name: "topic/z", OID: shared}, {Name: "lower", OID: lower}}, got)
	assert.NotContains(t, got, BranchStackMember{Name: "side", OID: side})
	assert.NotContains(t, got, BranchStackMember{Name: "merged", OID: mainTip})
	assert.NotEqual(t, "selected", gitTest(t, root, "branch", "--show-current"))
	assert.NotEqual(t, base, side)
}

func TestBranchSyncStackPreservesDirtyStateAndAllowsOccupiedBranch(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "selected")
	gitTest(t, root, "branch", "selected-occupied")
	gitTest(t, root, "worktree", "add", "-q", filepath.Join(root, "selected-linked"), "selected-occupied")
	assert.NoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("dirty unstaged\n"), 0o644))
	gitTest(t, root, "add", "tracked")
	assert.NoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("dirty staged and unstaged\n"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(root, "untracked"), []byte("untracked bytes\n"), 0o644))
	headBefore := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
	trackedBefore, err := os.ReadFile(filepath.Join(root, "tracked"))
	assert.NoError(t, err)
	got, err := branchSyncTestCommands(t, root).InspectBranchStack("selected-occupied", oid, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	trackedAfter, err := os.ReadFile(filepath.Join(root, "tracked"))
	assert.NoError(t, err)
	assert.Equal(t, trackedBefore, trackedAfter)
	assert.Equal(t, []byte("untracked bytes\n"), mustRead(t, filepath.Join(root, "untracked")))
}

func TestBranchSyncStackSkipsSymbolicHeadsAndRejectsBadSource(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "selected")
	gitTest(t, root, "symbolic-ref", "refs/heads/alias", "refs/heads/selected")
	gitTest(t, root, "symbolic-ref", "refs/heads/dangling-tip", "refs/heads/also-missing")
	got, err := branchSyncTestCommands(t, root).InspectBranchStack("selected", oid, nil)
	assert.NoError(t, err)
	assert.Equal(t, []BranchStackMember{{Name: "master", OID: oid}}, got)
	for _, name := range []string{"alias", "dangling-tip"} {
		_, err := branchSyncTestCommands(t, root).InspectBranchStack(name, oid, nil)
		assert.Error(t, err, name)
	}
	_, err = branchSyncTestCommands(t, root).InspectBranchStack("selected", oid[:12], nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	_, err = branchSyncTestCommands(t, root).InspectBranchStack("selected", "not-an-object-id", nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
}

func TestBranchSyncStackMalformedSnapshotFrames(t *testing.T) {
	for _, input := range []string{"refs/heads/a\x00abc", "refs/heads/a\x00abc\x00\x00", "refs/remotes/a\x00abc\x00\x00\n", "refs/heads/a\x00abc\x00\x00!"} {
		_, err := parseBranchStackHeads(input)
		assert.Error(t, err, "%q", input)
	}
}
