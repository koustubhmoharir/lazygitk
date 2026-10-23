package git_commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func localDeleteNoError(t *testing.T, err error) {
	t.Helper()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}

func localDeleteTarget(t *testing.T, commands *BranchCommands, ref, oid string) BranchActionTarget {
	t.Helper()
	target, err := commands.PrepareBranchActionTarget(ref, oid)
	localDeleteNoError(t, err)
	return target
}

func TestBranchActionLocalDeleteMergedAndLiteralNames(t *testing.T) {
	root, first, current := transactionHistory(t)
	name := first
	gitTest(t, root, "branch", name, first)
	gitTest(t, root, "branch", "other", first)
	gitTest(t, root, "update-ref", "refs/vendor/keep", first)
	gitTest(t, root, "config", "user.name", "preserve-this-config")
	commands := transactionCommands(t, root)
	target := localDeleteTarget(t, commands, "refs/heads/"+name, first)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	localDeleteNoError(t, err)
	assert.Len(t, plan.Sources(), 1)
	assert.True(t, plan.Sources()[0].Merged)
	cmd, err := commands.BranchActionLocalDeleteCommand(plan)
	localDeleteNoError(t, err)
	localDeleteNoError(t, cmd.Run())
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, "preserve-this-config", gitTest(t, root, "config", "user.name"))
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/other"))
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/vendor/keep"))
	_, err = commands.ResolveBranchActionCommit("refs/heads/" + name)
	assert.Error(t, err)
}

func TestBranchActionLocalDeleteBatchExclusionsRequireForce(t *testing.T) {
	root, _, current := transactionHistory(t)
	gitTest(t, root, "checkout", "-qb", "side")
	divergent := branchPlacementTestCommit(t, root, "side-file", "side\n", "side")
	gitTest(t, root, "branch", "one", divergent)
	gitTest(t, root, "branch", "two", divergent)
	gitTest(t, root, "checkout", "-q", "master")
	commands := transactionCommands(t, root)
	one := localDeleteTarget(t, commands, "refs/heads/one", divergent)
	two := localDeleteTarget(t, commands, "refs/heads/two", divergent)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{one, two}, []string{"refs/heads/one", "refs/heads/two"}, []string{"refs/heads/one", "refs/heads/two"})
	localDeleteNoError(t, err)
	for _, source := range plan.Sources() {
		assert.False(t, source.Merged)
	}
	_, err = commands.BranchActionLocalDeleteCommand(plan)
	assert.ErrorIs(t, err, ErrBranchDeleteForceApprovalRequired)
	cmd, err := commands.BranchActionLocalDeleteCommand(plan.WithForceApproved())
	localDeleteNoError(t, err)
	localDeleteNoError(t, cmd.Run())
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
	for _, ref := range []string{"refs/heads/one", "refs/heads/two"} {
		_, err := commands.ResolveBranchActionCommit(ref)
		assert.ErrorIs(t, err, ErrBranchActionChanged)
	}
}

