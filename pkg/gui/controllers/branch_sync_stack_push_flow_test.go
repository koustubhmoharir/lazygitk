package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func stackPushFixture(t *testing.T) (*branchSyncPushFixture, *branchSyncIntegrationExecUI, *branchStackPushFlow) {
	t.Helper()
	f, _, ui := integrationExecFixture(t)
	// Give main two commits so topic has a three-branch ancestry: topic,
	// unpublished lower, then its older base.
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(f.root, "main-only"), []byte("middle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "main-only")
	branchSyncPushGit(t, f.root, "commit", "-qm", "middle")
	mainOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	baseOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main^")
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	branchSyncPushGit(t, f.root, "reset", "--hard", "main")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("topic tip\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "commit", "-qam", "topic tip")
	topicOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "branch", "lower", mainOID)
	branchSyncPushGit(t, f.root, "branch", "bottom", baseOID)
	branchSyncPushGit(t, f.root, "config", "branch.lower.pushRemote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.bottom.pushRemote", "origin")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte{'s', 't', 'a', 'g', 'e', 'd', 0, 0xff, '\n'}, 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte{'u', 'n', 's', 't', 'a', 'g', 'e', 'd', 0, 0xfe, '\n'}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "stack-untracked"), []byte{0x00, 0xff, 'u', 'n', 't', 'r', 'a', 'c', 'k', 'e', 'd', '\n'}, 0o600); err != nil {
		t.Fatal(err)
	}
	target := newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true})
	batch := &branchStackPushFlow{target: target, members: []git_commands.BranchStackMember{
		{Name: "topic", OID: topicOID}, {Name: "lower", OID: mainOID}, {Name: "bottom", OID: baseOID},
	}}
	return f, ui, batch
}

func collectStackPushPlans(t *testing.T, f *branchSyncPushFixture, ui *branchSyncIntegrationExecUI, batch *branchStackPushFlow) {
	t.Helper()
	if err := batch.reviewNext(); err != nil {
		t.Fatal(err)
	}
	for len(batch.plans) < len(batch.members) {
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		menu := f.ui.menus[len(f.ui.menus)-1]
		wantProgress := fmt.Sprintf("branch %d of %d", len(batch.plans)+1, len(batch.members))
		if !strings.Contains(strings.ToLower(menu.Prompt), wantProgress) || !strings.Contains(strings.ToLower(menu.Prompt), "no branch is pushed") {
			t.Fatalf("branch review omitted progress/no-push context: %q", menu.Prompt)
		}
		if got := menuItem(t, menu, "branch.push-selected.execute").Label; got != f.sync.c.Tr.BranchSyncStackPreparePush {
			t.Fatalf("collection action label = %q", got)
		}
		if err := menuItem(t, menu, "branch.push-selected.execute").OnPress(); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if len(f.ui.confirms) == 0 {
			t.Fatalf("branch %q did not reach ordinary review", batch.members[len(batch.plans)].Name)
		}
		confirm := f.ui.confirms[len(f.ui.confirms)-1]
		if err := confirm.HandleConfirm(); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
	}
	if len(ui.blockingWorkers) != 0 {
		t.Fatal("plan collection queued a blocking network worker")
	}
}

func TestBranchSyncStackPushCollectsReviewsBeforeAnyPush(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	collectStackPushPlans(t, f, ui, batch)
	if len(batch.plans) != 3 || len(f.ui.menus) == 0 {
		t.Fatalf("plans = %d; menus = %d", len(batch.plans), len(f.ui.menus))
	}
	if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/topic"); err == nil {
		t.Fatal("topic destination exists before final approval")
	}
	if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/lower"); err == nil {
		t.Fatal("lower destination exists before final approval")
	}
	if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/bottom"); err == nil {
		t.Fatal("bottom destination exists before final approval")
	}
	final := f.ui.menus[len(f.ui.menus)-1]
	if final.Items[0].Verb != "cancel" {
		t.Fatal("batch approval is not cancel-first")
	}
	if final.Items[len(final.Items)-1].CommandID != "branch.push-stack.execute" {
		t.Fatal("batch execute is not the final menu item")
	}
	inspected := false
	for _, item := range final.Items {
		if item.CommandID == "branch.push-stack.inspect-plan" {
			if err := item.OnPress(); err != nil {
				t.Fatal(err)
			}
			details := f.ui.menus[len(f.ui.menus)-1]
			_ = menuItem(t, details, "branch.push-stack.back")
			if len(ui.blockingWorkers) != 0 {
				t.Fatal("inspecting a captured row queued a push")
			}
			if err := menuItem(t, details, "branch.push-stack.back").OnPress(); err != nil {
				t.Fatal(err)
			}
			inspected = true
			break
		}
	}
	if !inspected {
		t.Fatal("final batch menu omitted plan inspection rows")
	}
	if err := final.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.blockingWorkers) != 0 {
		t.Fatal("canceling final batch approval queued a push")
	}
}

