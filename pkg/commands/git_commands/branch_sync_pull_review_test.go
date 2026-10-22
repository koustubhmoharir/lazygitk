package git_commands

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func pullReviewNoError(t *testing.T, err error, message ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v %v", err, message)
	}
}

func pullReviewEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if !assert.Equal(t, expected, actual) {
		t.FailNow()
	}
}

func pullReviewContains(t *testing.T, list []string, value string) {
	t.Helper()
	if !assert.Contains(t, list, value) {
		t.FailNow()
	}
}

func prepareFetchedPullReview(t *testing.T, root, source, target string, choice PullIntegrationMethod) (*SyncCommands, CurrentPullIntegrationPlan) {
	t.Helper()
	commands, review, fetched := fetchCurrentPullReview(t, root, source, target)
	plan, err := commands.PrepareCurrentPullIntegration(review, fetched, choice)
	pullReviewNoError(t, err)
	return commands, plan
}

func fetchCurrentPullReview(t *testing.T, root, source, expectedTarget string) (*SyncCommands, CurrentPullReview, string) {
	t.Helper()
	commands := branchSyncTestCommands(t, root)
	review, err := commands.PrepareCurrentPullReview("master", source, currentPullOptions().TrackingRef)
	pullReviewNoError(t, err)
	fetch, err := commands.PrepareCurrentBranchFetch("master", source, currentPullOptions())
	pullReviewNoError(t, err)
	cmd, err := commands.BranchFetchCommand(fetch, nil)
	pullReviewNoError(t, err)
	output, err := cmd.GetCmd().CombinedOutput()
	pullReviewNoError(t, err, output)
	fetched, err := commands.FetchedBranchTip(fetch)
	pullReviewNoError(t, err)
	if expectedTarget != "" {
		pullReviewEqual(t, expectedTarget, fetched)
	}
	return commands, review, fetched
}

func runReviewedPull(t *testing.T, commands *SyncCommands, plan CurrentPullIntegrationPlan) ([]byte, error) {
	t.Helper()
	cmd, err := commands.CurrentPullIntegrationCommand(plan)
	if err != nil {
		return nil, err
	}
	assertReviewedPullCommand(t, cmd)
	return cmd.GetCmd().CombinedOutput()
}

func assertReviewedPullCommand(t *testing.T, cmd interface {
	Args() []string
	LockRetriesDisabled() bool
},
) {
	t.Helper()
	args := cmd.Args()
	assert.Contains(t, args, "--no-autostash")
	assert.Contains(t, args, "submodule.recurse=false")
	assert.True(t, cmd.LockRetriesDisabled())
	assert.True(t, slices.Contains(args, "merge") || slices.Contains(args, "rebase"))
	assert.NotContains(t, args, "pull")
	assert.NotContains(t, args, "fetch")
}

func TestCurrentPullReviewFetchPinsTargetAndLeavesCheckoutAlone(t *testing.T) {
	root, bare, source := currentPullFixture(t)
	target := advanceCurrentPullRemote(t, bare, "incoming", "incoming\n")
	head := gitTest(t, root, "rev-parse", "HEAD")
	index := gitTest(t, root, "write-tree")
	commands := branchSyncTestCommands(t, root)
	review, err := commands.PrepareCurrentPullReview("master", source, currentPullOptions().TrackingRef)
	pullReviewNoError(t, err)
	fetch, err := commands.PrepareCurrentBranchFetch("master", source, currentPullOptions())
	pullReviewNoError(t, err)
	cmd, err := commands.BranchFetchCommand(fetch, nil)
	pullReviewNoError(t, err)
	output, err := cmd.GetCmd().CombinedOutput()
	pullReviewNoError(t, err, output)
	fetched, err := commands.FetchedBranchTip(fetch)
	pullReviewNoError(t, err)
	assert.Equal(t, target, fetched)
	plan, err := commands.PrepareCurrentPullIntegration(review, fetched, PullIntegrationDefault)
	pullReviewNoError(t, err)
	assert.Equal(t, target, plan.TargetOID())
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, index, gitTest(t, root, "write-tree"))
	assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/heads/master"))
	assert.FileExists(t, filepath.Join(root, "tracked"))
}