func TestBranchActionLocalDeleteDirtyStateAndStaleSnapshots(t *testing.T) {
	root, first, current := transactionHistory(t)
	gitTest(t, root, "branch", "selected", first)
	commands := transactionCommands(t, root)
	target := localDeleteTarget(t, commands, "refs/heads/selected", first)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, []string{"refs/heads/master"}, nil)
	localDeleteNoError(t, err)
	tracked := filepath.Join(root, "tracked")
	localDeleteNoError(t, os.WriteFile(tracked, []byte("staged\n"), 0o644))
	gitTest(t, root, "add", "tracked")
	localDeleteNoError(t, os.WriteFile(tracked, []byte("dirty\x00bytes\n"), 0o644))
	localDeleteNoError(t, os.WriteFile(filepath.Join(root, "untracked"), []byte("keep"), 0o600))
	index := mustRead(t, filepath.Join(root, ".git", "index"))
	cmd, err := commands.BranchActionLocalDeleteCommand(plan)
	localDeleteNoError(t, err)
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, index, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, []byte("dirty\x00bytes\n"), mustRead(t, tracked))
	assert.Equal(t, []byte("keep"), mustRead(t, filepath.Join(root, "untracked")))
	localDeleteNoError(t, cmd.Run())
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, index, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, []byte("dirty\x00bytes\n"), mustRead(t, tracked))
	assert.Equal(t, []byte("keep"), mustRead(t, filepath.Join(root, "untracked")))

	root, first, _ = transactionHistory(t)
	gitTest(t, root, "branch", "selected", first)
	commands = transactionCommands(t, root)
	target = localDeleteTarget(t, commands, "refs/heads/selected", first)
	plan, err = commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	localDeleteNoError(t, err)
	third := branchPlacementTestCommit(t, root, "third", "third\n", "third")
	_, err = commands.BranchActionLocalDeleteCommand(plan)
	assert.True(t, errors.Is(err, ErrBranchActionChanged))
	assert.Equal(t, third, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestBranchActionLocalDeleteRejectsInvalidAndForeignTargets(t *testing.T) {
	root, first, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", first)
	c := transactionCommands(t, root)
	target := localDeleteTarget(t, c, "refs/heads/selected", first)
	for _, targets := range [][]BranchActionTarget{nil, {target, target}, {BranchActionTarget{}}} {
		_, err := c.PrepareBranchActionLocalDelete(targets, nil, nil)
		assert.Error(t, err)
	}
	other := transactionCommands(t, root)
	_, err := other.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	assert.Error(t, err)
	plan, err := c.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	localDeleteNoError(t, err)
	_, err = other.BranchActionLocalDeleteCommand(plan)
	assert.Error(t, err)
	gitTest(t, root, "update-ref", "refs/heads/selected", gitTest(t, root, "rev-parse", "HEAD"), first)
	_, err = c.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	gitTest(t, root, "update-ref", "-d", "refs/heads/selected")
	_, err = c.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	assert.Error(t, err)
}

func TestBranchActionLocalDeleteBatchRequiresSameCapturedHEAD(t *testing.T) {
	root, first, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", first)
	commands := transactionCommands(t, root)
	firstHead := localDeleteTarget(t, commands, "refs/heads/selected", first)
	gitTest(t, root, "checkout", "--detach", "-q", first)
	sameSourceDifferentHead := localDeleteTarget(t, commands, "refs/heads/selected", first)
	_, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{firstHead, sameSourceDifferentHead}, nil, nil)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
}

func TestBranchActionLocalDeleteProofAndConfigFreshness(t *testing.T) {
	t.Run("excluded proof may disappear", func(t *testing.T) {
		root, first, current := transactionHistory(t)
		gitTest(t, root, "branch", "selected", first)
		gitTest(t, root, "update-ref", "refs/remotes/origin/deleted", first)
		commands := transactionCommands(t, root)
		target := localDeleteTarget(t, commands, "refs/heads/selected", first)
		plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, []string{"refs/heads/master"}, []string{"refs/remotes/origin/deleted"})
		localDeleteNoError(t, err)
		gitTest(t, root, "update-ref", "-d", "refs/remotes/origin/deleted")
		cmd, err := commands.BranchActionLocalDeleteCommand(plan)
		localDeleteNoError(t, err)
		localDeleteNoError(t, cmd.Run())
		assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
	})
	t.Run("proof and configuration changes invalidate", func(t *testing.T) {
		root, first, current := transactionHistory(t)
		gitTest(t, root, "branch", "selected", first)
		gitTest(t, root, "update-ref", "refs/recovery/proof", current)
		commands := transactionCommands(t, root)
		target := localDeleteTarget(t, commands, "refs/heads/selected", first)
		plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, []string{"refs/recovery/proof"}, nil)
		localDeleteNoError(t, err)
		gitTest(t, root, "update-ref", "refs/recovery/proof", first, current)
		_, err = commands.BranchActionLocalDeleteCommand(plan)
		assert.ErrorIs(t, err, ErrBranchActionChanged)

		root, first, _ = transactionHistory(t)
		gitTest(t, root, "branch", "selected", first)
		commands = transactionCommands(t, root)
		target = localDeleteTarget(t, commands, "refs/heads/selected", first)
		plan, err = commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
		localDeleteNoError(t, err)
		gitTest(t, root, "config", "user.name", "changed after review")
		_, err = commands.BranchActionLocalDeleteCommand(plan)
		assert.ErrorIs(t, err, ErrBranchActionChanged)
	})
}