func execGitOptional(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	return cmd.CombinedOutput()
}

func stackPushFileSnapshot(t *testing.T, f *branchSyncPushFixture) map[string][]byte {
	t.Helper()
	snapshot := make(map[string][]byte)
	for _, path := range []string{".git/HEAD", ".git/index", "tracked", "stack-untracked"} {
		contents, err := os.ReadFile(filepath.Join(f.root, path))
		if err != nil {
			t.Fatalf("read snapshot %s: %v", path, err)
		}
		snapshot[path] = contents
	}
	return snapshot
}

func assertStackPushFileSnapshot(t *testing.T, f *branchSyncPushFixture, before map[string][]byte) {
	t.Helper()
	after := stackPushFileSnapshot(t, f)
	for path, contents := range before {
		if !bytes.Equal(after[path], contents) {
			t.Errorf("stack push changed raw snapshot %s", path)
		}
	}
}

func TestBranchSyncStackPushExecutesCapturedPlansAndConsumesApproval(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	collectStackPushPlans(t, f, ui, batch)
	filesBefore := stackPushFileSnapshot(t, f)
	headBefore := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	mainBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	final := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, final, "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	first := syncOperationItem(f.git, "topic")
	second := syncOperationItem(f.git, "lower")
	third := syncOperationItem(f.git, "bottom")
	if f.ui.state.GetItemOperation(first) != types.ItemOperationPushing || f.ui.state.GetItemOperation(second) != types.ItemOperationPushing || f.ui.state.GetItemOperation(third) != types.ItemOperationPushing {
		t.Fatal("batch did not claim every branch busy before dispatch")
	}
	if err := batch.execute(); err == nil {
		t.Fatal("duplicate execution accepted")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/topic"); got != batch.members[0].OID {
		t.Fatalf("topic destination = %s, want %s", got, batch.members[0].OID)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/lower"); got != batch.members[1].OID {
		t.Fatalf("lower destination = %s, want %s", got, batch.members[1].OID)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/bottom"); got != batch.members[2].OID {
		t.Fatalf("bottom destination = %s, want %s", got, batch.members[2].OID)
	}
	if f.ui.state.GetItemOperation(first) != types.ItemOperationNone || f.ui.state.GetItemOperation(second) != types.ItemOperationNone || f.ui.state.GetItemOperation(third) != types.ItemOperationNone {
		t.Fatal("batch left busy flags set")
	}
	if !strings.EqualFold(branchSyncPushGit(t, f.root, "symbolic-ref", "--short", "HEAD"), "main") {
		t.Fatal("stack push changed HEAD")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != headBefore {
		t.Fatalf("stack push changed HEAD OID from %s to %s", headBefore, got)
	}
	assertStackPushFileSnapshot(t, f, filesBefore)
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main"); got != mainBefore {
		t.Fatalf("stack push changed unrelated main ref from %s to %s", mainBefore, got)
	}
}

func TestBranchSyncStackPushRejectsDuplicatePlanAcceptanceCallback(t *testing.T) {
	f, _, batch := stackPushFixture(t)
	if err := batch.reviewNext(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("confirmation count = %d", len(f.ui.confirms))
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.ui) != 1 {
		t.Fatalf("acceptance callback count = %d", len(f.ui.ui))
	}
	callback := f.ui.ui[0]
	f.ui.ui = f.ui.ui[1:]
	if err := callback(); err != nil {
		t.Fatal(err)
	}
	if err := callback(); !errors.Is(err, git_commands.ErrBranchSyncChanged) {
		t.Fatalf("duplicate callback error = %v, want changed-plan refusal", err)
	}
	if len(batch.plans) != 1 {
		t.Fatalf("duplicate callback accepted %d plans", len(batch.plans))
	}
}

