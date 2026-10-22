package git_commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

type branchFetchHistory struct {
	root, bare, base, selected string
}

func branchFetchTestHistory(t *testing.T) branchFetchHistory {
	t.Helper()
	root, base, _ := transactionHistory(t)
	bare := filepath.Join(t.TempDir(), "origin.git")
	gitTest(t, filepath.Dir(bare), "init", "--bare", "-q", bare)
	gitTest(t, root, "branch", "selected", base)
	gitTest(t, root, "push", "-q", bare, "refs/heads/selected:refs/heads/selected")
	gitTest(t, root, "remote", "add", "origin", bare)
	gitTest(t, root, "checkout", "-q", "selected")
	selected := branchPlacementTestCommit(t, root, "selected-file", "selected\n", "selected")
	gitTest(t, root, "checkout", "-q", "master")
	return branchFetchHistory{root: root, bare: bare, base: base, selected: selected}
}

func (h branchFetchHistory) options() BranchFetchOptions {
	return BranchFetchOptions{Remote: "origin", RemoteRef: "refs/heads/selected", TrackingRef: "refs/remotes/custom/selected"}
}

func branchFetchPlan(t *testing.T, h branchFetchHistory, opts BranchFetchOptions) (*SyncCommands, BranchFetchPlan) {
	t.Helper()
	commands := branchSyncTestCommands(t, h.root)
	plan, err := commands.PrepareBranchFetch("selected", h.selected, opts)
	if err != nil {
		t.Fatal(err)
	}
	return commands, plan
}

func runBranchFetch(t *testing.T, commands *SyncCommands, plan BranchFetchPlan) ([]byte, error) {
	t.Helper()
	cmd, err := commands.BranchFetchCommand(plan, nil)
	if err != nil {
		return nil, err
	}
	return cmd.GetCmd().CombinedOutput()
}

func branchFetchGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func branchFetchRefExists(t *testing.T, dir, ref string) bool {
	t.Helper()
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", ref)
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return true
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return false
	}
	t.Fatalf("git show-ref --verify --quiet %s failed: %v", ref, err)
	return false
}