func TestBranchActionLocalDeleteCurrentReservationCannotBeForceApproved(t *testing.T) {
	root, _, current := transactionHistory(t)
	commands := transactionCommands(t, root)
	target := localDeleteTarget(t, commands, "refs/heads/master", current)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	localDeleteNoError(t, err)
	assert.Equal(t, "checkout", plan.Sources()[0].Info.Reservation.Kind)
	_, err = commands.BranchActionLocalDeleteCommand(plan.WithForceApproved())
	assert.ErrorContains(t, err, "reserved")
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestBranchActionLocalDeleteUsesOnlyEachSourcesOwnUpstream(t *testing.T) {
	root, first, _ := transactionHistory(t)
	gitTest(t, root, "checkout", "-qb", "side", first)
	aOID := branchPlacementTestCommit(t, root, "a-file", "A\n", "A")
	gitTest(t, root, "branch", "source-a", aOID)
	bOID := branchPlacementTestCommit(t, root, "b-file", "B\n", "B")
	gitTest(t, root, "branch", "source-b", bOID)
	gitTest(t, root, "branch", "upstream-b", bOID)
	gitTest(t, root, "config", "branch.source-b.remote", ".")
	gitTest(t, root, "config", "branch.source-b.merge", "refs/heads/upstream-b")
	gitTest(t, root, "checkout", "-q", "master")
	commands := transactionCommands(t, root)
	a := localDeleteTarget(t, commands, "refs/heads/source-a", aOID)
	b := localDeleteTarget(t, commands, "refs/heads/source-b", bOID)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{a, b}, nil, nil)
	localDeleteNoError(t, err)
	got := plan.Sources()
	assert.False(t, got[0].Merged, "source A must not inherit source B's upstream as deletion proof")
	assert.True(t, got[1].Merged, "source B is reachable from its own upstream")
	_, err = commands.BranchActionLocalDeleteCommand(plan)
	assert.ErrorIs(t, err, ErrBranchDeleteForceApprovalRequired)
	_, err = commands.BranchActionLocalDeleteCommand(plan.WithForceApproved())
	localDeleteNoError(t, err)
}

func TestBranchActionLocalDeleteExcludedCustomUpstreamMayDisappear(t *testing.T) {
	root, first, _ := transactionHistory(t)
	gitTest(t, root, "checkout", "-qb", "side", first)
	sourceOID := branchPlacementTestCommit(t, root, "side-file", "side\n", "side")
	gitTest(t, root, "branch", "selected", sourceOID)
	gitTest(t, root, "checkout", "-q", "master")
	tracking := "refs/custom/cache/review"
	gitTest(t, root, "config", "remote.review.url", filepath.Join(root, "remote.git"))
	gitTest(t, root, "config", "remote.review.fetch", "+refs/heads/*:refs/custom/cache/*")
	gitTest(t, root, "config", "branch.selected.remote", "review")
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/review")
	gitTest(t, root, "update-ref", tracking, sourceOID)
	commands := transactionCommands(t, root)
	target := localDeleteTarget(t, commands, "refs/heads/selected", sourceOID)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, []string{tracking})
	localDeleteNoError(t, err)
	assert.Equal(t, tracking, plan.Sources()[0].Info.UpstreamTrackingRef)
	gitTest(t, root, "update-ref", "-d", tracking)
	cmd, err := commands.BranchActionLocalDeleteCommand(plan.WithForceApproved())
	localDeleteNoError(t, err)
	assert.Contains(t, cmd.GetCmd().Args, "selected")
}