func TestCurrentPullReviewLocalUpstreamCapturesDirectRefOID(t *testing.T) {
	root, _, source := currentPullFixture(t)
	upstreamRef := "refs/heads/local-upstream"
	gitTest(t, root, "update-ref", upstreamRef, source)
	commands := branchSyncTestCommands(t, root)
	headBefore := gitTest(t, root, "rev-parse", "HEAD")
	review, targetOID, err := commands.PrepareLocalCurrentPullReview("master", source, upstreamRef)
	pullReviewNoError(t, err)
	assert.Equal(t, source, targetOID)
	assert.Equal(t, headBefore, gitTest(t, root, "rev-parse", "HEAD"))

	movedOID := gitTest(t, root, "rev-parse", source+"^")
	gitTest(t, root, "update-ref", upstreamRef, movedOID, targetOID)
	plan, err := commands.PrepareCurrentPullIntegration(review, targetOID, PullIntegrationMerge)
	pullReviewNoError(t, err)
	cmd, err := commands.CurrentPullIntegrationCommand(plan)
	pullReviewNoError(t, err)
	assert.Contains(t, cmd.Args(), targetOID+"^{commit}")
	assert.NotContains(t, cmd.Args(), movedOID+"^{commit}")
	assert.Equal(t, headBefore, gitTest(t, root, "rev-parse", "HEAD"))

	symbolicRef := "refs/heads/symbolic-upstream"
	gitTest(t, root, "symbolic-ref", symbolicRef, upstreamRef)
	_, _, err = commands.PrepareLocalCurrentPullReview("master", source, symbolicRef)
	assert.Error(t, err, "symbolic local upstream refs must not be followed")
}

func TestCurrentPullReviewPolicyAndExplicitChoices(t *testing.T) {
	t.Run("branch rebase overrides pull merge", func(t *testing.T) {
		root, bare, _ := currentPullFixture(t)
		gitTest(t, root, "config", "pull.rebase", "false")
		gitTest(t, root, "config", "branch.master.rebase", "true")
		branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
		advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		_, plan := prepareFetchedPullReview(t, root, gitTest(t, root, "rev-parse", "HEAD"), "", PullIntegrationDefault)
		assert.Equal(t, PullIntegrationRebase, plan.Method())
	})
	t.Run("pull ff only overrides configured rebase", func(t *testing.T) {
		root, bare, _ := currentPullFixture(t)
		gitTest(t, root, "config", "pull.rebase", "true")
		gitTest(t, root, "config", "pull.ff", "only")
		branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
		advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		localTip := gitTest(t, root, "rev-parse", "HEAD")
		commands, review, target := fetchCurrentPullReview(t, root, localTip, "")
		_, err := commands.PrepareCurrentPullIntegration(review, target, PullIntegrationDefault)
		assert.ErrorIs(t, err, ErrBranchSyncDiverged)
		assert.Equal(t, localTip, gitTest(t, root, "rev-parse", "HEAD"))
	})
	t.Run("pull ff false forces merge commit", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "pull.ff", "false")
		target := advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		commands, plan := prepareFetchedPullReview(t, root, source, target, PullIntegrationDefault)
		assert.Equal(t, PullIntegrationMergeNoFF, plan.Method())
		_, err := runReviewedPull(t, commands, plan)
		pullReviewNoError(t, err)
		assert.Equal(t, 3, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
	})
	t.Run("merge ff fallback", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "merge.ff", "false")
		target := advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		_, plan := prepareFetchedPullReview(t, root, source, target, PullIntegrationDefault)
		assert.Equal(t, PullIntegrationMergeNoFF, plan.Method())
	})
	t.Run("no divergence policy requires choice", func(t *testing.T) {
		root, bare, _ := currentPullFixture(t)
		local := branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
		advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		commands := branchSyncTestCommands(t, root)
		review, err := commands.PrepareCurrentPullReview("master", local, currentPullOptions().TrackingRef)
		pullReviewNoError(t, err)
		fetch, err := commands.PrepareCurrentBranchFetch("master", local, currentPullOptions())
		pullReviewNoError(t, err)
		cmd, err := commands.BranchFetchCommand(fetch, nil)
		pullReviewNoError(t, err)
		_, err = cmd.GetCmd().CombinedOutput()
		pullReviewNoError(t, err)
		tip, err := commands.FetchedBranchTip(fetch)
		pullReviewNoError(t, err)
		_, err = commands.PrepareCurrentPullIntegration(review, tip, PullIntegrationDefault)
		assert.ErrorIs(t, err, ErrPullIntegrationChoiceRequired)
	})
}

