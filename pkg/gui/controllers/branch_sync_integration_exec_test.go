package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// This adapter deliberately queues both worker and UI work so tests can inspect
// the busy interval and change repositories at each asynchronous boundary.
type branchSyncIntegrationExecUI struct {
	*branchSyncPushTestUI
	blockingWorkers []func(gocui.Task) error
	subprocessCalls int
	subprocessErr   error
	dispatchCalls   int
	dispatchFailAt  int
}

func (ui *branchSyncIntegrationExecUI) WithWaitingStatusBlockingInput(_ types.WaitingStatusOpts, fn func(gocui.Task) error) error {
	ui.dispatchCalls++
	if ui.dispatchCalls == ui.dispatchFailAt {
		return errors.New("worker dispatch failure")
	}
	ui.blockingWorkers = append(ui.blockingWorkers, fn)
	return nil
}

func (ui *branchSyncIntegrationExecUI) Refresh(opts types.RefreshOptions) {
	ui.branchSyncPushTestUI.RefreshFromWorker(opts)
}

func (ui *branchSyncIntegrationExecUI) RunSubprocess(cmd *oscommands.CmdObj) (bool, error) {
	ui.subprocessCalls++
	if ui.subprocessErr != nil {
		return false, ui.subprocessErr
	}
	return true, cmd.Run()
}

func (ui *branchSyncIntegrationExecUI) runBlockingWorker(t *testing.T) error {
	t.Helper()
	if len(ui.blockingWorkers) == 0 {
		t.Fatal("expected queued blocking worker")
	}
	fn := ui.blockingWorkers[0]
	ui.blockingWorkers = ui.blockingWorkers[1:]
	return fn(nil)
}