func TestBranchSyncStackPushRejectsAliasedRemoteDestinationDuringCollection(t *testing.T) {
	f, _, batch := stackPushFixture(t)
	if err := batch.reviewNext(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(batch.plans) != 1 {
		t.Fatalf("first plan count = %d", len(batch.plans))
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "remote", "add", "alias", f.remote)
	branchSyncPushGit(t, f.root, "config", "branch.lower.pushRemote", "alias")
	menu := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, menu, "branch.push-selected.destination").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[len(f.ui.prompts)-1].HandleConfirm("topic"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	err := f.ui.runUI(t)
	if err == nil || strings.Contains(err.Error(), f.remote) {
		t.Fatalf("aliased destination error = %v; want safe overlap refusal", err)
	}
	if len(batch.plans) != 1 {
		t.Fatalf("overlapping alias plan was accepted; count = %d", len(batch.plans))
	}
	if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/topic"); err == nil {
		t.Fatal("destination overlap check pushed before batch approval")
	}
}

func TestBranchSyncStackPushMidBatchNativeFailureKeepsCompletedPush(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	// A native pre-receive hook rejects lower after topic has completed.
	if err := os.MkdirAll(filepath.Join(f.remote, "hooks"), 0o700); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\nwhile read old new ref; do\n  if [ \"$ref\" = refs/heads/lower ]; then echo rejected >&2; exit 1; fi\ndone\nexit 0\n"
	if err := os.WriteFile(filepath.Join(f.remote, "hooks", "pre-receive"), []byte(hook), 0o700); err != nil {
		t.Fatal(err)
	}
	collectStackPushPlans(t, f, ui, batch)
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	err := ui.runBlockingWorker(t)
	if err == nil || !strings.Contains(err.Error(), "lower") || !strings.Contains(err.Error(), "topic") {
		t.Fatalf("mid-batch error = %v", err)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/topic"); got != batch.members[0].OID {
		t.Fatalf("completed topic push was rolled back or absent: %s", got)
	}
	if _, showErr := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/lower"); showErr == nil {
		t.Fatal("rejected lower push unexpectedly succeeded")
	}
	if _, showErr := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/bottom"); showErr == nil {
		t.Fatal("third push was attempted after native rejection")
	}
	if err := batch.execute(); err == nil {
		t.Fatal("failed batch approval was reusable")
	}
}

func TestBranchSyncStackPushSourceChangeRefusesWholeBatchBeforeWrites(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	collectStackPushPlans(t, f, ui, batch)
	oldTip := batch.members[0].OID
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", batch.members[1].OID, oldTip)
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil {
		t.Fatal("changed source passed whole-batch preflight")
	}
	for _, name := range []string{"topic", "lower", "bottom"} {
		if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/"+name); err == nil {
			t.Fatalf("preflight failure pushed %s", name)
		}
	}
	if err := batch.execute(); err == nil {
		t.Fatal("preflight-failed approval was reusable")
	}
}

func TestBranchSyncStackPushRepositoryChangeBeforeQueuedWorkerDoesNotRefresh(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	collectStackPushPlans(t, f, ui, batch)
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	other := f.git
	f.ui.git = nil
	err := ui.runBlockingWorker(t)
	if err == nil {
		t.Fatal("queued batch worker accepted a repository pointer change")
	}
	f.ui.git = other
	if f.ui.refreshes != 0 {
		t.Fatal("stale batch worker refreshed a different repository")
	}
	for _, name := range []string{"topic", "lower", "bottom"} {
		if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/"+name); err == nil {
			t.Fatalf("stale worker pushed %s", name)
		}
	}
}

func TestBranchSyncStackPushRemoteConfigurationChangeRefusesWholeBatch(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	collectStackPushPlans(t, f, ui, batch)
	changedRemote := filepath.Join(t.TempDir(), "changed.git")
	branchSyncPushGit(t, t.TempDir(), "init", "--bare", "-q", changedRemote)
	branchSyncPushGit(t, f.root, "remote", "set-url", "origin", changedRemote)
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil {
		t.Fatal("changed remote configuration passed whole-batch preflight")
	}
	for _, remote := range []string{f.remote, changedRemote} {
		for _, name := range []string{"topic", "lower", "bottom"} {
			if _, err := execGitOptional(remote, "show-ref", "--verify", "refs/heads/"+name); err == nil {
				t.Fatalf("configuration preflight failure pushed %s to %s", name, remote)
			}
		}
	}
}