func TestCurrentPullReviewDefaultRebasePreservesConfiguredOptions(t *testing.T) {
	root, bare, _ := currentPullFixture(t)
	gitTest(t, root, "config", "pull.rebase", "true")
	gitTest(t, root, "config", "rebase.rebaseMerges", "true")
	gitTest(t, root, "config", "rebase.autoSquash", "true")
	local := branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
	advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
	commands, review, target := fetchCurrentPullReview(t, root, local, "")
	defaultPlan, err := commands.PrepareCurrentPullIntegration(review, target, PullIntegrationDefault)
	pullReviewNoError(t, err)
	assert.Equal(t, PullIntegrationRebase, defaultPlan.Method())
	assert.True(t, defaultPlan.UsesConfiguredRebaseOptions())
	defaultCmd, err := commands.CurrentPullIntegrationCommand(defaultPlan)
	pullReviewNoError(t, err)
	assertReviewedPullCommand(t, defaultCmd)
	assert.NotContains(t, defaultCmd.Args(), "--no-rebase-merges")
	assert.NotContains(t, defaultCmd.Args(), "--no-autosquash")
	assert.Contains(t, defaultCmd.Args(), "rebase.updateRefs=false")

	explicitPlan, err := commands.PrepareCurrentPullIntegration(review, target, PullIntegrationRebase)
	pullReviewNoError(t, err)
	assert.False(t, explicitPlan.UsesConfiguredRebaseOptions())
	explicitCmd, err := commands.CurrentPullIntegrationCommand(explicitPlan)
	pullReviewNoError(t, err)
	assertReviewedPullCommand(t, explicitCmd)
	assert.Contains(t, explicitCmd.Args(), "--no-rebase-merges")
	assert.Contains(t, explicitCmd.Args(), "--no-autosquash")
	assert.Contains(t, explicitCmd.Args(), "rebase.updateRefs=false")
}