func TestBranchFetchExplicitRefOnlyAndThenFastForwardsSelectedBranch(t *testing.T) {
	h := branchFetchTestHistory(t)
	gitTest(t, h.root, "update-ref", "refs/remotes/custom/selected", h.base)
	gitTest(t, h.root, "update-ref", "refs/remotes/origin/selected", h.base)
	gitTest(t, h.root, "update-ref", "refs/remotes/custom/unrelated", h.base)
	gitTest(t, h.root, "update-ref", "refs/remotes/overlap/selected", h.base)
	gitTest(t, h.root, "tag", "preserve-tag", h.base)
	gitTest(t, h.root, "config", "fetch.prune", "true")
	gitTest(t, h.root, "config", "fetch.pruneTags", "true")
	gitTest(t, h.root, "config", "remote.origin.pruneTags", "true")
	gitTest(t, h.root, "config", "remote.origin.tagOpt", "--tags")
	gitTest(t, h.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/remotes/overlap/*")
	gitTest(t, h.root, "update-ref", "refs/remotes/overlap/unrelated", h.base)
	gitTest(t, h.root, "tag", "server-tag", h.selected)
	gitTest(t, h.root, "push", "-q", h.bare, "refs/heads/selected:refs/heads/selected", "refs/tags/server-tag:refs/tags/server-tag")
	gitTest(t, h.root, "tag", "-d", "server-tag")
	gitTest(t, h.root, "update-ref", "refs/heads/selected", h.base, h.selected)
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("staged\n"), 0o644)
	gitTest(t, h.root, "add", "tracked")
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("unstaged\n"), 0o644)
	writeTestFile(t, filepath.Join(h.root, "untracked"), []byte("untracked\n"), 0o600)
	gitDir := filepath.Join(h.root, ".git")
	masterBefore := gitTest(t, h.root, "rev-parse", "refs/heads/master")
	headBefore, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	indexBefore, err := os.ReadFile(filepath.Join(gitDir, "index"))
	if err != nil {
		t.Fatal(err)
	}
	fetchHeadBefore := []byte("preserve this existing FETCH_HEAD sentinel\n")
	if err := os.WriteFile(filepath.Join(gitDir, "FETCH_HEAD"), fetchHeadBefore, 0o600); err != nil {
		t.Fatal(err)
	}
	commands := branchSyncTestCommands(t, h.root)
	plan, err := commands.PrepareBranchFetch("selected", h.base, h.options())
	if err != nil {
		t.Fatal(err)
	}
	cmdObj, err := commands.BranchFetchCommand(plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	args := cmdObj.Args()
	for _, flag := range []string{"--refmap=", "--no-tags", "--no-prune", "--no-prune-tags", "--no-recurse-submodules", "--no-write-fetch-head", "+refs/heads/selected:refs/remotes/custom/selected"} {
		assert.Contains(t, args, flag)
	}
	assert.Equal(t, oscommands.PROMPT, cmdObj.GetCredentialStrategy())
	assert.True(t, cmdObj.LockRetriesDisabled())
	output, err := cmdObj.GetCmd().CombinedOutput()
	if err != nil {
		t.Fatalf("fetch failed: %v\n%s", err, output)
	}
	headAfter, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	indexAfter, err := os.ReadFile(filepath.Join(gitDir, "index"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, headBefore, headAfter)
	assert.Equal(t, indexBefore, indexAfter)
	assert.Equal(t, h.selected, branchFetchGit(t, h.root, "rev-parse", "refs/remotes/custom/selected"))
	assert.Equal(t, h.base, branchFetchGit(t, h.root, "rev-parse", "refs/remotes/origin/selected"))
	assert.Equal(t, h.base, branchFetchGit(t, h.root, "rev-parse", "refs/remotes/custom/unrelated"))
	assert.Equal(t, h.base, branchFetchGit(t, h.root, "rev-parse", "refs/remotes/overlap/selected"))
	assert.Equal(t, h.base, branchFetchGit(t, h.root, "rev-parse", "refs/remotes/overlap/unrelated"))
	assert.False(t, branchFetchRefExists(t, h.root, "refs/tags/server-tag"))
	assert.Equal(t, h.base, branchFetchGit(t, h.root, "rev-parse", "refs/tags/preserve-tag"))
	assert.Equal(t, fetchHeadBefore, mustRead(t, filepath.Join(gitDir, "FETCH_HEAD")))
	assert.Equal(t, "unstaged\n", string(mustRead(t, filepath.Join(h.root, "tracked"))))
	assert.Equal(t, "staged", gitTest(t, h.root, "show", ":tracked"))
	assert.Equal(t, "untracked\n", string(mustRead(t, filepath.Join(h.root, "untracked"))))

	update, err := commands.PrepareFetchedBranchUpdate(plan)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, update.NeedsUpdate(), "the fetched target must really advance the selected local branch")
	if err := commands.ApplyBranchFastForward(update); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, h.selected, gitTest(t, h.root, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, masterBefore, gitTest(t, h.root, "rev-parse", "refs/heads/master"))
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(gitDir, "HEAD")))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(gitDir, "index")))
	assert.Equal(t, "unstaged\n", string(mustRead(t, filepath.Join(h.root, "tracked"))))
	assert.Equal(t, "staged", gitTest(t, h.root, "show", ":tracked"))
	assert.Equal(t, "untracked\n", string(mustRead(t, filepath.Join(h.root, "untracked"))))
}

func TestBranchFetchValidationAndRevalidation(t *testing.T) {
	h := branchFetchTestHistory(t)
	commands := branchSyncTestCommands(t, h.root)
	opts := h.options()
	for _, tc := range []struct{ ref, storage string }{
		{"refs/tags/tag", opts.TrackingRef},
		{"refs/heads/", opts.TrackingRef},
		{opts.RemoteRef, "refs/heads/unsafe"},
		{opts.RemoteRef, "refs/cache/../x"},
		{opts.RemoteRef, "refs/cache/trailing/"},
	} {
		bad := opts
		bad.RemoteRef, bad.TrackingRef = tc.ref, tc.storage
		_, err := commands.PrepareBranchFetch("selected", h.selected, bad)
		assert.Error(t, err)
	}
	_, err := commands.PrepareBranchFetch("selected", h.selected[:12], opts)
	assert.Error(t, err)
	_, err = commands.PrepareBranchFetch("master", gitTest(t, h.root, "rev-parse", "master"), opts)
	assert.ErrorContains(t, err, "off-branch")
	gitTest(t, h.root, "remote", "add", "push-only", h.bare)
	gitTest(t, h.root, "config", "--unset-all", "remote.push-only.url")
	gitTest(t, h.root, "config", "remote.push-only.pushurl", h.bare)
	_, err = commands.PrepareBranchFetch("selected", h.selected, BranchFetchOptions{Remote: "push-only", RemoteRef: opts.RemoteRef, TrackingRef: opts.TrackingRef})
	assert.ErrorContains(t, err, "no configured fetch URL")

	commands, plan := branchFetchPlan(t, h, opts)
	foreign, err := branchSyncTestCommands(t, h.root).BranchFetchCommand(plan, nil)
	assert.Nil(t, foreign)
	assert.Error(t, err)
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/cache/changed/*")
	_, err = commands.BranchFetchCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	gitTest(t, h.root, "config", "--unset-all", "remote.origin.fetch")
	changed := branchPlacementTestCommit(t, h.root, "other", "change\n", "change selected")
	gitTest(t, h.root, "update-ref", "refs/heads/selected", changed, h.selected)
	_, err = commands.BranchFetchCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)

	gitTest(t, h.root, "update-ref", "refs/heads/selected", h.selected, changed)
	_, plan = branchFetchPlan(t, h, opts)
	gitTest(t, h.root, "checkout", "-q", "selected")
	_, err = commands.PrepareFetchedBranchUpdate(plan)
	assert.Error(t, err)
}

func TestBranchFetchFailureDivergenceAndOccupiedReservation(t *testing.T) {
	h := branchFetchTestHistory(t)
	commands, plan := branchFetchPlan(t, h, h.options())
	gitTest(t, h.root, "push", "-q", h.bare, ":refs/heads/selected")
	_, err := runBranchFetch(t, commands, plan)
	assert.Error(t, err, "a deleted server branch must fail without applying a local update")
	assert.Equal(t, h.selected, gitTest(t, h.root, "rev-parse", "refs/heads/selected"))

	gitTest(t, h.root, "push", "-q", h.bare, "refs/heads/selected:refs/heads/selected")
	commands, plan = branchFetchPlan(t, h, h.options())
	writeTestFile(t, filepath.Join(h.root, "rewritten"), []byte("rewrite\n"), 0o644)
	gitTest(t, h.root, "add", "rewritten")
	gitTest(t, h.root, "commit", "-qm", "rewritten target")
	rewritten := gitTest(t, h.root, "rev-parse", "HEAD")
	gitTest(t, h.root, "push", "-q", "--force", h.bare, "refs/heads/master:refs/heads/selected")
	output, err := runBranchFetch(t, commands, plan)
	if err != nil {
		t.Fatalf("fetch should succeed before divergence is assessed: %v\n%s", err, output)
	}
	assert.Equal(t, rewritten, gitTest(t, h.root, "rev-parse", "refs/remotes/custom/selected"))
	_, err = commands.PrepareFetchedBranchUpdate(plan)
	assert.ErrorIs(t, err, ErrBranchSyncDiverged)
	assert.Equal(t, h.selected, gitTest(t, h.root, "rev-parse", "refs/heads/selected"))

	gitTest(t, h.root, "branch", "reserved")
	gitTest(t, h.root, "push", "-q", h.bare, "refs/heads/reserved:refs/heads/reserved")
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, h.root, "worktree", "add", "-q", linked, "reserved")
	linkedGitDir := branchFetchGit(t, linked, "rev-parse", "--absolute-git-dir")
	linkedHeadBefore, err := os.ReadFile(filepath.Join(linkedGitDir, "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	linkedIndexBefore, err := os.ReadFile(filepath.Join(linkedGitDir, "index"))
	if err != nil {
		t.Fatal(err)
	}
	linkedTrackedBefore, err := os.ReadFile(filepath.Join(linked, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(h.root, "reserved-advance"), []byte("advance\n"), 0o644)
	gitTest(t, h.root, "add", "reserved-advance")
	gitTest(t, h.root, "commit", "-qm", "advance reserved")
	gitTest(t, h.root, "push", "-q", "--force", h.bare, "refs/heads/master:refs/heads/selected")
	reservedOID := gitTest(t, h.root, "rev-parse", "refs/heads/reserved")
	reservedPlan, err := commands.PrepareBranchFetch("reserved", reservedOID, h.options())
	if err != nil {
		t.Fatal(err)
	}
	before := gitTest(t, h.root, "rev-parse", "refs/heads/reserved")
	output, err = runBranchFetch(t, commands, reservedPlan)
	if err != nil {
		t.Fatalf("fetch to reserved branch tracking ref should be allowed: %v\n%s", err, output)
	}
	_, err = commands.PrepareFetchedBranchUpdate(reservedPlan)
	assert.Error(t, err, "a real update to an occupied branch must be blocked")
	assert.Equal(t, before, gitTest(t, h.root, "rev-parse", "refs/heads/reserved"))
	assert.Equal(t, linkedHeadBefore, mustRead(t, filepath.Join(linkedGitDir, "HEAD")))
	assert.Equal(t, linkedIndexBefore, mustRead(t, filepath.Join(linkedGitDir, "index")))
	assert.Equal(t, linkedTrackedBefore, mustRead(t, filepath.Join(linked, "tracked")))
}

func TestBranchFetchAheadAndNoopNeverRewind(t *testing.T) {
	h := branchFetchTestHistory(t)
	commands, plan := branchFetchPlan(t, h, h.options())
	gitTest(t, h.root, "update-ref", "refs/remotes/custom/selected", h.selected)
	update, err := commands.PrepareFetchedBranchUpdate(plan)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, update.NeedsUpdate())
	gitTest(t, h.root, "update-ref", "refs/remotes/custom/selected", h.base)
	update, err = commands.PrepareFetchedBranchUpdate(plan)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, update.NeedsUpdate())
	if err := commands.ApplyBranchFastForward(update); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, h.selected, gitTest(t, h.root, "rev-parse", "refs/heads/selected"))
}
