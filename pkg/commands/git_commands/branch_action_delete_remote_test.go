package git_commands

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func remoteDeleteRequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func remoteDeleteFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	root, _, _ := transactionHistory(t)
	topic := branchPlacementTestCommit(t, root, "topic-file", "topic\n", "topic")
	head := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected", topic)
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, filepath.Dir(bare), "init", "--bare", "-q", bare)
	gitTest(t, root, "push", "-q", bare, "refs/heads/selected:refs/heads/selected")
	gitTest(t, root, "remote", "add", "origin", bare)
	gitTest(t, root, "update-ref", "refs/remotes/origin/selected", topic)
	return root, bare, topic, head
}

func remoteDeletePrepare(t *testing.T, root, sourceOID string) (*SyncCommands, BranchActionRemoteDeletePlan) {
	t.Helper()
	commands := branchSyncTestCommands(t, root)
	target, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/heads/selected", sourceOID)
	remoteDeleteRequireNoError(t, err)
	plan, err := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{{Target: target, Remote: "origin", RemoteRef: "refs/heads/selected"}})
	remoteDeleteRequireNoError(t, err)
	return commands, plan
}

func TestBranchActionRemoteDeletePreservesDirtyOffBranchState(t *testing.T) {
	root, bare, source, head := remoteDeleteFixture(t)
	tracked := filepath.Join(root, "tracked")
	writeTestFile(t, tracked, []byte("staged\x00bytes\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	writeTestFile(t, tracked, []byte("unstaged\xffbytes\n"), 0o600)
	untracked := filepath.Join(root, "untracked")
	writeTestFile(t, untracked, []byte("untracked\x00\xfe\n"), 0o600)
	index := mustRead(t, filepath.Join(root, ".git", "index"))
	trackedBytes, untrackedBytes := mustRead(t, tracked), mustRead(t, untracked)
	commands, plan := remoteDeletePrepare(t, root, source)
	err := commands.RunBranchActionRemoteDelete(plan, nil, nil)
	assert.ErrorIs(t, err, ErrBranchRemoteDeleteApprovalRequired)
	assert.Equal(t, source, gitTest(t, bare, "rev-parse", "refs/heads/selected"))
	err = commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
	remoteDeleteRequireNoError(t, err)
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, index, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, trackedBytes, mustRead(t, tracked))
	assert.Equal(t, untrackedBytes, mustRead(t, untracked))
	assert.Empty(t, gitTest(t, bare, "for-each-ref", "--format=%(refname)", "refs/heads/selected"))
	assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/heads/selected"))
}

func TestBranchActionRemoteDeleteExplicitDestinationAndDeduplicatedSelection(t *testing.T) {
	root, bare, source, _ := remoteDeleteFixture(t)
	gitTest(t, root, "branch", "unrelated-local", source)
	gitTest(t, bare, "update-ref", "refs/heads/unrelated", source)
	gitTest(t, root, "tag", "keep-tag", source)
	gitTest(t, root, "tag", "-a", "reachable-annotated", "-m", "reachable", source)
	gitTest(t, root, "config", "remote.origin.mirror", "true")
	gitTest(t, root, "config", "push.followTags", "true")
	gitTest(t, root, "config", "push.recurseSubmodules", "on-demand")
	gitTest(t, root, "config", "--add", "remote.origin.push", "refs/heads/selected:refs/heads/unwanted-mapped")
	commands := branchSyncTestCommands(t, root)
	target, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/remotes/origin/selected", source)
	remoteDeleteRequireNoError(t, err)
	selections := []BranchActionRemoteDeleteSelection{
		{Target: target, Remote: "origin", RemoteRef: "refs/heads/selected"},
		{Target: target, Remote: "origin", RemoteRef: "refs/heads/selected"},
	}
	plan, err := commands.PrepareBranchActionRemoteDelete(selections)
	remoteDeleteRequireNoError(t, err)
	got := plan.Selections()
	assert.Len(t, got, 2)
	got[0].RemoteRef = "refs/heads/changed"
	assert.Equal(t, "refs/heads/selected", plan.Selections()[0].RemoteRef)
	refs := plan.TrackingRefs()
	originalRefs := plan.TrackingRefs()
	if assert.NotEmpty(t, refs) {
		refs[0] = "refs/heads/changed"
	}
	assert.Equal(t, originalRefs, plan.TrackingRefs())
	err = commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
	remoteDeleteRequireNoError(t, err)
	assert.Empty(t, gitTest(t, bare, "for-each-ref", "--format=%(refname)", "refs/heads/selected"))
	assert.Equal(t, source, gitTest(t, bare, "rev-parse", "refs/heads/unrelated"))
	assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/heads/unrelated-local"))
	assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/tags/keep-tag"))
	assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/tags/reachable-annotated^{}"))
	assert.Empty(t, gitTest(t, bare, "for-each-ref", "--format=%(refname)", "refs/heads/unwanted-mapped"))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", "remote.origin.mirror"))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", "push.followTags"))
}