func TestCurrentPullReviewConfiguredMergeOptions(t *testing.T) {
	t.Run("default branch no-ff is exposed and creates merge commit on fast-forward", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "merge.ff", "true")
		gitTest(t, root, "config", "branch.master.mergeOptions", "--no-ff")
		target := advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		commands, plan := prepareFetchedPullReview(t, root, source, target, PullIntegrationDefault)
		assert.Equal(t, PullIntegrationConfiguredMerge, plan.Method())
		assert.Equal(t, "--no-ff", plan.MergeOptions())
		assert.Empty(t, plan.ConfiguredMergeFF())
		output, err := runReviewedPull(t, commands, plan)
		pullReviewNoError(t, err, output)
		assert.Equal(t, 3, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
	})

	t.Run("merge ff only is exposed but branch no-ff bundle wins", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "merge.ff", "only")
		gitTest(t, root, "config", "branch.master.mergeOptions", "--no-ff")
		target := advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		commands, plan := prepareFetchedPullReview(t, root, source, target, PullIntegrationDefault)
		assert.Equal(t, PullIntegrationConfiguredMerge, plan.Method())
		assert.Equal(t, "--no-ff", plan.MergeOptions())
		assert.Empty(t, plan.ConfiguredMergeFF())
		output, err := runReviewedPull(t, commands, plan)
		pullReviewNoError(t, err, output)
		assert.Equal(t, 3, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
	})

	t.Run("branch no-ff overrides pull ff only", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "pull.ff", "only")
		gitTest(t, root, "config", "branch.master.mergeOptions", "--no-ff")
		local := branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
		target := advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		commands, review, fetched := fetchCurrentPullReview(t, root, local, target)
		_, err := commands.PrepareCurrentPullIntegration(review, fetched, PullIntegrationDefault)
		assert.ErrorIs(t, err, ErrBranchSyncDiverged, "pull.ff=only must reject divergence before branch merge options can trigger an ignored-file review")
		assert.Equal(t, local, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, source, gitTest(t, root, "rev-parse", "HEAD~1"))
	})

	t.Run("pull ff true overrides branch no-ff", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "pull.ff", "true")
		gitTest(t, root, "config", "branch.master.mergeOptions", "--no-ff")
		target := advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		commands, plan := prepareFetchedPullReview(t, root, source, target, PullIntegrationDefault)
		assert.Equal(t, PullIntegrationConfiguredMerge, plan.Method())
		assert.Equal(t, "--no-ff", plan.MergeOptions())
		assert.Equal(t, "true", plan.ConfiguredMergeFF())
		output, err := runReviewedPull(t, commands, plan)
		pullReviewNoError(t, err, output)
		assert.Equal(t, 2, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
	})

	t.Run("explicit merge replaces bundle for command without changing config", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "branch.master.mergeOptions", "--squash --no-commit")
		target := advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		configPath := filepath.Join(root, ".git", "config")
		before, err := os.ReadFile(configPath)
		pullReviewNoError(t, err)
		commands, plan := prepareFetchedPullReview(t, root, source, target, PullIntegrationMerge)
		cmd, err := commands.CurrentPullIntegrationCommand(plan)
		pullReviewNoError(t, err)
		assertReviewedPullCommand(t, cmd)
		assert.Contains(t, cmd.Args(), "branch.master.mergeOptions=")
		output, err := cmd.GetCmd().CombinedOutput()
		pullReviewNoError(t, err, output)
		after, err := os.ReadFile(configPath)
		pullReviewNoError(t, err)
		assert.Equal(t, before, after)
		assert.Equal(t, 2, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
	})
}

func TestCurrentPullReviewRebaseAndMergeDirtRules(t *testing.T) {
	t.Run("rebase rejects tracked dirt and allows untracked", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "pull.rebase", "true")
		advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		writeTestFile(t, filepath.Join(root, "harmless-untracked"), []byte("keep\n"), 0o600)
		commands, plan := prepareFetchedPullReview(t, root, source, "", PullIntegrationDefault)
		output, err := runReviewedPull(t, commands, plan)
		pullReviewNoError(t, err, output)
		assert.FileExists(t, filepath.Join(root, "harmless-untracked"))
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
		review, err := commands.PrepareCurrentPullReview("master", gitTest(t, root, "rev-parse", "HEAD"), currentPullOptions().TrackingRef)
		pullReviewNoError(t, err)
		_, err = commands.PrepareCurrentPullIntegration(review, gitTest(t, root, "rev-parse", "refs/remotes/custom/master"), PullIntegrationRebase)
		assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	})
	t.Run("explicit merge permits dirty files without autostash", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		gitTest(t, root, "config", "merge.autostash", "true")
		advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
		commands, plan := prepareFetchedPullReview(t, root, source, "", PullIntegrationMerge)
		cmd, err := commands.CurrentPullIntegrationCommand(plan)
		pullReviewNoError(t, err)
		assertReviewedPullCommand(t, cmd)
		assert.Contains(t, cmd.Args(), "--no-autostash")
		assert.Contains(t, cmd.Args(), "merge.autostash=false")
		output, err := cmd.GetCmd().CombinedOutput()
		pullReviewNoError(t, err, output)
		assert.Equal(t, "dirty\n", string(mustRead(t, filepath.Join(root, "tracked"))))
	})
}

