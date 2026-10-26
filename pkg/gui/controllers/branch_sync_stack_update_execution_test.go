package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// These tests start after batch approval: modal collection and its choice
// semantics are covered by branch_sync_stack_update_flow_test.go.
func executionStackBatch(t *testing.T, method git_commands.PullIntegrationMethod) (*branchSyncPushFixture, *branchSyncCurrentPullUI, *branchStackUpdateFlow) {
	t.Helper()
	f, ui, batch := stackUpdateFixture(t, []string{"lower"}, true)
	base := batch.members[0].OID
	target := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	if target == base {
		branchSyncPushGit(t, f.root, "checkout", "topic")
		branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "incoming")
		target = branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
		branchSyncPushGit(t, f.root, "checkout", "main")
	}
	branchSyncPushGit(t, f.root, "config", "branch.main.remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/topic")
	review, _, err := f.git.Sync.PrepareLocalCurrentPullReview("main", base, "refs/heads/topic")
	if err != nil {
		t.Fatal(err)
	}
	current, err := f.git.Sync.PrepareCurrentPullIntegration(review, target, method)
	if err != nil {
		t.Fatal(err)
	}
	info, err := f.git.Sync.InspectBranchSync("lower")
	if err != nil {
		t.Fatal(err)
	}
	forward, err := f.git.Sync.PrepareBranchFastForward("lower", info.OID, target)
	if err != nil {
		t.Fatal(err)
	}
	batch.plans = []reviewedBranchUpdate{{current: &current}, {forward: &forward}}
	batch.members[0].OID = current.SourceOID()
	batch.members[1].OID = forward.OldOID()
	return f, ui, batch
}

func executionStackApproveAndRunOffPhase(t *testing.T, ui *branchSyncCurrentPullUI, batch *branchStackUpdateFlow) {
	t.Helper()
	if !batch.hasCompleteReview() {
		t.Fatal("fixture batch is not a complete backend prepared review")
	}
	if err := batch.execute(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.ui) != 1 {
		t.Fatalf("UI handoffs=%d, want one", len(ui.ui))
	}
}

func executionStackRunToNativeValidation(t *testing.T, ui *branchSyncCurrentPullUI) error {
	t.Helper()
	if err := ui.runUI(t); err != nil {
		return err
	}
	return ui.runBlockingWorker(t)
}

