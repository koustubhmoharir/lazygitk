package git_commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/app/daemon"
	"github.com/stretchr/testify/assert"
)

type branchActionRebaseSnapshot struct {
	index, gitHead []byte
	tracked        map[string][]byte
	head, refs     string
}

func snapshotBranchActionRebase(t *testing.T, root string) branchActionRebaseSnapshot {
	t.Helper()
	index, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	selectedRebaseRequireNoError(t, err)
	gitHead, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	selectedRebaseRequireNoError(t, err)
	trackedPaths := strings.Split(gitTest(t, root, "ls-files", "-z"), "\x00")
	tracked := make(map[string][]byte, len(trackedPaths))
	for _, path := range trackedPaths {
		if path == "" {
			continue
		}
		contents, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil && !os.IsNotExist(readErr) {
			t.Fatal(readErr)
		}
		tracked[path] = contents
	}
	return branchActionRebaseSnapshot{index: index, gitHead: gitHead, tracked: tracked, head: gitTest(t, root, "rev-parse", "HEAD"), refs: gitTest(t, root, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads", "refs/tags")}
}

func assertBranchActionRebaseSnapshot(t *testing.T, root string, before branchActionRebaseSnapshot) {
	t.Helper()
	after := snapshotBranchActionRebase(t, root)
	assert.Equal(t, before, after)
}

func branchActionRebaseTarget(t *testing.T, commands *RebaseCommands, ref string) BranchActionTarget {
	t.Helper()
	branches := NewBranchCommands(commands.GitCommon)
	oid := gitTest(t, commands.repoPaths.WorktreePath(), "rev-parse", ref)
	target, err := branches.PrepareBranchActionTarget(ref, oid)
	selectedRebaseRequireNoError(t, err)
	return target
}

func TestBranchActionRebasePrepareAndBuildCommand(t *testing.T) {
	root := initHistoricalRepo(t)
	base := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	main := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	head := selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	gitTest(t, root, "checkout", "-q", main)
	targetOID := selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	gitTest(t, root, "checkout", "-q", "topic")
	gitTest(t, root, "branch", "unrelated", head)
	gitTest(t, root, "config", "rebase.updateRefs", "true")
	commands := selectedCommitRebaseCommands(t, root)
	target := branchActionRebaseTarget(t, commands, "refs/heads/"+main)
	before := snapshotBranchActionRebase(t, root)
	plan, err := commands.PrepareBranchActionRebase(target, "", false)
	selectedRebaseRequireNoError(t, err)
	foreign := selectedCommitRebaseCommands(t, root)
	foreignBefore := snapshotBranchActionRebase(t, root)
	_, err = foreign.BranchActionRebaseCommand(plan)
	assert.Error(t, err)
	assertBranchActionRebaseSnapshot(t, root, foreignBefore)
	cmd, err := commands.BranchActionRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	args := cmd.GetCmd().Args
	assert.Contains(t, strings.Join(args, " "), targetOID+"^{commit}")
	assert.Contains(t, strings.Join(args, " "), "--no-autostash")
	assert.Contains(t, strings.Join(args, " "), "--rebase-merges")
	assert.Contains(t, strings.Join(args, " "), "rebase.updateRefs=false")
	assertBranchActionRebaseSnapshot(t, root, before)
	cmd.RemoveEnvVar("GIT_SEQUENCE_EDITOR").AddEnvVars("GIT_SEQUENCE_EDITOR=true")
	selectedRebaseRequireNoError(t, cmd.Run())
	assert.Equal(t, targetOID, gitTest(t, root, "merge-base", "HEAD", targetOID))
	assert.NotEqual(t, head, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, targetOID, gitTest(t, root, "rev-parse", "refs/heads/"+main))
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "refs/heads/unrelated"))
	assert.Equal(t, base, gitTest(t, root, "merge-base", head, base))
}