func TestCurrentPullReviewDoesNotRefreshIndexStatCache(t *testing.T) {
	root, bare, initial := currentPullFixture(t)
	gitTest(t, root, "config", "core.logAllRefUpdates", "true")
	gitTest(t, root, "update-ref", currentPullOptions().TrackingRef, initial)
	local := branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
	advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
	trackedPath := filepath.Join(root, "tracked")
	info, err := os.Stat(trackedPath)
	pullReviewNoError(t, err)
	changedTime := info.ModTime().Add(time.Hour)
	pullReviewNoError(t, os.Chtimes(trackedPath, changedTime, changedTime))
	indexPath := filepath.Join(root, ".git", "index")
	indexBefore, err := os.ReadFile(indexPath)
	pullReviewNoError(t, err)
	commands := branchSyncTestCommands(t, root)
	review, err := commands.PrepareCurrentPullReview("master", local, currentPullOptions().TrackingRef)
	pullReviewNoError(t, err)
	fetch, err := commands.PrepareCurrentBranchFetch("master", local, currentPullOptions())
	pullReviewNoError(t, err)
	cmd, err := commands.BranchFetchCommand(fetch, nil)
	pullReviewNoError(t, err)
	output, err := cmd.GetCmd().CombinedOutput()
	pullReviewNoError(t, err, output)
	target, err := commands.FetchedBranchTip(fetch)
	pullReviewNoError(t, err)
	plan, err := commands.PrepareCurrentPullIntegration(review, target, PullIntegrationRebase)
	pullReviewNoError(t, err)
	cmd, err = commands.CurrentPullIntegrationCommand(plan)
	pullReviewNoError(t, err)
	assertReviewedPullCommand(t, cmd)
	indexAfter, err := os.ReadFile(indexPath)
	pullReviewNoError(t, err)
	assert.Equal(t, indexBefore, indexAfter)
}

func TestCurrentPullReviewRejectsStaleInputsAndPinsTarget(t *testing.T) {
	t.Run("tracking ref movement never retargets plan", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		first := advanceCurrentPullRemote(t, bare, "first", "first\n")
		commands, plan := prepareFetchedPullReview(t, root, source, first, PullIntegrationMerge)
		gitTest(t, root, "update-ref", currentPullOptions().TrackingRef, source)
		cmd, err := commands.CurrentPullIntegrationCommand(plan)
		pullReviewNoError(t, err)
		assert.Contains(t, cmd.Args(), first+"^{commit}")
		assert.NotContains(t, cmd.Args(), source+"^{commit}")
	})
	t.Run("owner head source and policy changes reject", func(t *testing.T) {
		root, _, source := currentPullFixture(t)
		commands := branchSyncTestCommands(t, root)
		review, err := commands.PrepareCurrentPullReview("master", source, currentPullOptions().TrackingRef)
		pullReviewNoError(t, err)
		_, err = branchSyncTestCommands(t, root).PrepareCurrentPullIntegration(review, source, PullIntegrationMerge)
		assert.Error(t, err)
		other, err := commands.PrepareCurrentPullIntegration(review, source, PullIntegrationMerge)
		pullReviewNoError(t, err)
		gitTest(t, root, "config", "pull.ff", "false")
		_, err = commands.CurrentPullIntegrationCommand(other)
		assert.ErrorIs(t, err, ErrBranchSyncChanged)
	})
	t.Run("source move invalidates captured review", func(t *testing.T) {
		root, _, source := currentPullFixture(t)
		commands := branchSyncTestCommands(t, root)
		review, err := commands.PrepareCurrentPullReview("master", source, "")
		pullReviewNoError(t, err)
		changed := branchPlacementTestCommit(t, root, "changed", "changed\n", "changed")
		_, err = commands.PrepareCurrentPullIntegration(review, changed, PullIntegrationMerge)
		assert.ErrorIs(t, err, ErrBranchSyncChanged)
	})
	t.Run("head switch invalidates captured review", func(t *testing.T) {
		root, _, source := currentPullFixture(t)
		commands := branchSyncTestCommands(t, root)
		review, err := commands.PrepareCurrentPullReview("master", source, "")
		pullReviewNoError(t, err)
		gitTest(t, root, "branch", "other")
		gitTest(t, root, "checkout", "-q", "other")
		_, err = commands.PrepareCurrentPullIntegration(review, source, PullIntegrationMerge)
		assert.ErrorIs(t, err, ErrBranchSyncChanged)
	})
}