func TestBranchSyncStackUpdateExecutionLateCurrentChanges(t *testing.T) {
	f, ui, batch := executionStackBatch(t, git_commands.PullIntegrationFastForwardOnly)
	executionStackApproveAndRunOffPhase(t, ui, batch)
	refreshes := f.ui.refreshes
	item := syncOperationItem(f.git, "main")
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if f.ui.state.GetItemOperation(item) != types.ItemOperationPulling {
		t.Fatal("UI handoff did not claim current operation")
	}
	f.ui.state.generation++
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "late current tip")
	err := ui.runBlockingWorker(t)
	if err == nil || !strings.Contains(err.Error(), "repository changed") {
		t.Fatalf("validation error=%v", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("stale current integration launched subprocess")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != batch.plans[1].targetOID() {
		t.Fatalf("lower tip=%s want advanced %s", got, batch.plans[1].targetOID())
	}
	if !strings.Contains(err.Error(), "lower") {
		t.Fatalf("progress omitted lower branch: %v", err)
	}
	if f.ui.refreshes != refreshes {
		t.Fatalf("refreshes=%d want unchanged %d after repository context change", f.ui.refreshes, refreshes)
	}
	if len(ui.blockingWorkers) != 0 || len(ui.ui) != 0 {
		t.Fatal("validation retried or refetched")
	}
	if f.ui.state.GetItemOperation(item) != types.ItemOperationNone || f.ui.state.GetItemOperation(syncOperationItem(f.git, "lower")) != types.ItemOperationNone {
		t.Fatal("operation flags were not cleared")
	}
}

func TestBranchSyncStackUpdateExecutionIgnoredRiskAndNativeFailures(t *testing.T) {
	t.Run("ignored bytes changed after approval", func(t *testing.T) {
		f, ui, batch := executionStackBatch(t, git_commands.PullIntegrationFastForwardOnly)
		ignored := filepath.Join(f.root, "ignored-risk")
		if err := os.WriteFile(ignored, []byte("approved bytes\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, ".git", "info", "exclude"), []byte("ignored-risk\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		branchSyncPushGit(t, f.root, "checkout", "topic")
		if err := os.WriteFile(filepath.Join(f.root, "ignored-risk"), []byte("incoming ignored bytes\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		branchSyncPushGit(t, f.root, "add", "-f", "ignored-risk")
		branchSyncPushGit(t, f.root, "commit", "-qm", "incoming ignored file")
		branchSyncPushGit(t, f.root, "checkout", "main")
		if err := os.WriteFile(ignored, []byte("approved bytes\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		// Reprepare through the real backend after the ignored file exists; this
		// records the risk, then explicitly models the user's prior approval.
		base := batch.plans[0].current.SourceOID()
		target := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
		review, _, err := f.git.Sync.PrepareLocalCurrentPullReview("main", base, "refs/heads/topic")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := f.git.Sync.PrepareCurrentPullIntegration(review, target, git_commands.PullIntegrationFastForwardOnly)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.IgnoredFiles()) == 0 {
			t.Fatal("fixture did not capture ignored-file risk")
		}
		approved := plan.WithIgnoredFilesApproved()
		batch.plans[0].current = &approved
		batch.members[0].OID = approved.SourceOID()
		executionStackApproveAndRunOffPhase(t, ui, batch)
		if err := os.WriteFile(ignored, []byte("user changed bytes\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		err = executionStackRunToNativeValidation(t, ui)
		if err == nil || !errors.Is(err, git_commands.ErrBranchSyncChanged) {
			t.Fatalf("changed ignored risk error=%v", err)
		}
		got, readErr := os.ReadFile(ignored)
		if readErr != nil || string(got) != "user changed bytes\n" {
			t.Fatalf("ignored contents=%q err=%v", got, readErr)
		}
		if !strings.Contains(err.Error(), "lower") || ui.subprocessCalls != 0 {
			t.Fatalf("missing progress or unexpected native run: %v", err)
		}
	})

	t.Run("native query failure after off phase", func(t *testing.T) {
		f, ui, batch := executionStackBatch(t, git_commands.PullIntegrationFastForwardOnly)
		executionStackApproveAndRunOffPhase(t, ui, batch)
		badConfig := []byte("[broken\n")
		configPath := filepath.Join(f.root, ".git", "config")
		originalConfig, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, badConfig, 0o600); err != nil {
			t.Fatal(err)
		}
		err = executionStackRunToNativeValidation(t, ui)
		if err == nil || strings.Contains(err.Error(), "branch or sync target changed") {
			t.Fatalf("expected native query error, got %v", err)
		}
		if !strings.Contains(err.Error(), "lower") || ui.subprocessCalls != 0 {
			t.Fatalf("query failure lost progress or launched subprocess: %v", err)
		}
		got, readErr := os.ReadFile(configPath)
		if readErr != nil || string(got) != string(badConfig) {
			t.Fatalf("malformed config was repaired: %v", readErr)
		}
		if len(ui.blockingWorkers) != 0 || len(ui.ui) != 0 {
			t.Fatal("query error queued retry/refetch")
		}
		// Restore only so fixture teardown and later native assertions see a
		// usable repository; the assertion above verifies production left it bad.
		if err := os.WriteFile(configPath, originalConfig, 0o600); err != nil {
			t.Fatal(err)
		}
	})
}

func TestBranchSyncStackUpdateExecutionNativeHookFailure(t *testing.T) {
	f, ui, batch := executionStackBatch(t, git_commands.PullIntegrationFastForwardOnly)
	hookDir := filepath.Join(f.root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(f.root, "NATIVE_HOOK_BLOCK")
	hook := "#!/bin/sh\nprintf NATIVE_HOOK_BLOCK > '" + marker + "'\nexit 1\n"
	if err := os.WriteFile(filepath.Join(hookDir, "pre-merge-commit"), []byte(hook), 0o700); err != nil {
		t.Fatal(err)
	}
	// Force a real merge commit attempt, preserving a native failure boundary.
	branchSyncPushGit(t, f.root, "checkout", "topic")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "incoming merge tip")
	branchSyncPushGit(t, f.root, "checkout", "main")
	// Existing plan pins an older target; use a divergent current commit so
	// native merge is required and reconstruct the plans with real backend data.
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "local divergence")
	target := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	review, _, err := f.git.Sync.PrepareLocalCurrentPullReview("main", branchSyncPushGit(t, f.root, "rev-parse", "HEAD"), "refs/heads/topic")
	if err != nil {
		t.Fatal(err)
	}
	current, err := f.git.Sync.PrepareCurrentPullIntegration(review, target, git_commands.PullIntegrationMerge)
	if err != nil {
		t.Fatal(err)
	}
	batch.plans[0].current = &current
	forward, err := f.git.Sync.PrepareBranchFastForward("lower", batch.members[1].OID, target)
	if err != nil {
		t.Fatal(err)
	}
	batch.plans[1].forward = &forward
	batch.members[0].OID = current.SourceOID()
	batch.members[1].OID = forward.OldOID()
	executionStackApproveAndRunOffPhase(t, ui, batch)
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("native attempts=%d want 1", ui.subprocessCalls)
	}
	err = ui.runBlockingWorker(t)
	if err == nil || !strings.Contains(err.Error(), "lower") || !strings.Contains(err.Error(), "Not committing merge") {
		t.Fatalf("hook failure report=%v", err)
	}
	if got, readErr := os.ReadFile(marker); readErr != nil || string(got) != "NATIVE_HOOK_BLOCK" {
		t.Fatalf("native hook marker=%q err=%v", got, readErr)
	}
	if ui.subprocessCalls != 1 || len(ui.blockingWorkers) != 0 || len(ui.ui) != 0 {
		t.Fatal("native failure retried")
	}
	if strings.Count(err.Error(), "lower") != 1 || strings.Count(err.Error(), "Not committing merge") != 1 {
		t.Fatalf("progress/native cause was not reported exactly once: %v", err)
	}

	t.Run("completion dispatch failure joins native failure once", func(t *testing.T) {
		f, ui, batch := executionStackBatch(t, git_commands.PullIntegrationFastForwardOnly)
		hookDir := filepath.Join(f.root, ".git", "hooks")
		if err := os.MkdirAll(hookDir, 0o700); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(f.root, "NATIVE_HOOK_BLOCK")
		hook := "#!/bin/sh\nprintf NATIVE_HOOK_BLOCK > '" + marker + "'\nexit 1\n"
		if err := os.WriteFile(filepath.Join(hookDir, "pre-merge-commit"), []byte(hook), 0o700); err != nil {
			t.Fatal(err)
		}
		branchSyncPushGit(t, f.root, "checkout", "topic")
		branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "incoming merge tip")
		branchSyncPushGit(t, f.root, "checkout", "main")
		branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "local divergence")
		target := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
		review, _, err := f.git.Sync.PrepareLocalCurrentPullReview("main", branchSyncPushGit(t, f.root, "rev-parse", "HEAD"), "refs/heads/topic")
		if err != nil {
			t.Fatal(err)
		}
		current, err := f.git.Sync.PrepareCurrentPullIntegration(review, target, git_commands.PullIntegrationMerge)
		if err != nil {
			t.Fatal(err)
		}
		forward, err := f.git.Sync.PrepareBranchFastForward("lower", batch.members[1].OID, target)
		if err != nil {
			t.Fatal(err)
		}
		batch.plans[0].current, batch.plans[1].forward = &current, &forward
		batch.members[0].OID, batch.members[1].OID = current.SourceOID(), forward.OldOID()
		ui.dispatchFailAt = 3 // ref worker, current validation worker, completion worker
		executionStackApproveAndRunOffPhase(t, ui, batch)
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runBlockingWorker(t); err != nil {
			t.Fatal(err)
		}
		err = ui.runUI(t)
		if err == nil || !strings.Contains(err.Error(), "Not committing merge") || !strings.Contains(err.Error(), "worker dispatch failure") || !strings.Contains(err.Error(), "lower") {
			t.Fatalf("joined completion error=%v", err)
		}
		for _, fragment := range []string{"Not committing merge", "worker dispatch failure", "lower"} {
			if n := strings.Count(err.Error(), fragment); n != 1 {
				t.Errorf("%q count=%d in %v", fragment, n, err)
			}
		}
		if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
			t.Fatal("completion dispatch failure left current busy")
		}
	})
}

func TestBranchSyncStackUpdateExecutionConfiguredRebaseAndDirtyPreflight(t *testing.T) {
	t.Run("configured default rebase preserves unrelated ref and untracked bytes", func(t *testing.T) {
		f, ui, batch := executionStackBatch(t, git_commands.PullIntegrationDefault)
		branchSyncPushGit(t, f.root, "config", "pull.rebase", "true")
		branchSyncPushGit(t, f.root, "config", "rebase.updateRefs", "true")
		// A harmless untracked file survives native rebase.
		untracked := filepath.Join(f.root, "harmless-untracked")
		if err := os.WriteFile(untracked, []byte("keep me\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "local commit")
		// Reconstruct a real configured rebase plan from a divergent fork point.
		branchSyncPushGit(t, f.root, "checkout", "topic")
		branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "upstream after fork")
		target := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
		branchSyncPushGit(t, f.root, "checkout", "main")
		branchSyncPushGit(t, f.root, "config", "branch.main.remote", ".")
		branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/topic")
		review, _, err := f.git.Sync.PrepareLocalCurrentPullReview("main", branchSyncPushGit(t, f.root, "rev-parse", "HEAD"), "refs/heads/topic")
		if err != nil {
			t.Fatal(err)
		}
		current, err := f.git.Sync.PrepareCurrentPullIntegration(review, target, git_commands.PullIntegrationDefault)
		if err != nil {
			t.Fatal(err)
		}
		if current.Method() != git_commands.PullIntegrationRebase {
			t.Fatalf("configured default method=%v want rebase", current.Method())
		}
		batch.plans[0].current = &current
		forward, err := f.git.Sync.PrepareBranchFastForward("lower", batch.members[1].OID, target)
		if err != nil {
			t.Fatal(err)
		}
		batch.plans[1].forward = &forward
		batch.members[0].OID = current.SourceOID()
		batch.members[1].OID = forward.OldOID()
		branchSyncPushGit(t, f.root, "branch", "unrelated-point", current.SourceOID())
		unrelated := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/unrelated-point")
		executionStackApproveAndRunOffPhase(t, ui, batch)
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runBlockingWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runBlockingWorker(t); err != nil {
			t.Fatal(err)
		}
		if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD^"); got != target {
			t.Fatalf("rebased current parent=%s want target %s", got, target)
		}
		if got := branchSyncPushGit(t, f.root, "merge-base", "main", "topic"); got != target {
			t.Fatalf("rebased current does not descend from approved target: merge-base=%s target=%s", got, target)
		}
		if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/unrelated-point"); got != unrelated {
			t.Fatalf("rebase.updateRefs moved unrelated ref from %s to %s", unrelated, got)
		}
		if got, err := os.ReadFile(untracked); err != nil || string(got) != "keep me\n" {
			t.Fatalf("untracked bytes=%q err=%v", got, err)
		}
		if got := branchSyncPushGit(t, f.root, "stash", "list"); got != "" {
			t.Fatalf("rebase created stash entries: %s", got)
		}
	})

	t.Run("tracked dirt after approval blocks all refs", func(t *testing.T) {
		f, ui, batch := executionStackBatch(t, git_commands.PullIntegrationRebase)
		if err := batch.execute(); err != nil {
			t.Fatal(err)
		}
		// Introduce tracked dirt after approval but before ref preflight.
		if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("late dirty\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := ui.runBlockingWorker(t)
		if err == nil || !errors.Is(err, git_commands.ErrRebaseRequiresClean) {
			t.Fatalf("dirty rebase preflight error=%v", err)
		}
		if ui.subprocessCalls != 0 {
			t.Fatal("dirty rebase launched native command")
		}
		if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != batch.members[1].OID {
			t.Fatal("dirty preflight wrote the lower ref")
		}
	})
}