func integrationExecPlan(t *testing.T, fixture *branchSyncPushFixture, method git_commands.PullIntegrationMethod) git_commands.CurrentPullIntegrationPlan {
	t.Helper()
	source := branchSyncPushGit(t, fixture.root, "rev-parse", "refs/heads/main")
	target := branchSyncPushGit(t, fixture.root, "rev-parse", "refs/heads/topic")
	review, err := fixture.git.Sync.PrepareCurrentPullReview("main", source, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := fixture.git.Sync.PrepareCurrentPullIntegration(review, target, method)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func integrationExecFixture(t *testing.T) (*branchSyncPushFixture, *branchSyncTarget, *branchSyncIntegrationExecUI) {
	t.Helper()
	f := newBranchSyncPushFixture(t)
	ui := &branchSyncIntegrationExecUI{branchSyncPushTestUI: f.ui}
	// The fake UI owns terminal execution in these tests; no credential helper
	// or external editor is needed for the noninteractive fast-forward command.
	f.ui = ui.branchSyncPushTestUI
	f.sync.c.HelperCommon.IGuiCommon = ui
	target := newBranchSyncTarget(f.sync.c, branchSyncSource{name: "main"})
	return f, target, ui
}

func TestBranchSyncIntegrationExecHandoffAndFastForward(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	if err := target.runReviewedIntegration(plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	item := syncOperationItem(f.git, "main")
	if f.ui.state.GetItemOperation(item) != types.ItemOperationPulling {
		t.Fatal("branch was not marked busy before worker dispatch")
	}
	if err := target.runReviewedIntegration(plan, nil, nil); err == nil {
		t.Fatal("concurrent invocation was accepted")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.ui) != 1 || f.ui.state.GetItemOperation(item) != types.ItemOperationPulling {
		t.Fatal("busy flag was not retained through worker to UI handoff")
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 1 || f.ui.state.GetItemOperation(item) != types.ItemOperationPulling {
		t.Fatal("terminal execution did not occur with busy flag retained")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if f.ui.state.GetItemOperation(item) != types.ItemOperationNone {
		t.Fatal("busy flag remained after terminal completion")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main"); got != plan.TargetOID() {
		t.Fatalf("fast-forward tip = %s, want %s", got, plan.TargetOID())
	}
}

func TestBranchSyncIntegrationExecRejectsPlanForDifferentBranch(t *testing.T) {
	f, _, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	otherTarget := newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic"})
	if err := otherTarget.runReviewedIntegration(plan, nil, nil); err == nil {
		t.Fatal("plan for another branch was accepted")
	}
	if len(ui.blockingWorkers) != 0 || len(ui.ui) != 0 || ui.subprocessCalls != 0 {
		t.Fatal("mismatched branch plan queued work or launched a subprocess")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "topic")) != types.ItemOperationNone {
		t.Fatal("mismatched branch plan claimed the captured branch busy flag")
	}
}

func TestBranchSyncIntegrationExecAfterExecutionRunsOnWorker(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	called := false
	if err := target.runReviewedIntegration(plan, nil, func(actual error) error {
		called = true
		if actual != nil {
			return actual
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("afterExecution ran on the UI thread")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("afterExecution was not called after native command completion")
	}
}

func TestBranchSyncIntegrationExecPreparationErrorsCleanup(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/main", plan.TargetOID())
	if err := target.runReviewedIntegration(plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchSyncChanged) {
		t.Fatalf("prepare error = %v, want changed plan", err)
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("prepare failure left branch busy")
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("prepare failure launched a native command")
	}
}

func TestBranchSyncIntegrationExecReviewCallbackIsQueued(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/main", plan.TargetOID())
	called := false
	if err := target.runReviewedIntegration(plan, func(error) error { called = true; return nil }, nil); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if called || len(ui.ui) != 1 || ui.subprocessCalls != 0 {
		t.Fatal("review callback ran on worker or command executed")
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !called || ui.subprocessCalls != 0 {
		t.Fatal("review callback was not handled exclusively on UI")
	}
}

func TestBranchSyncIntegrationExecRepositorySwitchRefusal(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	if err := target.runReviewedIntegration(plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	other := newBranchSyncPushFixture(t)
	originalTip := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	oldGit := f.ui.git
	f.ui.git = other.git
	if err := ui.runUI(t); err == nil {
		t.Fatal("stale UI callback launched after repository switch")
	}
	if ui.subprocessCalls != 0 || branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") != originalTip {
		t.Fatal("stale UI callback executed a command against the captured repository")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(oldGit, "main")) != types.ItemOperationNone {
		t.Fatal("stale operation flag was not cleared")
	}
}

func TestBranchSyncIntegrationExecGenerationChangeRefusal(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	originalTip := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	if err := target.runReviewedIntegration(plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation++
	if err := ui.runUI(t); err == nil {
		t.Fatal("stale UI callback accepted a repository generation change")
	}
	if ui.subprocessCalls != 0 || branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") != originalTip {
		t.Fatal("stale UI callback launched a native command")
	}
	if f.ui.refreshes != 0 {
		t.Fatal("stale UI callback refreshed after repository generation changed")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("captured operation flag was not cleared")
	}
}

func TestBranchSyncIntegrationExecHeadChangeBeforeTerminalRefusesLaunch(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	reviewCalled := false
	if err := target.runReviewedIntegration(plan, func(err error) error {
		reviewCalled = errors.Is(err, git_commands.ErrBranchSyncChanged)
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !reviewCalled || ui.subprocessCalls != 0 {
		t.Fatal("changed HEAD launched the reviewed command instead of requiring fresh review")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("HEAD change left selected branch busy")
	}
}

func TestBranchSyncIntegrationExecRepositorySwitchAfterTerminal(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	called := false
	if err := target.runReviewedIntegration(plan, nil, func(err error) error {
		called = true
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	other := newBranchSyncPushFixture(t)
	f.ui.git = other.git
	f.ui.state.generation++
	if err := ui.runBlockingWorker(t); err == nil {
		t.Fatal("post-terminal worker accepted a changed repository")
	}
	if called {
		t.Fatal("afterExecution ran after a repository switch")
	}
	if ui.subprocessCalls != 1 || f.ui.refreshes != 0 {
		t.Fatal("stale post-terminal work ran a command or refreshed the switched repository")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("captured repository operation flag was not cleared")
	}
}

func TestBranchSyncIntegrationExecNativeFailureCleanup(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	ui.subprocessErr = errors.New("terminal failure")
	if err := target.runReviewedIntegration(plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil || !strings.Contains(err.Error(), "terminal failure") {
		t.Fatalf("terminal failure not propagated: %v", err)
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("native failure left branch busy")
	}
}

func TestBranchSyncIntegrationExecPostTerminalDispatchFailureCleanup(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationFastForwardOnly)
	nativeErr := errors.New("terminal native failure")
	ui.subprocessErr = nativeErr
	ui.dispatchFailAt = 2
	if err := target.runReviewedIntegration(plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err == nil || !errors.Is(err, nativeErr) || !strings.Contains(err.Error(), "worker dispatch failure") {
		t.Fatalf("native and post-terminal dispatch errors were not preserved: %v", err)
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("post-terminal dispatch failure left branch busy")
	}
	if f.ui.refreshes != 1 {
		t.Fatalf("UI-safe refresh count = %d, want 1", f.ui.refreshes)
	}
}

func TestBranchSyncIntegrationExecInteractiveCommandUsesTerminal(t *testing.T) {
	f, target, ui := integrationExecFixture(t)
	// Give the current branch a commit that must be replayed on the incoming tip.
	if err := os.WriteFile(filepath.Join(f.root, "main-only"), []byte("change"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "main-only")
	branchSyncPushGit(t, f.root, "commit", "-qm", "main extra")
	branchSyncPushGit(t, f.root, "config", "core.editor", "true")
	branchSyncPushGit(t, f.root, "config", "sequence.editor", "true")
	plan := integrationExecPlan(t, f, git_commands.PullIntegrationRebaseInteractive)
	if err := target.runReviewedIntegration(plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 1 || len(ui.blockingWorkers) != 1 {
		t.Fatal("interactive rebase did not use terminal execution and queue postprocessing")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
}