func TestBranchActionLocalDeleteWorktreeAndOperationReservations(t *testing.T) {
	t.Run("other worktree reserves selected branch", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		linked := filepath.Join(t.TempDir(), "linked")
		gitTest(t, root, "worktree", "add", "-qb", "reserved", linked, first)
		commands := transactionCommands(t, root)
		target := localDeleteTarget(t, commands, "refs/heads/reserved", first)
		plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
		localDeleteNoError(t, err)
		_, err = commands.BranchActionLocalDeleteCommand(plan.WithForceApproved())
		assert.ErrorContains(t, err, "reserved")
	})
	t.Run("operation reservation blocks its branch but not another", func(t *testing.T) {
		root, first, current := transactionHistory(t)
		gitTest(t, root, "checkout", "-qb", "reserved", first)
		reservedOID := branchPlacementTestCommit(t, root, "reserved-file", "reserved\n", "reserved")
		gitTest(t, root, "branch", "free", first)
		editor := filepath.Join(t.TempDir(), "sequence-editor.sh")
		localDeleteNoError(t, os.WriteFile(editor, []byte("#!/bin/sh\nset -eu\nprintf 'break\\n' > \"$1.tmp\"\ncat \"$1\" >> \"$1.tmp\"\nmv \"$1.tmp\" \"$1\"\n"), 0o700))
		rebase := exec.Command("git", "rebase", "-i", "master", "reserved")
		rebase.Dir = root
		rebase.Env = append(os.Environ(), "GIT_SEQUENCE_EDITOR="+editor)
		output, err := rebase.CombinedOutput()
		if err != nil {
			t.Fatalf("pause interactive rebase: %v\n%s", err, output)
		}
		commands := transactionCommands(t, root)
		rebaseDir := filepath.Join(commands.repoPaths.WorktreeGitDirPath(), "rebase-merge")
		assert.Equal(t, "refs/heads/reserved\n", string(mustRead(t, filepath.Join(rebaseDir, "head-name"))))
		assert.Equal(t, "HEAD", gitTest(t, root, "rev-parse", "--symbolic-full-name", "HEAD"))
		assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
		localDeleteNoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("staged\n"), 0o644))
		gitTest(t, root, "add", "tracked")
		localDeleteNoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("unstaged\x00bytes\n"), 0o644))
		localDeleteNoError(t, os.WriteFile(filepath.Join(root, "untracked"), []byte("untracked\x00bytes\n"), 0o600))
		snapshot := func(dir string) map[string]string {
			t.Helper()
			files := map[string]string{}
			localDeleteNoError(t, filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					if entry.Name() == ".git" {
						return filepath.SkipDir
					}
					return nil
				}
				relative, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				files[relative] = string(mustRead(t, path))
				return nil
			}))
			return files
		}
		rebaseBefore, worktreeBefore := snapshot(rebaseDir), snapshot(root)
		indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
		headBefore := mustRead(t, filepath.Join(root, ".git", "HEAD"))
		reserved := localDeleteTarget(t, commands, "refs/heads/reserved", reservedOID)
		reservedPlan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{reserved}, nil, nil)
		localDeleteNoError(t, err)
		assert.Equal(t, "rebase", reservedPlan.Sources()[0].Info.Reservation.Kind)
		_, err = commands.BranchActionLocalDeleteCommand(reservedPlan.WithForceApproved())
		assert.ErrorContains(t, err, "reserved")
		free := localDeleteTarget(t, commands, "refs/heads/free", first)
		freePlan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{free}, nil, nil)
		localDeleteNoError(t, err)
		assert.True(t, freePlan.Sources()[0].Merged)
		cmd, err := commands.BranchActionLocalDeleteCommand(freePlan)
		localDeleteNoError(t, err)
		localDeleteNoError(t, cmd.Run())
		_, err = commands.ResolveBranchActionCommit("refs/heads/free")
		assert.Error(t, err)
		assert.Equal(t, reservedOID, gitTest(t, root, "rev-parse", "refs/heads/reserved"))
		assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
		assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
		assert.Equal(t, rebaseBefore, snapshot(rebaseDir))
		assert.Equal(t, worktreeBefore, snapshot(root))
	})
}

func TestBranchActionLocalDeleteBareHEADReservationIsAllowed(t *testing.T) {
	root, _, current := transactionHistory(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	gitTest(t, filepath.Dir(bare), "init", "--bare", "-q", bare)
	gitTest(t, bare, "fetch", "-q", root, "refs/heads/master:refs/heads/master")
	gitTest(t, bare, "symbolic-ref", "HEAD", "refs/heads/master")
	osCommand := oscommands.NewDummyOSCommand()
	paths := &RepoPaths{worktreePath: bare, worktreeGitDirPath: bare, repoPath: filepath.Dir(bare), repoGitDirPath: bare, repoName: "bare", isBareRepo: true}
	commands := NewBranchCommands(NewGitCommon(common.NewDummyCommon(), nil, osCommand.Cmd, osCommand, paths, nil, nil))
	target := localDeleteTarget(t, commands, "refs/heads/master", current)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	localDeleteNoError(t, err)
	assert.Empty(t, plan.Sources()[0].Info.Reservation.Kind)
	cmd, err := commands.BranchActionLocalDeleteCommand(plan.WithForceApproved())
	localDeleteNoError(t, err)
	localDeleteNoError(t, cmd.Run())
	_, err = commands.ResolveBranchActionCommit("refs/heads/master")
	assert.Error(t, err)
}

func TestBranchActionLocalDeleteDetachedHEADIsCapturedProof(t *testing.T) {
	root, _, current := transactionHistory(t)
	gitTest(t, root, "checkout", "--detach", "-q", current)
	gitTest(t, root, "branch", "selected", current)
	commands := transactionCommands(t, root)
	target := localDeleteTarget(t, commands, "refs/heads/selected", current)
	plan, err := commands.PrepareBranchActionLocalDelete([]BranchActionTarget{target}, nil, nil)
	localDeleteNoError(t, err)
	assert.True(t, plan.Sources()[0].Merged)
	cmd, err := commands.BranchActionLocalDeleteCommand(plan)
	localDeleteNoError(t, err)
	localDeleteNoError(t, cmd.Run())
	assert.Equal(t, "HEAD", gitTest(t, root, "rev-parse", "--symbolic-full-name", "HEAD"))
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"))
}
