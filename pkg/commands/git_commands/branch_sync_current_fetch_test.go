package git_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCurrentBranchFetchRequiresExactNativeHeadAndRetainsMode(t *testing.T) {
	h := branchFetchTestHistory(t)
	commands := branchSyncTestCommands(t, h.root)
	_, err := commands.PrepareCurrentBranchFetch("selected", h.selected, h.options())
	assert.Error(t, err, "the selected source is off-branch")

	gitTest(t, h.root, "checkout", "-q", "selected")
	currentPlan, err := commands.PrepareCurrentBranchFetch("selected", h.selected, h.options())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commands.BranchFetchCommand(currentPlan, nil); err != nil {
		t.Fatal(err)
	}
	offBranchPlan, err := commands.PrepareBranchFetch("selected", h.selected, h.options())
	assert.Error(t, err, "the existing method must still reject native HEAD")
	assert.Empty(t, offBranchPlan)

	gitTest(t, h.root, "checkout", "-q", "master")
	_, err = commands.BranchFetchCommand(currentPlan, nil)
	assert.Error(t, err, "a current plan cannot be demoted after preparation")
	_, err = commands.PrepareFetchedBranchUpdate(currentPlan)
	assert.Error(t, err, "current plans cannot become ref-only updates")

	offPlan, err := commands.PrepareBranchFetch("selected", h.selected, h.options())
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, h.root, "checkout", "-q", "selected")
	_, err = commands.BranchFetchCommand(offPlan, nil)
	assert.Error(t, err, "an off-branch plan cannot be promoted to current mode")
}

func TestCurrentBranchFetchPreservesDirtyStateAndReturnsCapturedCommitTip(t *testing.T) {
	h := branchFetchTestHistory(t)
	gitTest(t, h.root, "checkout", "-q", "selected")
	writeTestFile(t, filepath.Join(h.root, "server-next"), []byte("remote only\n"), 0o644)
	gitTest(t, h.root, "add", "server-next")
	gitTest(t, h.root, "commit", "-qm", "server advances selected")
	serverOID := gitTest(t, h.root, "rev-parse", "HEAD")
	gitTest(t, h.root, "push", "-q", h.bare, "refs/heads/selected:refs/heads/selected")
	gitTest(t, h.root, "checkout", "-q", "master")
	gitTest(t, h.root, "update-ref", "refs/heads/selected", h.selected, serverOID)
	gitTest(t, h.root, "checkout", "-q", "selected")
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("staged\n"), 0o644)
	gitTest(t, h.root, "add", "tracked")
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("unstaged\n"), 0o644)
	writeTestFile(t, filepath.Join(h.root, "untracked"), []byte("untracked\n"), 0o600)
	gitDir := branchFetchGit(t, h.root, "rev-parse", "--absolute-git-dir")
	headBefore := mustRead(t, filepath.Join(gitDir, "HEAD"))
	branchBefore := gitTest(t, h.root, "rev-parse", "refs/heads/selected")
	assert.Equal(t, h.selected, branchBefore, "the local current branch remains at its original source commit")
	indexBefore := mustRead(t, filepath.Join(gitDir, "index"))
	fetchHeadPath := filepath.Join(gitDir, "FETCH_HEAD")
	fetchHeadBefore := []byte("keep current FETCH_HEAD\n")
	if err := os.WriteFile(fetchHeadPath, fetchHeadBefore, 0o600); err != nil {
		t.Fatal(err)
	}

	commands := branchSyncTestCommands(t, h.root)
	plan, err := commands.PrepareCurrentBranchFetch("selected", h.selected, h.options())
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := commands.BranchFetchCommand(plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--refmap=", "--no-tags", "--no-prune", "--no-prune-tags", "--no-recurse-submodules", "--no-write-fetch-head", "+refs/heads/selected:refs/remotes/custom/selected"} {
		assert.Contains(t, cmd.Args(), flag)
	}
	output, err := cmd.GetCmd().CombinedOutput()
	if err != nil {
		t.Fatalf("current branch fetch failed: %v\n%s", err, output)
	}
	tip, err := commands.FetchedBranchTip(plan)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, serverOID, tip, "fetch returns the newer server commit for review")
	assert.NotEqual(t, branchBefore, tip, "the fetched tip must be newer than the current branch source")
	assert.Equal(t, branchBefore, gitTest(t, h.root, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(gitDir, "HEAD")))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(gitDir, "index")))
	assert.Equal(t, "unstaged\n", string(mustRead(t, filepath.Join(h.root, "tracked"))))
	assert.Equal(t, "staged", gitTest(t, h.root, "show", ":tracked"))
	assert.Equal(t, "untracked\n", string(mustRead(t, filepath.Join(h.root, "untracked"))))
	assert.Equal(t, fetchHeadBefore, mustRead(t, fetchHeadPath))

	changedTip := branchPlacementTestCommit(t, h.root, "after-fetch", "after\n", "after fetch")
	gitTest(t, h.root, "update-ref", plan.options.TrackingRef, changedTip, tip)
	assert.Equal(t, serverOID, tip, "the returned value is a full commit OID snapshot, not a live ref alias")
}

func TestCurrentBranchFetchRevalidationAndOwnership(t *testing.T) {
	h := branchFetchTestHistory(t)
	gitTest(t, h.root, "checkout", "-q", "selected")
	commands := branchSyncTestCommands(t, h.root)
	plan, err := commands.PrepareCurrentBranchFetch("selected", h.selected, h.options())
	if err != nil {
		t.Fatal(err)
	}
	_, err = branchSyncTestCommands(t, h.root).BranchFetchCommand(plan, nil)
	assert.Error(t, err, "plans are owned by their creating backend")

	gitTest(t, h.root, "checkout", "-q", "master")
	_, err = commands.BranchFetchCommand(plan, nil)
	assert.Error(t, err, "native HEAD switch must refuse before fetch")
	gitTest(t, h.root, "checkout", "-q", "selected")
	changed := branchPlacementTestCommit(t, h.root, "changed-source", "changed\n", "changed source")
	_, err = commands.BranchFetchCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)

	plan, err = commands.PrepareCurrentBranchFetch("selected", changed, h.options())
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, h.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/remotes/custom-map/*")
	_, err = commands.BranchFetchCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged, "remote profile changes refuse before fetch")
}