func TestBranchActionRebaseMarkedBaseAndInitialBreak(t *testing.T) {
	root := initHistoricalRepo(t)
	base := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	main := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "old-path", "old\n", "add old path")
	gitTest(t, root, "rm", "--", "old-path")
	gitTest(t, root, "commit", "-qm", "remove old path")
	targetOID := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected-target")
	selectedCommitRebaseCommit(t, root, "local", "local\n", "local commit")
	gitTest(t, root, "checkout", "-q", main)
	selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	gitTest(t, root, "checkout", "-q", "topic")
	commands := selectedCommitRebaseCommands(t, root)
	target := branchActionRebaseTarget(t, commands, "refs/heads/selected-target")
	plan, err := commands.PrepareBranchActionRebase(target, base, true)
	selectedRebaseRequireNoError(t, err)
	cmd, err := commands.BranchActionRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	joined := strings.Join(cmd.GetCmd().Args, " ")
	assert.Contains(t, joined, "--onto "+targetOID+"^{commit}")
	assert.Contains(t, joined, base+"^{commit}")
	for _, env := range daemon.ToEnvVars(daemon.NewInsertBreakInstruction()) {
		assert.Contains(t, cmd.GetEnvVars(), env)
	}
	cmd.RemoveEnvVar("GIT_SEQUENCE_EDITOR").AddEnvVars("GIT_SEQUENCE_EDITOR=true")
	selectedRebaseRequireNoError(t, cmd.Run())
	assert.Equal(t, targetOID, gitTest(t, root, "merge-base", "HEAD", targetOID))
	assert.Equal(t, targetOID, gitTest(t, root, "rev-parse", "refs/heads/selected-target"))
}

func TestBranchActionRebasePlanRejectsStaleAndInvalidTargets(t *testing.T) {
	root := initHistoricalRepo(t)
	targetOID := selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	commands := selectedCommitRebaseCommands(t, root)
	branches := NewBranchCommands(commands.GitCommon)
	ref := gitTest(t, root, "symbolic-ref", "HEAD")
	currentBranch := strings.TrimPrefix(ref, "refs/heads/")
	target, err := branches.PrepareBranchActionTarget(ref, targetOID)
	selectedRebaseRequireNoError(t, err)
	plan, err := commands.PrepareBranchActionRebase(target, "", false)
	selectedRebaseRequireNoError(t, err)
	unchanged := snapshotBranchActionRebase(t, root)
	_, err = commands.BranchActionRebaseCommand(BranchActionRebasePlan{})
	assert.Error(t, err)
	_, err = commands.PrepareBranchActionRebase(BranchActionTarget{}, "", false)
	assert.Error(t, err)
	_, err = commands.PrepareBranchActionRebase(target, "not-an-oid", false)
	assert.Error(t, err)
	assertBranchActionRebaseSnapshot(t, root, unchanged)
	gitTest(t, root, "branch", "alias")
	gitTest(t, root, "checkout", "-q", "alias")
	aliasSnapshot := snapshotBranchActionRebase(t, root)
	_, err = commands.BranchActionRebaseCommand(plan)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assertBranchActionRebaseSnapshot(t, root, aliasSnapshot)
	gitTest(t, root, "checkout", "-q", currentBranch)
	selectedCommitRebaseCommit(t, root, "head-change", "head change\n", "head change")
	headChangedSnapshot := snapshotBranchActionRebase(t, root)
	_, err = commands.BranchActionRebaseCommand(plan)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assertBranchActionRebaseSnapshot(t, root, headChangedSnapshot)
}

func TestBranchActionRebaseRejectsChangedSourceTip(t *testing.T) {
	root := initHistoricalRepo(t)
	base := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	selectedCommitRebaseCommit(t, root, "next", "next\n", "next")
	current := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "branch", "selected-source", base)
	commands := selectedCommitRebaseCommands(t, root)
	target := branchActionRebaseTarget(t, commands, "refs/heads/selected-source")
	plan, err := commands.PrepareBranchActionRebase(target, "", false)
	selectedRebaseRequireNoError(t, err)
	gitTest(t, root, "update-ref", "refs/heads/selected-source", gitTest(t, root, "rev-parse", current))
	sourceChangedSnapshot := snapshotBranchActionRebase(t, root)
	_, err = commands.BranchActionRebaseCommand(plan)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assertBranchActionRebaseSnapshot(t, root, sourceChangedSnapshot)
}

