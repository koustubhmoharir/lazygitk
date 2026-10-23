package git_commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func literalMergeFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root, base, head := transactionHistory(t)
	gitTest(t, root, "branch", "-m", "master", "current=one")
	gitTest(t, root, "checkout", "-qb", "incoming", base)
	branchPlacementTestCommit(t, root, "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "current=one")
	return root, head, gitTest(t, root, "rev-parse", "refs/heads/incoming")
}

func TestGitLiteralConfigActionsMergeDoesNotDuplicateBranchOptions(t *testing.T) {
	root, _, incoming := literalMergeFixture(t)
	key := "branch.current=one.mergeOptions"
	gitTest(t, root, "config", key, "--no-ff --message=reviewed")
	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/heads/incoming", incoming)
	if !assert.NoError(t, err) {
		return
	}
	review, err := branches.PrepareBranchActionMergeReview(target, "")
	if !assert.NoError(t, err) {
		return
	}
	plan, err := branches.PrepareBranchActionMerge(review, BranchActionMergeConfigured, "fallback")
	if !assert.NoError(t, err) {
		return
	}
	cmd, err := branches.BranchActionMergeCommand(plan)
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, cmd.Run())
	assert.Equal(t, "reviewed", gitTest(t, root, "show", "-s", "--format=%B", "HEAD"))
	assert.Len(t, strings.Fields(gitTest(t, root, "show", "-s", "--format=%P", "HEAD")), 2)
	assert.Equal(t, "--no-ff --message=reviewed", gitTest(t, root, "config", "--get", key))
	assert.Equal(t, incoming, gitTest(t, root, "rev-parse", "refs/heads/incoming"))
}

func TestGitLiteralConfigActionsPullChoiceClearsBranchOptions(t *testing.T) {
	root, head, incoming := literalMergeFixture(t)
	key := "branch.current=one.mergeOptions"
	gitTest(t, root, "config", key, "--squash")
	sync := branchSyncTestCommands(t, root)
	review, target, err := sync.PrepareLocalCurrentPullReview("current=one", head, "refs/heads/incoming")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, incoming, target)
	plan, err := sync.PrepareCurrentPullIntegration(review, target, PullIntegrationMergeNoFF)
	if !assert.NoError(t, err) {
		return
	}
	cmd, err := sync.CurrentPullIntegrationCommand(plan)
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, cmd.Run())
	assert.Len(t, strings.Fields(gitTest(t, root, "show", "-s", "--format=%P", "HEAD")), 2)
	assert.Equal(t, "--squash", gitTest(t, root, "config", "--get", key))
	assert.Equal(t, incoming, gitTest(t, root, "rev-parse", "refs/heads/incoming"))
}

func TestGitLiteralConfigActionsPullPreservesTagPolicyForEqualsRemote(t *testing.T) {
	root, bare, head := currentPullFixture(t)
	gitTest(t, root, "remote", "rename", "origin", "peer=one")
	gitTest(t, root, "config", "remote.peer=one.pruneTags", "true")
	gitTest(t, root, "config", "remote.peer=one.tagOpt", "--tags")
	gitTest(t, root, "config", "pull.rebase", "false")
	gitTest(t, bare, "tag", "server-tag", "refs/heads/master")
	sync := branchSyncTestCommands(t, root)
	plan, err := sync.PrepareCurrentBranchPull("master", head, BranchFetchOptions{
		Remote: "peer=one", RemoteRef: "refs/heads/master", TrackingRef: "refs/remotes/custom/master",
	})
	if !assert.NoError(t, err) {
		return
	}
	cmd, err := sync.CurrentBranchPullCommand(plan, nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, cmd.Run())
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Empty(t, gitTest(t, root, "for-each-ref", "--format=%(refname)", "refs/tags/server-tag"))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", "remote.peer=one.pruneTags"))
}