func TestBranchSyncStackPushDispatchFailureClearsEveryBusyFlag(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	collectStackPushPlans(t, f, ui, batch)
	ui.dispatchFailAt = 1
	err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress()
	if err == nil || !strings.Contains(err.Error(), "worker dispatch failure") {
		t.Fatalf("dispatch error = %v", err)
	}
	for _, name := range []string{"topic", "lower", "bottom"} {
		if f.ui.state.GetItemOperation(syncOperationItem(f.git, name)) != types.ItemOperationNone {
			t.Fatalf("dispatch failure left %s busy", name)
		}
		if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/"+name); err == nil {
			t.Fatalf("dispatch failure pushed %s", name)
		}
	}
	if err := batch.execute(); err == nil {
		t.Fatal("dispatch-failed approval was reusable")
	}
}

func TestBranchSyncStackPushCustomFirstDestinationUsesExplicitFullRef(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	if err := batch.reviewNext(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	firstMenu := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, firstMenu, "branch.push-selected.destination").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[len(f.ui.prompts)-1].HandleConfirm("releases/topic-v2"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.confirms[len(f.ui.confirms)-1].Prompt, "refs/heads/releases/topic-v2") {
		t.Fatal("destination review omitted the normalized full ref")
	}
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	for len(batch.plans) < len(batch.members) {
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		menu := f.ui.menus[len(f.ui.menus)-1]
		if err := menuItem(t, menu, "branch.push-selected.execute").OnPress(); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
	}
	if got := batch.plans[0].Options().DestinationRef; got != "refs/heads/releases/topic-v2" {
		t.Fatalf("captured first destination = %q", got)
	}
	final := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, final, "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/releases/topic-v2"); got != batch.members[0].OID {
		t.Fatalf("custom ref tip = %s, want %s", got, batch.members[0].OID)
	}
	if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/topic"); err == nil {
		t.Fatal("default topic destination was also pushed")
	}
	for i := 1; i < len(batch.members); i++ {
		if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/"+batch.members[i].Name); got != batch.members[i].OID {
			t.Fatalf("other member %s changed destination unexpectedly: %s", batch.members[i].Name, got)
		}
	}
}

func TestBranchSyncStackPushForceLeasePlanIsOpaqueAndPreflighted(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	leaseOID := batch.members[1].OID
	batch.members = batch.members[:1]
	leaseRef := "refs/reviewed/stack-topic-lease"
	branchSyncPushGit(t, f.root, "update-ref", leaseRef, leaseOID)
	if err := batch.reviewNext(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	memberMenu := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, memberMenu, "branch.push-selected.force-with-lease").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatal("force lease without a tracked tip did not prompt for a local lease ref")
	}
	if err := f.ui.prompts[0].HandleConfirm(leaseRef); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	forceReview := f.ui.menus[len(f.ui.menus)-1]
	if len(forceReview.Items) < 2 || forceReview.Items[0].Verb != "cancel" {
		t.Fatal("force review was not cancel-first")
	}
	if len(batch.plans) != 0 {
		t.Fatal("force plan was collected before explicit force approval")
	}
	if err := menuItem(t, forceReview, "branch.push-selected.force-confirm").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(batch.plans) != 1 {
		t.Fatalf("force plan count = %d", len(batch.plans))
	}
	plan := batch.plans[0]
	if !plan.Options().ForceWithLease || plan.LeaseRef() != leaseRef || plan.LeaseOID() != leaseOID {
		t.Fatalf("collected opaque lease plan lost approval inputs: opts=%+v ref=%q oid=%q", plan.Options(), plan.LeaseRef(), plan.LeaseOID())
	}
	branchSyncPushGit(t, f.root, "update-ref", leaseRef, batch.members[0].OID, leaseOID)
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	err := ui.runBlockingWorker(t)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "preflight") {
		t.Fatalf("changed force lease preflight error = %v", err)
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "topic")) != types.ItemOperationNone {
		t.Fatal("force lease preflight failure left topic busy")
	}
	for _, name := range []string{"topic", "lower", "bottom"} {
		if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/"+name); err == nil {
			t.Fatalf("stale force lease pushed %s", name)
		}
	}
}