func TestCurrentPullReviewReplaysOnlyLocalCommitsAfterUpstreamRewrite(t *testing.T) {
	root, bare, oldUpstream := currentPullFixture(t)
	gitTest(t, root, "config", "pull.rebase", "true")
	gitTest(t, root, "update-ref", currentPullOptions().TrackingRef, oldUpstream)
	gitTest(t, root, "config", "core.logAllRefUpdates", "true")
	gitTest(t, root, "update-ref", currentPullOptions().TrackingRef, oldUpstream)
	local := branchPlacementTestCommit(t, root, "local-only", "local\n", "local change")
	gitTest(t, root, "branch", "other-local", local)
	gitTest(t, root, "config", "rebase.updateRefs", "true")
	oldTree := gitTest(t, root, "rev-parse", oldUpstream+"^{tree}")
	reviewCommands := branchSyncTestCommands(t, root)
	review, err := reviewCommands.PrepareCurrentPullReview("master", local, currentPullOptions().TrackingRef)
	pullReviewNoError(t, err)
	server := filepath.Join(t.TempDir(), "rewrite")
	gitTest(t, filepath.Dir(server), "clone", "-q", bare, server)
	gitTest(t, server, "reset", "--hard", "HEAD~1")
	gitTest(t, server, "push", "-q", "--force", "origin", "HEAD:refs/heads/master")
	target := gitTest(t, server, "rev-parse", "HEAD")
	fetch, err := reviewCommands.PrepareCurrentBranchFetch("master", local, currentPullOptions())
	pullReviewNoError(t, err)
	cmd, err := reviewCommands.BranchFetchCommand(fetch, nil)
	pullReviewNoError(t, err)
	output, err := cmd.GetCmd().CombinedOutput()
	pullReviewNoError(t, err, output)
	tip, err := reviewCommands.FetchedBranchTip(fetch)
	pullReviewNoError(t, err)
	pullReviewEqual(t, target, tip)
	plan, err := reviewCommands.PrepareCurrentPullIntegration(review, tip, PullIntegrationRebase)
	pullReviewNoError(t, err)
	assert.Equal(t, oldUpstream, plan.ReplayBaseOID())
	cmd, err = reviewCommands.CurrentPullIntegrationCommand(plan)
	pullReviewNoError(t, err)
	assertReviewedPullCommand(t, cmd)
	assert.Contains(t, cmd.Args(), oldUpstream+"^{commit}")
	output, err = cmd.GetCmd().CombinedOutput()
	pullReviewNoError(t, err, output)
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "HEAD^"))
	assert.Equal(t, "1", gitTest(t, root, "rev-list", "--count", target+"..HEAD"))
	assert.Equal(t, "local change", gitTest(t, root, "log", "-1", "--format=%s"))
	assert.Equal(t, "one\n", string(mustRead(t, filepath.Join(root, "tracked"))), "the dropped upstream commit's tree must not be replayed")
	assert.Equal(t, local, gitTest(t, root, "rev-parse", "refs/heads/other-local"), "rebase.updateRefs must not move another branch")
	assert.NotEqual(t, oldTree, gitTest(t, root, "rev-parse", "HEAD^{tree}"))
	assert.Equal(t, "local\n", string(mustRead(t, filepath.Join(root, "local-only"))))
}