func TestBranchActionRemoteDeleteStopsOnGuardAndNativePartialFailure(t *testing.T) {
	root, firstBare, source, _ := remoteDeleteFixture(t)
	secondBare := filepath.Join(t.TempDir(), "second.git")
	gitTest(t, filepath.Dir(secondBare), "init", "--bare", "-q", secondBare)
	gitTest(t, root, "push", "-q", secondBare, "refs/heads/selected:refs/heads/selected")
	commands := branchSyncTestCommands(t, root)
	target, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/heads/selected", source)
	remoteDeleteRequireNoError(t, err)
	// Configure the second endpoint as a separate named remote to exercise real
	// sequential native pushes and preserve the first endpoint's partial success.
	gitTest(t, firstBare, "update-ref", "refs/heads/selected", source)
	gitTest(t, root, "remote", "add", "second", secondBare)
	gitTest(t, root, "update-ref", "refs/remotes/second/selected", source)
	secondTarget, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/remotes/second/selected", source)
	remoteDeleteRequireNoError(t, err)
	plan, err := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{
		{Target: target, Remote: "origin", RemoteRef: "refs/heads/selected"},
		{Target: secondTarget, Remote: "second", RemoteRef: "refs/heads/selected"},
	})
	remoteDeleteRequireNoError(t, err)
	assert.NoError(t, os.MkdirAll(filepath.Join(secondBare, "hooks"), 0o755))
	hook := filepath.Join(secondBare, "hooks", "pre-receive")
	remoteDeleteRequireNoError(t, os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755))
	err = commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "pre-receive hook declined")
	assert.Empty(t, gitTest(t, firstBare, "for-each-ref", "--format=%(refname)", "refs/heads/selected"))
	assert.Equal(t, source, gitTest(t, secondBare, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/heads/selected"), "remote deletion never removes the local source")
}

func TestBranchActionRemoteDeleteRejectsStaleSnapshotsBeforePush(t *testing.T) {
	for _, mutation := range []string{"source", "head", "config", "foreign backend"} {
		t.Run(mutation, func(t *testing.T) {
			root, bare, source, _ := remoteDeleteFixture(t)
			commands, plan := remoteDeletePrepare(t, root, source)
			switch mutation {
			case "source":
				other := branchPlacementTestCommit(t, root, "other", "other\n", "other")
				gitTest(t, root, "update-ref", "refs/heads/selected", other, source)
			case "head":
				gitTest(t, root, "checkout", "--detach", "-q", source)
			case "config":
				gitTest(t, root, "config", "remote.origin.url", filepath.Join(t.TempDir(), "different.git"))
			case "foreign backend":
				commands = branchSyncTestCommands(t, root)
			}
			index := mustRead(t, filepath.Join(root, ".git", "index"))
			tracked := mustRead(t, filepath.Join(root, "tracked"))
			err := commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
			assert.Error(t, err)
			assert.Equal(t, source, gitTest(t, bare, "rev-parse", "refs/heads/selected"))
			assert.Equal(t, index, mustRead(t, filepath.Join(root, ".git", "index")))
			assert.Equal(t, tracked, mustRead(t, filepath.Join(root, "tracked")))
		})
	}
}

func TestBranchActionRemoteDeleteRejectsProtectedTrackingDestinations(t *testing.T) {
	for _, mapping := range []string{"branch destination", "symbolic alias"} {
		t.Run(mapping, func(t *testing.T) {
			root, bare, source, _ := remoteDeleteFixture(t)
			gitTest(t, root, "config", "--replace-all", "remote.origin.fetch", "+refs/heads/*:refs/heads/*")
			if mapping == "symbolic alias" {
				gitTest(t, root, "config", "--replace-all", "remote.origin.fetch", "+refs/heads/*:refs/cache/origin/*")
				gitTest(t, root, "symbolic-ref", "refs/cache/origin/selected", "refs/heads/selected")
			}
			commands := branchSyncTestCommands(t, root)
			target, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/heads/selected", source)
			remoteDeleteRequireNoError(t, err)
			_, err = commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{{Target: target, Remote: "origin", RemoteRef: "refs/heads/selected"}})
			assert.Error(t, err)
			assert.Equal(t, source, gitTest(t, bare, "rev-parse", "refs/heads/selected"))
		})
	}
}