func TestBranchActionRebaseDirtyAndUntrackedPolicy(t *testing.T) {
	root := initHistoricalRepo(t)
	targetOID := selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	commands := selectedCommitRebaseCommands(t, root)
	target := branchActionRebaseTarget(t, commands, "refs/heads/"+gitTest(t, root, "branch", "--show-current"))
	writeTestFile(t, filepath.Join(root, "target"), []byte("dirty\n"), 0o644)
	before := snapshotBranchActionRebase(t, root)
	_, err := commands.PrepareBranchActionRebase(target, "", false)
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	assertBranchActionRebaseSnapshot(t, root, before)
	gitTest(t, root, "checkout", "--", "target")
	writeTestFile(t, filepath.Join(root, "target"), []byte("staged\n"), 0o644)
	gitTest(t, root, "add", "--", "target")
	staged := snapshotBranchActionRebase(t, root)
	_, err = commands.PrepareBranchActionRebase(target, "", false)
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	assertBranchActionRebaseSnapshot(t, root, staged)
	gitTest(t, root, "reset", "-q", "HEAD", "--", "target")
	gitTest(t, root, "checkout", "--", "target")
	lockPath := filepath.Join(root, ".git", "index.lock")
	lockBytes := []byte("caller-owned lock bytes\x00\xff")
	selectedRebaseRequireNoError(t, os.WriteFile(lockPath, lockBytes, 0o600))
	lockSnapshot := snapshotBranchActionRebase(t, root)
	_, err = commands.PrepareBranchActionRebase(target, "", false)
	assert.NoError(t, err, "read-only plan preparation does not acquire index.lock")
	gotLock, readErr := os.ReadFile(lockPath)
	selectedRebaseRequireNoError(t, readErr)
	assert.Equal(t, lockBytes, gotLock)
	assertBranchActionRebaseSnapshot(t, root, lockSnapshot)
	selectedRebaseRequireNoError(t, os.Remove(lockPath))
	writeTestFile(t, filepath.Join(root, "harmless-untracked"), []byte("keep\n"), 0o644)
	plan, err := commands.PrepareBranchActionRebase(target, "", false)
	selectedRebaseRequireNoError(t, err)
	cmd, err := commands.BranchActionRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	cmd.RemoveEnvVar("GIT_SEQUENCE_EDITOR").AddEnvVars("GIT_SEQUENCE_EDITOR=true")
	selectedRebaseRequireNoError(t, cmd.Run())
	contents, err := os.ReadFile(filepath.Join(root, "harmless-untracked"))
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, "keep\n", string(contents))
	assert.Equal(t, targetOID, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestBranchActionRebaseNativeIndexLockFailureRunsOnceAndPreservesLock(t *testing.T) {
	root := initHistoricalRepo(t)
	base := selectedCommitRebaseCommit(t, root, "shared", "base\n", "base")
	main := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	topicHead := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", main)
	selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	gitTest(t, root, "checkout", "-q", "topic")
	commands := selectedCommitRebaseCommands(t, root)
	target := branchActionRebaseTarget(t, commands, "refs/heads/"+main)
	plan, err := commands.PrepareBranchActionRebase(target, "", false)
	selectedRebaseRequireNoError(t, err)
	cmd, err := commands.BranchActionRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	cmd.RemoveEnvVar("GIT_SEQUENCE_EDITOR").AddEnvVars("GIT_SEQUENCE_EDITOR=true")
	lockPath := filepath.Join(root, ".git", "index.lock")
	lockBytes := []byte("caller-owned native execution lock\x00\xff")
	selectedRebaseRequireNoError(t, os.WriteFile(lockPath, lockBytes, 0o600))
	before := snapshotBranchActionRebase(t, root)
	runErr := cmd.Run()
	if assert.Error(t, runErr) {
		assert.Contains(t, strings.ToLower(runErr.Error()), "index.lock")
	}
	gotLock, readErr := os.ReadFile(lockPath)
	selectedRebaseRequireNoError(t, readErr)
	assert.Equal(t, lockBytes, gotLock, "Git must not remove or replace the caller's lock")
	// This divergent, non-conflicting replay fails before applying its tree; the
	// native failure must leave repository state available for an explicit retry.
	assertBranchActionRebaseSnapshot(t, root, before)
	assert.Equal(t, topicHead, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, base, gitTest(t, root, "merge-base", topicHead, base))
}

func TestBranchActionRebaseInProgressGuardTracksConflictIdentity(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "shared", "base\n", "base")
	main := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "shared", "topic edit\n", "topic edit")
	topicHead := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", main)
	selectedCommitRebaseCommit(t, root, "shared", "first target edit\n", "first target")
	gitTest(t, root, "checkout", "-q", "topic")
	commands := selectedCommitRebaseCommands(t, root)
	firstTarget := branchActionRebaseTarget(t, commands, "refs/heads/"+main)
	firstPlan, err := commands.PrepareBranchActionRebase(firstTarget, "", false)
	selectedRebaseRequireNoError(t, err)
	firstCmd, err := commands.BranchActionRebaseCommand(firstPlan)
	selectedRebaseRequireNoError(t, err)
	firstCmd.RemoveEnvVar("GIT_SEQUENCE_EDITOR").AddEnvVars("GIT_SEQUENCE_EDITOR=true")
	if err = firstCmd.Run(); err == nil {
		t.Fatal("expected real conflict while rebasing the topic edit onto the first target")
	}
	assert.Error(t, commands.CheckBranchActionRebaseInProgress(BranchActionRebasePlan{}))
	foreign := selectedCommitRebaseCommands(t, root)
	assert.Error(t, foreign.CheckBranchActionRebaseInProgress(firstPlan))
	selectedRebaseRequireNoError(t, commands.CheckBranchActionRebaseInProgress(firstPlan))
	selectedRebaseRequireNoError(t, commands.AbortRebase())
	assert.ErrorIs(t, commands.CheckBranchActionRebaseInProgress(firstPlan), ErrBranchActionChanged)
	assert.Equal(t, topicHead, gitTest(t, root, "rev-parse", "HEAD"))

	gitTest(t, root, "checkout", "-q", main)
	secondTargetOID := selectedCommitRebaseCommit(t, root, "shared", "second target edit\n", "second target")
	secondTargetRef := "refs/heads/second-target"
	gitTest(t, root, "branch", "second-target", secondTargetOID)
	gitTest(t, root, "checkout", "-q", "topic")
	secondTarget := branchActionRebaseTarget(t, commands, secondTargetRef)
	secondPlan, err := commands.PrepareBranchActionRebase(secondTarget, "", false)
	selectedRebaseRequireNoError(t, err)
	secondCmd, err := commands.BranchActionRebaseCommand(secondPlan)
	selectedRebaseRequireNoError(t, err)
	secondCmd.RemoveEnvVar("GIT_SEQUENCE_EDITOR").AddEnvVars("GIT_SEQUENCE_EDITOR=true")
	if err = secondCmd.Run(); err == nil {
		t.Fatal("expected real conflict while rebasing onto the second target")
	}
	assert.ErrorIs(t, commands.CheckBranchActionRebaseInProgress(firstPlan), ErrBranchActionChanged)
	selectedRebaseRequireNoError(t, commands.CheckBranchActionRebaseInProgress(secondPlan))
	selectedRebaseRequireNoError(t, commands.AbortRebase())
	assert.Equal(t, secondTargetOID, gitTest(t, root, "rev-parse", "refs/heads/"+main))
}

