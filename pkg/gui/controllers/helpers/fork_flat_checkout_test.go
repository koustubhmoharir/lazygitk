package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

// These tests drive the flat checkout handlers against a real repository. The
// fixture's main is at f.oid, and branch other is at f.other with a changed
// tracked file.

// enterCheckoutFixtureRepo makes the fixture repository the working directory,
// as the app runs Git there, so a stray command cannot touch the test process's
// own repository.
func enterCheckoutFixtureRepo(t *testing.T, f *namedCheckoutFixture) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(f.root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

// startPendingMerge leaves a real merge in progress: MERGE_HEAD exists and the
// merge is not committed.
func startPendingMerge(t *testing.T, f *namedCheckoutFixture) {
	t.Helper()
	namedCheckoutGit(t, f.root, "merge", "--no-ff", "--no-commit", "other")
	if _, err := os.Stat(filepath.Join(f.root, ".git", "MERGE_HEAD")); err != nil {
		t.Fatalf("expected MERGE_HEAD after a pending merge: %v", err)
	}
}

// startConflictedRebase leaves a real rebase in progress that stopped on a conflict.
func startConflictedRebase(t *testing.T, f *namedCheckoutFixture) {
	t.Helper()
	namedCheckoutWrite(t, f.root, "tracked", "main edit\n")
	namedCheckoutGit(t, f.root, "commit", "-qam", "main edit")
	// Git exits non-zero on the conflict, which is the state under test.
	_ = exec.Command("git", "-C", f.root, "rebase", "other").Run()
	_, errMerge := os.Stat(filepath.Join(f.root, ".git", "rebase-merge"))
	_, errApply := os.Stat(filepath.Join(f.root, ".git", "rebase-apply"))
	if errMerge != nil && errApply != nil {
		t.Fatal("expected a rebase in progress after a conflicting rebase")
	}
}

func assertNoLocalBranch(t *testing.T, f *namedCheckoutFixture, name string) {
	t.Helper()
	assert.Equal(t, "", namedCheckoutGit(t, f.root, "branch", "--list", name), "no branch %q should exist", name)
}

func addOriginTopic(t *testing.T, f *namedCheckoutFixture, at string) {
	t.Helper()
	namedCheckoutGit(t, f.root, "remote", "add", "origin", "https://example.invalid/repo.git")
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/topic", at)
}

func TestDetachedCheckoutRefusesPendingMerge(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	startPendingMerge(t, f)
	headRef, head := checkoutByNameHead(t, f)

	if err := (&RefsHelper{c: f.c}).CheckoutSelectedRefDetached("refs/heads/other"); err != nil {
		t.Fatal(err)
	}
	err := drainCheckoutByName(t, ui)
	if err == nil {
		t.Fatal("detached checkout must refuse during a merge")
	}
	assert.Contains(t, err.Error(), "merge")
	assert.FileExists(t, filepath.Join(f.root, ".git", "MERGE_HEAD"))
	ref, after := checkoutByNameHead(t, f)
	assert.Equal(t, headRef, ref)
	assert.Equal(t, head, after)
}

func TestDetachedCheckoutRefusesRebaseInProgress(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	startConflictedRebase(t, f)
	_, head := checkoutByNameHead(t, f)

	if err := (&RefsHelper{c: f.c}).CheckoutCommitDetached(f.other); err != nil {
		t.Fatal(err)
	}
	err := drainCheckoutByName(t, ui)
	if err == nil {
		t.Fatal("detached checkout must refuse during a rebase")
	}
	assert.Contains(t, err.Error(), "rebase")
	_, after := checkoutByNameHead(t, f)
	assert.Equal(t, head, after)
	assert.True(t, fileExists(filepath.Join(f.root, ".git", "rebase-merge")) || fileExists(filepath.Join(f.root, ".git", "rebase-apply")))
}

func TestPreviousDetachedRouteRefusesPendingMerge(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	// The previous checkout is a commit: main was left at a detached f.other.
	namedCheckoutGit(t, f.root, "checkout", "-q", "--detach", f.other)
	namedCheckoutGit(t, f.root, "checkout", "-q", "main")
	startPendingMerge(t, f)

	if err := (&RefsHelper{c: f.c}).CheckoutPreviousFlat(); err != nil {
		t.Fatal(err)
	}
	err := drainCheckoutByName(t, ui)
	if err == nil {
		t.Fatal("the detached route of s p must refuse during a merge")
	}
	assert.Contains(t, err.Error(), "merge")
	assert.FileExists(t, filepath.Join(f.root, ".git", "MERGE_HEAD"))
}

func TestRemoteCheckoutCreatesTrackingBranchAndChecksItOut(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	addOriginTopic(t, f, f.other)

	err := (&RefsHelper{c: f.c}).CheckoutSelectedRemoteBranch(&models.RemoteBranch{RemoteName: "origin", Name: "topic"})
	if err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	ref, head := checkoutByNameHead(t, f)
	assert.Equal(t, "refs/heads/topic", ref)
	assert.Equal(t, f.other, head)
	assert.Equal(t, "origin", namedCheckoutGit(t, f.root, "config", "branch.topic.remote"))
	assert.Equal(t, "refs/heads/topic", namedCheckoutGit(t, f.root, "config", "branch.topic.merge"))
	assert.Equal(t, "origin/topic", namedCheckoutGit(t, f.root, "rev-parse", "--abbrev-ref", "topic@{upstream}"))
}

func TestRemoteCheckoutUsesRemoteRefWhenSameNamedTagExists(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	addOriginTopic(t, f, f.other)
	// origin/topic as a start point would name this tag, not the remote branch.
	namedCheckoutGit(t, f.root, "tag", "origin/topic", f.oid)

	if err := (&RefsHelper{c: f.c}).CheckoutSelectedRemoteBranch(&models.RemoteBranch{RemoteName: "origin", Name: "topic"}); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, f.other, namedCheckoutGit(t, f.root, "rev-parse", "refs/heads/topic"))
	// The abbreviated name is ambiguous here, so compare the full upstream ref.
	assert.Equal(t, "refs/remotes/origin/topic", namedCheckoutGit(t, f.root, "rev-parse", "--symbolic-full-name", "topic@{upstream}"))
}

