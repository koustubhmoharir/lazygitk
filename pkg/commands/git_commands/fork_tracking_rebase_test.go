package git_commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackingRebaseCandidatesOrdersUpstreamFirstAndSkipsMissingRefs(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "feature")
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/origin.git")
	gitTest(t, root, "remote", "add", "fork", "https://example.invalid/fork.git")
	gitTest(t, root, "remote", "add", "alpha", "https://example.invalid/alpha.git")
	gitTest(t, root, "config", "branch.feature.remote", "origin")
	gitTest(t, root, "config", "branch.feature.merge", "refs/heads/feature")
	gitTest(t, root, "update-ref", "refs/remotes/origin/feature", oid)
	gitTest(t, root, "update-ref", "refs/remotes/fork/feature", oid)
	// alpha has no feature branch, so it contributes nothing.

	candidates, err := branchSyncTestCommands(t, root).TrackingRebaseCandidates("feature")
	if err != nil {
		t.Fatal(err)
	}
	// origin/feature is both the upstream and a same-named remote ref; it appears once, first.
	assert.Equal(t, []TrackingRebaseCandidate{
		{Ref: "refs/remotes/origin/feature", Upstream: true},
		{Ref: "refs/remotes/fork/feature", Upstream: false},
	}, candidates)
}

func TestTrackingRebaseCandidatesOmitsMissingUpstreamRef(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "feature")
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/origin.git")
	gitTest(t, root, "remote", "add", "fork", "https://example.invalid/fork.git")
	gitTest(t, root, "config", "branch.feature.remote", "origin")
	gitTest(t, root, "config", "branch.feature.merge", "refs/heads/feature")
	gitTest(t, root, "update-ref", "refs/remotes/fork/feature", oid)

	candidates, err := branchSyncTestCommands(t, root).TrackingRebaseCandidates("feature")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []TrackingRebaseCandidate{{Ref: "refs/remotes/fork/feature"}}, candidates)
}

func TestTrackingRebaseCandidatesEmptyWithoutTrackingRefs(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "feature")

	candidates, err := branchSyncTestCommands(t, root).TrackingRebaseCandidates("feature")
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, candidates)
}