func TestBranchActionRebaseIgnoredRiskAndChangedApproval(t *testing.T) {
	root := initHistoricalRepo(t)
	baseOID := selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "old-path", "old\n", "add old path")
	gitTest(t, root, "rm", "--", "old-path")
	gitTest(t, root, "commit", "-qm", "remove old path")
	targetOID := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected-target")
	selectedCommitRebaseCommit(t, root, "local", "local\n", "local commit")
	head := gitTest(t, root, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("old-path\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "old-path"), []byte("precious\n"), 0o644)
	commands := selectedCommitRebaseCommands(t, root)
	target := branchActionRebaseTarget(t, commands, "refs/heads/selected-target")
	ordinary, err := commands.PrepareBranchActionRebase(target, "", false)
	selectedRebaseRequireNoError(t, err)
	assert.Empty(t, ordinary.IgnoredFiles())
	marked, err := commands.PrepareBranchActionRebase(target, baseOID, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{"old-path"}, marked.IgnoredFiles())
	_, err = commands.BranchActionRebaseCommand(marked)
	assert.ErrorIs(t, err, ErrRebaseIgnoredApprovalRequired)
	precious, readErr := os.ReadFile(filepath.Join(root, "old-path"))
	selectedRebaseRequireNoError(t, readErr)
	assert.Equal(t, "precious\n", string(precious))
	approved := marked.WithIgnoredFilesApproved()
	writeTestFile(t, filepath.Join(root, "old-path"), []byte("changed\n"), 0o644)
	_, err = commands.BranchActionRebaseCommand(approved)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	changed, readErr := os.ReadFile(filepath.Join(root, "old-path"))
	selectedRebaseRequireNoError(t, readErr)
	assert.Equal(t, "changed\n", string(changed))
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
	assert.NotEqual(t, targetOID, head)
}
