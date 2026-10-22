package git_commands

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchSyncTrackingWriteRejectsProtectedNamespaces(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	gitTest(t, root, "symbolic-ref", "refs/cache/local-alias", gitTest(t, root, "symbolic-ref", "HEAD"))
	gitTest(t, root, "symbolic-ref", "refs/cache/dangling-alias", "refs/heads/missing-branch")
	assertBranchSyncTrackingQueryPreservesState(t, root, func() {
		for _, namespace := range []string{
			"refs/heads",
			"refs/tags",
			"refs/notes",
			"refs/replace",
			"refs/bisect",
			"refs/rewritten",
			"refs/worktree",
			"refs/namespaces",
			"refs/stash",
		} {
			t.Run(namespace, func(t *testing.T) {
				for _, ref := range []string{namespace, namespace + "/child"} {
					assert.Error(t, commands.checkBranchSyncTrackingWrite(ref), "ref %q", ref)
				}
			})
		}
		assert.Error(t, commands.checkBranchSyncTrackingWrite("refs/cache/local-alias"))
		assert.Error(t, commands.checkBranchSyncTrackingWrite("refs/cache/dangling-alias"))
	})
}

func TestBranchSyncTrackingWriteAcceptsTrackingRefs(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	gitTest(t, root, "update-ref", "refs/cache/existing", oid)
	assertBranchSyncTrackingQueryPreservesState(t, root, func() {
		for _, ref := range []string{
			"refs/cache/missing",
			"refs/cache/custom/topic",
			"refs/remotes/custom/topic",
		} {
			assert.NoError(t, commands.checkBranchSyncTrackingWrite(ref), "ref %q", ref)
		}

		assert.NoError(t, commands.checkBranchSyncTrackingWrite("refs/cache/existing"))
	})
}

func TestBranchSyncTrackingWriteRejectsAbbreviatedAndMalformedRefs(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	assertBranchSyncTrackingQueryPreservesState(t, root, func() {
		for _, ref := range []string{
			"HEAD/foo",
			"refs/cache/a..b",
			"refs/cache/trailing/",
			"refs/cache//double",
			"refs/cache/with\x00nul",
			"refs/cache/with\x01control",
		} {
			assert.Error(t, commands.checkBranchSyncTrackingWrite(ref), "ref %q", ref)
		}
	})
}

func assertBranchSyncTrackingQueryPreservesState(t *testing.T, root string, query func()) {
	t.Helper()
	head := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	index := mustRead(t, filepath.Join(root, ".git", "index"))
	refs := gitTest(t, root, "for-each-ref", "--format=%(refname)%00%(objectname)%00%(symref)")
	query()
	assert.True(t, bytes.Equal(head, mustRead(t, filepath.Join(root, ".git", "HEAD"))), "HEAD bytes changed")
	assert.True(t, bytes.Equal(index, mustRead(t, filepath.Join(root, ".git", "index"))), "index bytes changed")
	assert.Equal(t, refs, gitTest(t, root, "for-each-ref", "--format=%(refname)%00%(objectname)%00%(symref)"), "refs changed")
	assert.Equal(t, "", gitTest(t, root, "status", "--porcelain"), "fixture worktree should remain clean")
}