func TestBranchActionRemoteDeletePushURLPartialFailureIsNative(t *testing.T) {
	root, first, source, _ := remoteDeleteFixture(t)
	second := filepath.Join(t.TempDir(), "second.git")
	gitTest(t, filepath.Dir(second), "init", "--bare", "-q", second)
	gitTest(t, root, "push", "-q", second, "refs/heads/selected:refs/heads/selected")
	gitTest(t, root, "config", "--add", "remote.origin.pushurl", first)
	gitTest(t, root, "config", "--add", "remote.origin.pushurl", second)
	hook := filepath.Join(second, "hooks", "pre-receive")
	remoteDeleteRequireNoError(t, os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755))
	commands, plan := remoteDeletePrepare(t, root, source)
	err := commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "pre-receive hook declined")
	assert.Empty(t, gitTest(t, first, "for-each-ref", "--format=%(refname)", "refs/heads/selected"), "native push result: %v", err)
	assert.Equal(t, source, gitTest(t, second, "rev-parse", "refs/heads/selected"))
}

func TestBranchActionRemoteDeleteCachedSourceAndGuardBoundary(t *testing.T) {
	root, first, source, _ := remoteDeleteFixture(t)
	second := filepath.Join(t.TempDir(), "second.git")
	gitTest(t, filepath.Dir(second), "init", "--bare", "-q", second)
	gitTest(t, root, "push", "-q", second, "refs/heads/selected:refs/heads/selected")
	gitTest(t, root, "remote", "add", "second", second)
	gitTest(t, root, "update-ref", "refs/remotes/second/selected", source)
	commands := branchSyncTestCommands(t, root)
	firstTarget, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/remotes/origin/selected", source)
	remoteDeleteRequireNoError(t, err)
	secondTarget, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/remotes/second/selected", source)
	remoteDeleteRequireNoError(t, err)
	plan, err := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{
		{Target: firstTarget, Remote: "origin", RemoteRef: "refs/heads/selected"},
		{Target: secondTarget, Remote: "second", RemoteRef: "refs/heads/selected"},
	})
	remoteDeleteRequireNoError(t, err)
	calls := 0
	guardErr := errors.New("repository changed between remote groups")
	err = commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, func() error {
		calls++
		if calls == 5 {
			return guardErr
		}
		return nil
	})
	assert.ErrorIs(t, err, guardErr)
	assert.Empty(t, gitTest(t, first, "for-each-ref", "--format=%(refname)", "refs/heads/selected"))
	assert.Equal(t, source, gitTest(t, second, "rev-parse", "refs/heads/selected"))
	assert.Error(t, os.ErrNotExist, func() error {
		_, err := os.Stat(filepath.Join(root, ".git", "refs", "remotes", "origin", "selected"))
		return err
	}())

	// The same group boundary also rechecks effective config and HEAD after a
	// successful first push, while accepting deletion of its cached source ref.
	for _, changed := range []string{"config", "HEAD"} {
		t.Run("boundary rechecks "+changed, func(t *testing.T) {
			root, first, source, _ := remoteDeleteFixture(t)
			second := filepath.Join(t.TempDir(), "second.git")
			gitTest(t, filepath.Dir(second), "init", "--bare", "-q", second)
			gitTest(t, root, "push", "-q", second, "refs/heads/selected:refs/heads/selected")
			gitTest(t, root, "remote", "add", "second", second)
			gitTest(t, root, "update-ref", "refs/remotes/second/selected", source)
			commands := branchSyncTestCommands(t, root)
			one, e := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/remotes/origin/selected", source)
			remoteDeleteRequireNoError(t, e)
			two, e := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/remotes/second/selected", source)
			remoteDeleteRequireNoError(t, e)
			plan, e := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{{Target: one, Remote: "origin", RemoteRef: "refs/heads/selected"}, {Target: two, Remote: "second", RemoteRef: "refs/heads/selected"}})
			remoteDeleteRequireNoError(t, e)
			calls := 0
			e = commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, func() error {
				calls++
				if calls == 5 {
					if changed == "config" {
						gitTest(t, root, "config", "remote.second.url", second+"-changed")
					} else {
						gitTest(t, root, "checkout", "--detach", "-q", source)
					}
				}
				return nil
			})
			assert.Error(t, e)
			assert.Empty(t, gitTest(t, first, "for-each-ref", "--format=%(refname)", "refs/heads/selected"))
			assert.Equal(t, source, gitTest(t, second, "rev-parse", "refs/heads/selected"))
		})
	}
}

