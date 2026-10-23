package git_commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func prepareMergePlan(t *testing.T, commands *BranchCommands, mode BranchActionMergeMode) BranchActionMergePlan {
	t.Helper()
	target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, commands.repoPaths.WorktreePath(), "rev-parse", "refs/heads/source"))
	if !assert.NoError(t, err) {
		return BranchActionMergePlan{}
	}
	review, err := commands.PrepareBranchActionMergeReview(target, "")
	if !assert.NoError(t, err) {
		return BranchActionMergePlan{}
	}
	plan, err := commands.PrepareBranchActionMerge(review, mode, "")
	assert.NoError(t, err)
	return plan
}

func runMergePlan(t *testing.T, commands *BranchCommands, plan BranchActionMergePlan) error {
	t.Helper()
	cmd, err := commands.BranchActionMergeCommand(plan)
	if err != nil {
		return err
	}
	return cmd.Run()
}

func TestBranchActionMergePlanFastForwardAndExplicitAllowFF(t *testing.T) {
	for _, mode := range []BranchActionMergeMode{BranchActionMergeConfigured, BranchActionMergeAllowFF} {
		t.Run(map[BranchActionMergeMode]string{BranchActionMergeConfigured: "configured", BranchActionMergeAllowFF: "explicit allow ff"}[mode], func(t *testing.T) {
			root, _, head := transactionHistory(t)
			gitTest(t, root, "branch", "source", head)
			mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
			gitTest(t, root, "checkout", "-q", "master")
			commands := transactionCommands(t, root)
			plan := prepareMergePlan(t, commands, mode)
			before := mergePreviewSnapshot(t, root)
			cmd, err := commands.BranchActionMergeCommand(plan)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, before, mergePreviewSnapshot(t, root), "building the command is read-only")
			assert.NoError(t, cmd.Run())
			assert.Equal(t, gitTest(t, root, "rev-parse", "refs/heads/source"), gitTest(t, root, "rev-parse", "HEAD"))
			assert.FileExists(t, filepath.Join(root, "incoming"))
		})
	}
}

func TestBranchActionMergePlanExplicitAllowFFDivergenceCreatesMerge(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	mergeFixtureCommit(t, root, "master", "local", "local\n", "local")
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeAllowFF)
	assert.NoError(t, runMergePlan(t, commands, plan))
	assert.Equal(t, 3, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
	assert.FileExists(t, filepath.Join(root, "incoming"))
}

func TestBranchActionMergePlanNoFFForcesMergeCommitOnFastForward(t *testing.T) {
	root, _, head := transactionHistory(t)
	gitTest(t, root, "branch", "source", head)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeNoFF)
	assert.NoError(t, runMergePlan(t, commands, plan))
	assert.Equal(t, 3, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
}

func TestBranchActionMergePlanSquashLeavesRefsAndCommitsUntouched(t *testing.T) {
	for _, mode := range []BranchActionMergeMode{BranchActionMergeSquash, BranchActionMergeSquashAndCommit} {
		t.Run(map[BranchActionMergeMode]string{BranchActionMergeSquash: "squash", BranchActionMergeSquashAndCommit: "squash and commit plan phase one"}[mode], func(t *testing.T) {
			root, base, _ := transactionHistory(t)
			gitTest(t, root, "branch", "source", base)
			mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
			gitTest(t, root, "checkout", "-q", "master")
			gitTest(t, root, "branch", "unrelated", "HEAD")
			commands := transactionCommands(t, root)
			beforeHEAD := gitTest(t, root, "rev-parse", "HEAD")
			beforeSource := gitTest(t, root, "rev-parse", "refs/heads/source")
			beforeUnrelated := gitTest(t, root, "rev-parse", "refs/heads/unrelated")
			plan := prepareMergePlan(t, commands, mode)
			assert.NoError(t, runMergePlan(t, commands, plan))
			assert.Equal(t, beforeHEAD, gitTest(t, root, "rev-parse", "HEAD"))
			assert.Equal(t, beforeSource, gitTest(t, root, "rev-parse", "refs/heads/source"))
			assert.Equal(t, beforeUnrelated, gitTest(t, root, "rev-parse", "refs/heads/unrelated"))
			assert.Equal(t, "A  incoming", gitTest(t, root, "status", "--short", "--", "incoming"))
			assert.Equal(t, "incoming\n", string(mustRead(t, filepath.Join(root, "incoming"))))
		})
	}
}

func TestBranchActionMergePlanSquashAndCommitRequiresApprovalForStagedFiles(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	writeTestFile(t, filepath.Join(root, "already-staged"), []byte("precious\n"), 0o644)
	gitTest(t, root, "add", "already-staged")
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeSquashAndCommit)
	_, err := commands.BranchActionMergeCommand(plan)
	assert.ErrorIs(t, err, ErrMergeStagedApprovalRequired)
	_, err = commands.BranchActionMergeCommand(plan.WithStagedFilesApproved())
	assert.NoError(t, err, "explicit approval must pass the plan's staged-file gate")
	assert.Equal(t, "A  already-staged", gitTest(t, root, "status", "--short", "--", "already-staged"))
	assert.Equal(t, "precious\n", string(mustRead(t, filepath.Join(root, "already-staged"))))
}