func TestBranchSyncStackPushMultiURLRequiresCapturedApprovalAndPreflightsTransport(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	batch.members = batch.members[:1]
	oldURLs := []string{
		filepath.Join(t.TempDir(), "first.git"),
		filepath.Join(t.TempDir(), "second.git"),
	}
	newURLs := []string{
		filepath.Join(t.TempDir(), "changed-first.git"),
		filepath.Join(t.TempDir(), "changed-second.git"),
	}
	for _, remote := range append(append([]string{}, oldURLs...), newURLs...) {
		branchSyncPushGit(t, t.TempDir(), "init", "--bare", "-q", remote)
		if err := os.MkdirAll(filepath.Join(remote, "hooks"), 0o700); err != nil {
			t.Fatal(err)
		}
		hook := fmt.Sprintf("#!/bin/sh\nprintf called > %q\n", filepath.Join(remote, "hook-called"))
		if err := os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte(hook), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.pushurl", oldURLs[0])
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.pushurl", oldURLs[1])
	if err := batch.reviewNext(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	urlReview := f.ui.menus[len(f.ui.menus)-1]
	if urlReview.Items[0].Verb != "cancel" || menuItem(t, urlReview, "branch.push-selected.multiple-urls-approve") == nil {
		t.Fatal("multiple URL review omitted explicit cancel-first approval")
	}
	if len(batch.plans) != 0 {
		t.Fatal("multiple-URL plan entered the batch before explicit approval")
	}
	if err := menuItem(t, urlReview, "branch.push-selected.multiple-urls-approve").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(batch.plans) != 1 || len(batch.plans[0].ConfiguredPushURLs()) != 2 {
		t.Fatalf("captured plan lost multiple-URL approval: plans=%d", len(batch.plans))
	}
	if _, err := f.git.Sync.BranchPushCommand(batch.plans[0], nil); err != nil {
		t.Fatalf("captured multiple-URL approval was not retained by the opaque plan: %v", err)
	}
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.pushurl")
	for _, remote := range newURLs {
		branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.pushurl", remote)
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil {
		t.Fatal("changed multi-URL transport passed whole-batch preflight")
	}
	for _, remote := range append(append([]string{}, oldURLs...), newURLs...) {
		if _, err := execGitOptional(remote, "show-ref", "--verify", "refs/heads/topic"); err == nil {
			t.Fatalf("transport preflight failure contacted or changed %s", remote)
		}
		if _, err := os.Stat(filepath.Join(remote, "hook-called")); err == nil {
			t.Fatalf("transport preflight failure contacted push URL %s", remote)
		}
	}
}

func TestBranchSyncStackPushGenerationChangeAfterQueueClearsFlagsWithoutRefresh(t *testing.T) {
	f, ui, batch := stackPushFixture(t)
	collectStackPushPlans(t, f, ui, batch)
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-stack.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation++
	if err := ui.runBlockingWorker(t); err == nil {
		t.Fatal("queued batch worker accepted a repository generation change")
	}
	if f.ui.refreshes != 0 {
		t.Fatal("stale generation worker refreshed the repository")
	}
	for _, name := range []string{"topic", "lower", "bottom"} {
		if f.ui.state.GetItemOperation(syncOperationItem(f.git, name)) != types.ItemOperationNone {
			t.Fatalf("generation change left %s busy", name)
		}
		if _, err := execGitOptional(f.remote, "show-ref", "--verify", "refs/heads/"+name); err == nil {
			t.Fatalf("generation change pushed %s", name)
		}
	}
}

func TestBranchSyncStackPushCancelMidMemberReviewPushesNothing(t *testing.T) {
	f, _, batch := stackPushFixture(t)
	if err := batch.reviewNext(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	menu := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, menu, "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(batch.plans) != 1 {
		t.Fatalf("accepted first plan count = %d", len(batch.plans))
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	secondMemberMenu := f.ui.menus[len(f.ui.menus)-1]
	if err := secondMemberMenu.OnCancel(); err != nil {
		t.Fatal(err)
	}
	if len(batch.plans) != 1 {
		t.Fatal("canceling member review changed captured plan count")
	}
	for _, remote := range []string{f.remote} {
		for _, name := range []string{"topic", "lower", "bottom"} {
			if _, err := execGitOptional(remote, "show-ref", "--verify", "refs/heads/"+name); err == nil {
				t.Fatalf("mid-review cancellation pushed %s", name)
			}
		}
	}
}