func TestRemoteCheckoutRefusesPendingMergeWithoutCreatingBranch(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	addOriginTopic(t, f, f.other)
	startPendingMerge(t, f)
	ref, head := checkoutByNameHead(t, f)

	if err := (&RefsHelper{c: f.c}).CheckoutSelectedRemoteBranch(&models.RemoteBranch{RemoteName: "origin", Name: "topic"}); err != nil {
		t.Fatal(err)
	}
	err := drainCheckoutByName(t, ui)
	if err == nil {
		t.Fatal("remote checkout must refuse during a merge")
	}
	assert.Contains(t, err.Error(), "merge")
	assertNoLocalBranch(t, f, "topic")
	assert.FileExists(t, filepath.Join(f.root, ".git", "MERGE_HEAD"))
	afterRef, after := checkoutByNameHead(t, f)
	assert.Equal(t, ref, afterRef)
	assert.Equal(t, head, after)
}

func TestRemoteCheckoutCollidingChangeCreatesNoBranch(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	addOriginTopic(t, f, f.other)
	// The remote branch changes tracked, and this uncommitted edit would be overwritten.
	namedCheckoutWrite(t, f.root, "tracked", "local edits\n")

	if err := (&RefsHelper{c: f.c}).CheckoutSelectedRemoteBranch(&models.RemoteBranch{RemoteName: "origin", Name: "topic"}); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err == nil {
		t.Fatal("remote checkout must be refused when a tracked change would be overwritten")
	}
	assertNoLocalBranch(t, f, "topic")
	_, head := checkoutByNameHead(t, f)
	assert.Equal(t, f.oid, head)
	content, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "local edits\n", string(content))
}

func TestRemoteCheckoutRefusesRemoteMovedAfterPrepare(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	addOriginTopic(t, f, f.other)

	if err := (&RefsHelper{c: f.c}).CheckoutSelectedRemoteBranch(&models.RemoteBranch{RemoteName: "origin", Name: "topic"}); err != nil {
		t.Fatal(err)
	}
	// The worker prepares the checkout for the captured OID. The remote then moves
	// before the checkout is applied.
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/topic", f.oid)

	err := drainCheckoutByName(t, ui)
	if err == nil {
		t.Fatal("checkout must refuse a remote branch that moved after it was captured")
	}
	assert.Contains(t, err.Error(), "remote branch changed")
	assertNoLocalBranch(t, f, "topic")
	_, head := checkoutByNameHead(t, f)
	assert.Equal(t, f.oid, head)
}

func TestPreviousCheckoutFollowsBranchWhenTagHasItsName(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	// From main to other, so the previous checkout is main. A tag named main
	// makes @{-1} ambiguous.
	namedCheckoutGit(t, f.root, "checkout", "-q", "other")
	namedCheckoutGit(t, f.root, "tag", "main", f.oid)

	if err := (&RefsHelper{c: f.c}).CheckoutPreviousFlat(); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	ref, head := checkoutByNameHead(t, f)
	assert.Equal(t, "refs/heads/main", ref, "the previous branch must be checked out, not detached")
	assert.Equal(t, f.oid, head)
}

func TestPreviousCheckoutDetachesWhenSourceWasCommit(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	namedCheckoutGit(t, f.root, "checkout", "-q", "--detach", f.other)
	namedCheckoutGit(t, f.root, "checkout", "-q", "main")

	if err := (&RefsHelper{c: f.c}).CheckoutPreviousFlat(); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	ref, head := checkoutByNameHead(t, f)
	assert.Equal(t, "", ref, "HEAD should be detached at the previous commit")
	assert.Equal(t, f.other, head)
}

func TestPreviousCheckoutRefusesDeletedPreviousBranch(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	namedCheckoutGit(t, f.root, "checkout", "-q", "other")
	namedCheckoutGit(t, f.root, "branch", "-D", "main")
	ref, head := checkoutByNameHead(t, f)

	err := (&RefsHelper{c: f.c}).CheckoutPreviousFlat()
	if err == nil {
		err = drainCheckoutByName(t, ui)
	}
	if err == nil {
		t.Fatal("a previous branch that no longer exists must be refused")
	}
	assert.Contains(t, err.Error(), "main", "the refusal should name the missing previous branch")
	afterRef, after := checkoutByNameHead(t, f)
	assert.Equal(t, ref, afterRef)
	assert.Equal(t, head, after)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