func TestBranchActionMergePlanNoOpDoesNotCreateCommit(t *testing.T) {
	root, _, head := transactionHistory(t)
	gitTest(t, root, "branch", "source", head)
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeConfigured)
	before := gitTest(t, root, "rev-parse", "HEAD")
	assert.NoError(t, runMergePlan(t, commands, plan))
	assert.Equal(t, before, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestBranchActionMergePlanIgnoredCollisionRequiresBoundApproval(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "ignored", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore-rules"))
	writeTestFile(t, filepath.Join(root, "ignore-rules"), []byte("/ignored\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "ignored"), []byte("precious\n"), 0o644)
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeConfigured)
	assert.Equal(t, []string{"ignored"}, plan.IgnoredFiles())
	_, err := commands.BranchActionMergeCommand(plan)
	assert.ErrorIs(t, err, ErrMergeIgnoredApprovalRequired)
	assert.NoError(t, runMergePlan(t, commands, plan.WithIgnoredFilesApproved()))
	assert.Equal(t, "incoming\n", string(mustRead(t, filepath.Join(root, "ignored"))))
}

func TestBranchActionMergePlanIgnoredByteChangeInvalidatesApproval(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "ignored", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore-rules"))
	writeTestFile(t, filepath.Join(root, "ignore-rules"), []byte("/ignored\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "ignored"), []byte("old\n"), 0o644)
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeConfigured).WithIgnoredFilesApproved()
	writeTestFile(t, filepath.Join(root, "ignored"), []byte("new bytes\n"), 0o644)
	_, err := commands.BranchActionMergeCommand(plan)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assert.Equal(t, "new bytes\n", string(mustRead(t, filepath.Join(root, "ignored"))))
}

func TestBranchActionMergePlanRevalidationDoesNotRerunMergeDriver(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "shared", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	mergeFixtureCommit(t, root, "master", "shared", "local\n", "local")
	driver := filepath.Join(root, "driver.sh")
	count := filepath.Join(root, "driver-count")
	writeTestFile(t, count, nil, 0o644)
	writeTestFile(t, driver, []byte("#!/bin/sh\nprintf x >> '"+count+"'\nprintf '%s\\n' merged > \"$2\"\n"), 0o755)
	writeTestFile(t, filepath.Join(root, ".gitattributes"), []byte("shared merge=counting\n"), 0o644)
	gitTest(t, root, "add", ".gitattributes")
	gitTest(t, root, "commit", "-qm", "configure merge driver")
	gitTest(t, root, "config", "merge.counting.driver", driver+" %O %A %B")
	gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore-rules"))
	writeTestFile(t, filepath.Join(root, "ignore-rules"), []byte("/ignored-helper\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "ignored-helper"), []byte("harmless\n"), 0o644)
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeConfigured)
	countAfterPrepare, err := os.ReadFile(count)
	assert.NoError(t, err)
	cmd, err := commands.BranchActionMergeCommand(plan)
	assert.NoError(t, err)
	countAfterBuild, err := os.ReadFile(count)
	assert.NoError(t, err)
	assert.Equal(t, countAfterPrepare, countAfterBuild, "final command construction must use the cached preview")
	cmd.RemoveEnvVar("GIT_EDITOR").AddEnvVars("GIT_EDITOR=true")
	assert.NoError(t, cmd.Run())
}

func TestBranchActionMergePlanCustomStrategyConservativelyReviewsIgnoredFiles(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "not-ours", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore-rules"))
	writeTestFile(t, filepath.Join(root, "ignore-rules"), []byte("/ignored\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "ignored"), []byte("precious\n"), 0o644)
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
	assert.NoError(t, err)
	review, err := commands.PrepareBranchActionMergeReview(target, "--strategy=ours")
	assert.NoError(t, err)
	plan, err := commands.PrepareBranchActionMerge(review, BranchActionMergeConfigured, "")
	assert.NoError(t, err)
	assert.True(t, plan.ConservativeIgnoredReview())
	assert.Equal(t, []string{"ignored"}, plan.IgnoredFiles())
	cmd, err := commands.BranchActionMergeCommand(plan.WithIgnoredFilesApproved())
	assert.NoError(t, err)
	assert.Contains(t, cmd.GetCmd().Args, "--strategy=ours", "approval must preserve the configured strategy")
	assert.NoError(t, cmd.Run())
	assert.NoFileExists(t, filepath.Join(root, "not-ours"), "the preserved ours strategy controls the actual merge")
	assert.Equal(t, "precious\n", string(mustRead(t, filepath.Join(root, "ignored"))))
}

func TestBranchActionMergePlanAllowsOccupiedSourceButRefusesOccupiedDestination(t *testing.T) {
	root, _, head := transactionHistory(t)
	gitTest(t, root, "branch", "source", head)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	commands := transactionCommands(t, root)
	other := filepath.Join(t.TempDir(), "other-worktree")
	gitTest(t, root, "worktree", "add", "-q", other, "source")
	plan := prepareMergePlan(t, commands, BranchActionMergeConfigured)
	assert.NoError(t, runMergePlan(t, commands, plan), "the source branch is read-only for this operation")
	assert.Equal(t, gitTest(t, other, "rev-parse", "HEAD"), gitTest(t, root, "rev-parse", "refs/heads/source"))
	gitTest(t, root, "worktree", "remove", "-f", other)
	gitTest(t, root, "checkout", "-q", "source")
	mergeFixtureCommit(t, root, "source", "later", "later\n", "later source commit")
	gitTest(t, root, "checkout", "-q", "master")
	gitTest(t, root, "checkout", "-q", "-b", "current")
	other = filepath.Join(t.TempDir(), "other-worktree")
	gitTest(t, root, "worktree", "add", "-q", "--force", other, "current")
	target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
	assert.NoError(t, err)
	review, err := commands.PrepareBranchActionMergeReview(target, "")
	assert.NoError(t, err)
	_, err = commands.PrepareBranchActionMerge(review, BranchActionMergeNoFF, "")
	assert.Error(t, err, "moving a destination branch checked out in another worktree must be refused")
}

func TestBranchActionMergePlanRejectsInvalidModesAndUnbornRestrictions(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
	assert.NoError(t, err)
	review, err := commands.PrepareBranchActionMergeReview(target, "")
	assert.NoError(t, err)
	_, err = commands.PrepareBranchActionMerge(review, BranchActionMergeMode(255), "")
	assert.Error(t, err)

	unbornRoot, _, unbornHead := transactionHistory(t)
	gitTest(t, unbornRoot, "branch", "source", unbornHead)
	gitTest(t, unbornRoot, "checkout", "--orphan", "unborn")
	gitTest(t, unbornRoot, "rm", "-rf", ".")
	unbornCommands := transactionCommands(t, unbornRoot)
	target, err = unbornCommands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, unbornRoot, "rev-parse", "refs/heads/source"))
	assert.NoError(t, err)
	review, err = unbornCommands.PrepareBranchActionMergeReview(target, "")
	assert.NoError(t, err)
	for _, mode := range []BranchActionMergeMode{BranchActionMergeNoFF, BranchActionMergeSquash, BranchActionMergeSquashAndCommit} {
		_, err = unbornCommands.PrepareBranchActionMerge(review, mode, "")
		assert.Error(t, err, "mode %d cannot merge into an unborn branch", mode)
	}
}

func TestBranchActionMergePlanRejectsFastForwardOnlyDivergenceAndInvalidSquashOptions(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	mergeFixtureCommit(t, root, "master", "local", "local\n", "local")
	gitTest(t, root, "config", "merge.ff", "only")
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
	assert.NoError(t, err)
	review, err := commands.PrepareBranchActionMergeReview(target, "")
	assert.NoError(t, err)
	_, err = commands.PrepareBranchActionMerge(review, BranchActionMergeConfigured, "")
	assert.Error(t, err)
	gitTest(t, root, "config", "merge.ff", "true")
	for _, args := range []string{"--commit --squash", "--no-ff --squash"} {
		review, err = commands.PrepareBranchActionMergeReview(target, args)
		assert.NoError(t, err)
		_, err = commands.PrepareBranchActionMerge(review, BranchActionMergeConfigured, "")
		assert.Error(t, err, "invalid native squash combination %q must be rejected", args)
	}
}

func TestBranchActionMergePlanCarriesNonoverlappingDirtyChangesAndRejectsOverlap(t *testing.T) {
	t.Run("nonoverlapping dirty file is carried", func(t *testing.T) {
		root, base, _ := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
		gitTest(t, root, "checkout", "-q", "master")
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("local dirty\n"), 0o644)
		commands := transactionCommands(t, root)
		plan := prepareMergePlan(t, commands, BranchActionMergeConfigured)
		assert.NoError(t, runMergePlan(t, commands, plan))
		assert.Equal(t, "local dirty\n", string(mustRead(t, filepath.Join(root, "tracked"))))
		assert.FileExists(t, filepath.Join(root, "incoming"))
	})
	t.Run("overlap rejects and preserves dirty bytes", func(t *testing.T) {
		root, base, _ := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		mergeFixtureCommit(t, root, "source", "tracked", "incoming\n", "incoming")
		gitTest(t, root, "checkout", "-q", "master")
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("precious dirty\n"), 0o644)
		commands := transactionCommands(t, root)
		plan := prepareMergePlan(t, commands, BranchActionMergeConfigured)
		assert.Error(t, runMergePlan(t, commands, plan))
		assert.Equal(t, "precious dirty\n", string(mustRead(t, filepath.Join(root, "tracked"))))
	})
}