func TestCurrentPullReviewIgnoredApprovalAndChangedCollision(t *testing.T) {
	t.Run("already-ahead explicit merge has no ignored-file approval risk", func(t *testing.T) {
		root, _, source := currentPullFixture(t)
		local := branchPlacementTestCommit(t, root, "local-only", "local\n", "local ahead")
		writeTestFile(t, filepath.Join(root, "ignore-rules"), []byte("ignored\n"), 0o600)
		gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore-rules"))
		writeTestFile(t, filepath.Join(root, "ignored"), []byte("leave alone\n"), 0o600)
		commands := branchSyncTestCommands(t, root)
		review, err := commands.PrepareCurrentPullReview("master", local, currentPullOptions().TrackingRef)
		pullReviewNoError(t, err)
		plan, err := commands.PrepareCurrentPullIntegration(review, source, PullIntegrationMerge)
		pullReviewNoError(t, err)
		assert.Empty(t, plan.IgnoredFiles())
		cmd, err := commands.CurrentPullIntegrationCommand(plan)
		pullReviewNoError(t, err)
		assertReviewedPullCommand(t, cmd)
		output, err := cmd.GetCmd().CombinedOutput()
		pullReviewNoError(t, err, output)
		assert.Equal(t, local, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, "leave alone\n", string(mustRead(t, filepath.Join(root, "ignored"))))
	})

	for _, method := range []PullIntegrationMethod{PullIntegrationFastForwardOnly, PullIntegrationMerge, PullIntegrationRebase} {
		t.Run(map[PullIntegrationMethod]string{
			PullIntegrationFastForwardOnly: "fast-forward",
			PullIntegrationMerge:           "merge",
			PullIntegrationRebase:          "rebase",
		}[method], func(t *testing.T) {
			root, bare, source := currentPullFixture(t)
			if method == PullIntegrationMerge || method == PullIntegrationRebase {
				gitTest(t, root, "config", "core.logAllRefUpdates", "true")
				gitTest(t, root, "update-ref", currentPullOptions().TrackingRef, source)
				source = branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
			}
			advanceCurrentPullRemote(t, bare, "ignored", "incoming ignored path\n")
			gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore-rules"))
			writeTestFile(t, filepath.Join(root, "ignore-rules"), []byte("ignored\n"), 0o600)
			writeTestFile(t, filepath.Join(root, "ignored"), []byte("preserve\n"), 0o600)
			commands, plan := prepareFetchedPullReview(t, root, source, "", method)
			pullReviewContains(t, plan.IgnoredFiles(), "ignored")
			_, err := commands.CurrentPullIntegrationCommand(plan)
			assert.ErrorIs(t, err, ErrPullIgnoredApprovalRequired)
			approved := plan.WithIgnoredFilesApproved()
			cmd, err := commands.CurrentPullIntegrationCommand(approved)
			pullReviewNoError(t, err)
			assertReviewedPullCommand(t, cmd)
			output, err := cmd.GetCmd().CombinedOutput()
			pullReviewNoError(t, err, output)
			assert.Equal(t, "incoming ignored path\n", string(mustRead(t, filepath.Join(root, "ignored"))))
			switch method {
			case PullIntegrationFastForwardOnly:
				assert.Equal(t, plan.TargetOID(), gitTest(t, root, "rev-parse", "HEAD"))
			case PullIntegrationMerge:
				assert.Equal(t, 3, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
			case PullIntegrationRebase:
				assert.Equal(t, plan.TargetOID(), gitTest(t, root, "rev-parse", "HEAD^"))
				assert.Equal(t, "local divergence", gitTest(t, root, "log", "-1", "--format=%s"))
			default:
				t.Fatalf("unexpected test integration method: %d", method)
			}
		})
	}

	t.Run("changed collision after approval refuses before execution", func(t *testing.T) {
		root, bare, source := currentPullFixture(t)
		advanceCurrentPullRemote(t, bare, "ignored", "commit\n")
		gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore-rules"))
		writeTestFile(t, filepath.Join(root, "ignore-rules"), []byte("ignored\n"), 0o600)
		writeTestFile(t, filepath.Join(root, "ignored"), []byte("preserve\n"), 0o600)
		commands, plan := prepareFetchedPullReview(t, root, source, "", PullIntegrationMerge)
		pullReviewContains(t, plan.IgnoredFiles(), "ignored")
		approved := plan.WithIgnoredFilesApproved()
		writeTestFile(t, filepath.Join(root, "ignored"), []byte("changed since approval\n"), 0o600)
		_, err := commands.CurrentPullIntegrationCommand(approved)
		assert.ErrorIs(t, err, ErrBranchSyncChanged)
		assert.Equal(t, "changed since approval\n", string(mustRead(t, filepath.Join(root, "ignored"))))
	})
}