func TestBranchActionRemoteDeleteRejectsInvalidDestinations(t *testing.T) {
	root, _, source, _ := remoteDeleteFixture(t)
	commands := branchSyncTestCommands(t, root)
	target, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/heads/selected", source)
	remoteDeleteRequireNoError(t, err)
	for _, destination := range []string{".", "https://example.invalid/repo", "refs/tags/selected"} {
		_, err := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{{Target: target, Remote: "origin", RemoteRef: destination}})
		assert.Error(t, err, "destination %q", destination)
	}
	for _, remote := range []string{".", "https://example.invalid/repo"} {
		_, err := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{{Target: target, Remote: remote, RemoteRef: "refs/heads/selected"}})
		assert.Error(t, err, "remote %q", remote)
		if err != nil {
			assert.NotContains(t, strings.ToLower(err.Error()), "credential")
		}
	}
}

func TestBranchActionRemoteDeleteCachedSourceOwnPushSucceeds(t *testing.T) {
	root, _, source, _ := remoteDeleteFixture(t)
	commands := branchSyncTestCommands(t, root)
	target, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/remotes/origin/selected", source)
	remoteDeleteRequireNoError(t, err)
	plan, err := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{{Target: target, Remote: "origin", RemoteRef: "refs/heads/selected"}})
	remoteDeleteRequireNoError(t, err)
	err = commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
	assert.NoError(t, err)
	assert.Error(t, os.ErrNotExist, func() error {
		_, err := os.Stat(filepath.Join(root, ".git", "refs", "remotes", "origin", "selected"))
		return err
	}())
}

func TestBranchActionRemoteDeleteDoesNotRetryLockLikePrePushFailure(t *testing.T) {
	root, bare, source, _ := remoteDeleteFixture(t)
	counter := filepath.Join(t.TempDir(), "hook-count")
	hook := filepath.Join(root, ".git", "hooks", "pre-push")
	script := "#!/bin/sh\nprintf x >> " + strconv.Quote(counter) + "\necho 'my .lock file already exists' >&2\nexit 1\n"
	remoteDeleteRequireNoError(t, os.WriteFile(hook, []byte(script), 0o755))
	commands, plan := remoteDeletePrepare(t, root, source)
	err := commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
	assert.Error(t, err)
	count, readErr := os.ReadFile(counter)
	assert.NoError(t, readErr)
	assert.Equal(t, 1, len(count), "DisableLockRetries must prevent retrying hook output that resembles a lock error")
	assert.Equal(t, source, gitTest(t, bare, "rev-parse", "refs/heads/selected"))
}

func TestBranchActionRemoteDeleteLiteralEqualsRemoteName(t *testing.T) {
	root, bare, source, _ := remoteDeleteFixture(t)
	gitTest(t, root, "remote", "rename", "origin", "peer=one")
	gitTest(t, root, "config", "remote.peer=one.mirror", "true")
	gitTest(t, bare, "update-ref", "refs/heads/unrelated", source)
	commands := branchSyncTestCommands(t, root)
	target, err := NewBranchCommands(commands.GitCommon).PrepareBranchActionTarget("refs/heads/selected", source)
	remoteDeleteRequireNoError(t, err)
	plan, err := commands.PrepareBranchActionRemoteDelete([]BranchActionRemoteDeleteSelection{{Target: target, Remote: "peer=one", RemoteRef: "refs/heads/selected"}})
	remoteDeleteRequireNoError(t, err)
	err = commands.RunBranchActionRemoteDelete(plan.WithDeletionApproved(), nil, nil)
	assert.NoError(t, err)
	assert.Empty(t, gitTest(t, bare, "for-each-ref", "--format=%(refname)", "refs/heads/selected"))
	assert.Equal(t, source, gitTest(t, bare, "rev-parse", "refs/heads/unrelated"))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", "remote.peer=one.mirror"))
}
