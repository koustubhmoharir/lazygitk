package controllers

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type stackUpdateRawSnapshot struct {
	files         map[string][]byte
	unrelatedRefs string
}

func stackUpdateSnapshot(t *testing.T, f *branchSyncPushFixture) stackUpdateRawSnapshot {
	t.Helper()
	files := map[string][]byte{}
	for _, path := range []string{".git/HEAD", ".git/index", ".git/config", "tracked", "stack-untracked"} {
		full := filepath.Join(f.root, path)
		contents, err := os.ReadFile(full)
		if err != nil {
			if os.IsNotExist(err) && path == "stack-untracked" {
				continue
			}
			t.Fatalf("read snapshot %s: %v", path, err)
		}
		files[path] = contents
	}
	refs := branchSyncPushGit(t, f.root, "for-each-ref", "refs/heads/main", "refs/heads/topic", "--format=%(refname) %(objectname)")
	return stackUpdateRawSnapshot{files: files, unrelatedRefs: refs}
}

func stackUpdateAssertWorktreeSnapshot(t *testing.T, f *branchSyncPushFixture, before stackUpdateRawSnapshot) {
	t.Helper()
	for path, want := range before.files {
		full := filepath.Join(f.root, path)
		got, err := os.ReadFile(full)
		if err != nil {
			t.Fatalf("read snapshot %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("batch changed raw snapshot %s", path)
		}
	}
	if got := branchSyncPushGit(t, f.root, "for-each-ref", "refs/heads/main", "refs/heads/topic", "--format=%(refname) %(objectname)"); got != before.unrelatedRefs {
		t.Errorf("batch changed unrelated refs: %q want %q", got, before.unrelatedRefs)
	}
}

func stackUpdateAllRefs(t *testing.T, f *branchSyncPushFixture) string {
	t.Helper()
	return branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)")
}

// This fixture deliberately uses local-dot upstreams: review and collection
// still traverse the user-facing update flow while Git performs no transport.
func stackUpdateFixture(t *testing.T, names []string, includeCurrent bool) (*branchSyncPushFixture, *branchSyncCurrentPullUI, *branchStackUpdateFlow) {
	t.Helper()
	f, ui := newBranchSyncCurrentPullFixture(t)
	base := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	for _, name := range names {
		branchSyncPushGit(t, f.root, "branch", name, base)
		branchSyncPushGit(t, f.root, "config", "branch."+name+".remote", ".")
		branchSyncPushGit(t, f.root, "config", "branch."+name+".merge", "refs/heads/topic")
	}
	if includeCurrent {
		branchSyncPushGit(t, f.root, "config", "branch.main.remote", ".")
		branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/topic")
	}
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "newer upstream")
	_ = branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	members := make([]git_commands.BranchStackMember, 0, len(names)+1)
	rootName := names[0]
	if includeCurrent {
		rootName = "main"
		members = append(members, git_commands.BranchStackMember{Name: "main", OID: base})
	}
	for _, name := range names {
		members = append(members, git_commands.BranchStackMember{Name: name, OID: base})
	}
	target := newBranchSyncTarget(f.sync.c, branchSyncSource{name: rootName, selected: true})
	batch := &branchStackUpdateFlow{target: target, members: members}
	return f, ui, batch
}

func collectStackUpdate(t *testing.T, f *branchSyncPushFixture, ui *branchSyncCurrentPullUI, batch *branchStackUpdateFlow, approved ...func()) {
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
		for _, check := range approved {
			check()
		}
		menu := f.ui.menus[len(f.ui.menus)-1]
		actionID := "branch.update-selected.fetch"
		if batch.members[len(batch.plans)].Name == "main" {
			actionID = "branch.pull-current.fetch"
		}
		if err := menuItem(t, menu, actionID).OnPress(); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if batch.members[len(batch.plans)].Name == "main" {
			stackUpdateChooseCurrentFFOnly(t, f)
		}
		if len(f.ui.confirms) == 0 {
			t.Fatal("local update did not create ordinary review")
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
	if len(batch.plans) != len(batch.members) {
		t.Fatalf("plans=%d members=%d", len(batch.plans), len(batch.members))
	}
	if ui.subprocessCalls != 0 || len(ui.blockingWorkers) != 0 {
		t.Fatal("collecting update reviews launched worktree integration or a mutation worker")
	}
}

// The current member reaches the integration choices after its local update
// review. Selecting fast-forward only keeps these batch tests on that route.
func stackUpdateChooseCurrentFFOnly(t *testing.T, f *branchSyncPushFixture) {
	t.Helper()
	stackUpdateDrain(t, f)
	menu := f.ui.menus[len(f.ui.menus)-1]
	if menu.Title != f.sync.c.Tr.BranchSyncIntegrationMenu {
		t.Fatalf("current review did not offer integration choices: title=%q", menu.Title)
	}
	if err := menuItem(t, menu, "branch.pull-current.ff-only").OnPress(); err != nil {
		t.Fatal(err)
	}
	stackUpdateDrain(t, f)
}

func stackUpdateDrain(t *testing.T, f *branchSyncPushFixture) {
	t.Helper()
	for len(f.ui.workers) > 0 || len(f.ui.ui) > 0 {
		if len(f.ui.workers) > 0 {
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
	}
}

func stackUpdateApprove(t *testing.T, f *branchSyncPushFixture) error {
	t.Helper()
	menu := f.ui.menus[len(f.ui.menus)-1]
	return menuItem(t, menu, "branch.update-stack.execute").OnPress()
}

func TestBranchSyncStackUpdateSelectedOffBranchBatchReviewAndExecution(t *testing.T) {
	newDirtyBatch := func() (*branchSyncPushFixture, *branchSyncCurrentPullUI, *branchStackUpdateFlow, stackUpdateRawSnapshot) {
		f, ui, batch := stackUpdateFixture(t, []string{"lower", "bottom"}, false)
		if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte{'s', 0x00, 0xff}, 0o600); err != nil {
			t.Fatal(err)
		}
		branchSyncPushGit(t, f.root, "add", "tracked")
		if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte{'u', 0x00, 0xfe}, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, "stack-untracked"), []byte{0x00, 0xff, 'u'}, 0o600); err != nil {
			t.Fatal(err)
		}
		branchSyncPushGit(t, f.root, "config", "branch.unrelated.description", "must remain untouched")
		return f, ui, batch, stackUpdateSnapshot(t, f)
	}
	f, ui, batch, before := newDirtyBatch()
	collectStackUpdate(t, f, ui, batch, func() {
		for _, member := range batch.members {
			if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/"+member.Name); got != member.OID {
				t.Errorf("%s moved during collection", member.Name)
			}
		}
		stackUpdateAssertWorktreeSnapshot(t, f, before)
	})
	if len(batch.plans) != 2 {
		t.Fatalf("plans=%d", len(batch.plans))
	}
	if branchSyncPushGit(t, f.root, "symbolic-ref", "--short", "HEAD") != "main" {
		t.Fatal("off-branch batch changed current branch")
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.update-stack.cancel").OnPress(); err != nil {
		t.Fatal(err)
	}
	for _, member := range batch.members {
		if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/"+member.Name); got != member.OID {
			t.Fatalf("%s moved before approval: %s", member.Name, got)
		}
	}
	stackUpdateAssertWorktreeSnapshot(t, f, before)
	// Cancellation is terminal for that displayed approval. Use a fresh actual
	// review/collection to exercise execution with a separate approval.
	f, ui, batch, before = newDirtyBatch()
	collectStackUpdate(t, f, ui, batch, func() { stackUpdateAssertWorktreeSnapshot(t, f, before) })
	if err := stackUpdateApprove(t, f); err != nil {
		t.Fatal(err)
	}
	for _, member := range batch.members {
		if f.ui.state.GetItemOperation(syncOperationItem(f.git, member.Name)) != types.ItemOperationPulling {
			t.Fatalf("%s not busy before dispatch", member.Name)
		}
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Fatalf("HEAD changed to %s", got)
	}
	stackUpdateAssertWorktreeSnapshot(t, f, before)
	for i, member := range batch.members {
		if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/"+member.Name); got != batch.plans[i].targetOID() {
			t.Fatalf("%s tip=%s want exact approved %s", member.Name, got, batch.plans[i].targetOID())
		}
		if f.ui.state.GetItemOperation(syncOperationItem(f.git, member.Name)) != types.ItemOperationNone {
			t.Fatalf("%s remained busy", member.Name)
		}
	}
}

func TestBranchSyncStackUpdateCurrentBranchRunsAfterOffBranchRefPhase(t *testing.T) {
	f, ui, batch := stackUpdateFixture(t, []string{"lower"}, true)
	collectStackUpdate(t, f, ui, batch)
	if batch.plans[0].current == nil || batch.plans[1].forward == nil {
		t.Fatalf("review order not current then off: %+v", batch.plans)
	}
	if err := stackUpdateApprove(t, f); err != nil {
		t.Fatal(err)
	}
	currentItem := syncOperationItem(f.git, "main")
	if f.ui.state.GetItemOperation(currentItem) != types.ItemOperationPulling {
		t.Fatal("current branch not held before dispatch")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got == batch.members[1].OID {
		t.Fatal("lower ref did not advance before current handoff")
	}
	if ui.subprocessCalls != 0 || len(ui.ui) != 1 || f.ui.state.GetItemOperation(currentItem) != types.ItemOperationNone {
		t.Fatal("current phase did not release its flag before UI handoff")
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 0 || len(ui.blockingWorkers) != 1 || f.ui.state.GetItemOperation(currentItem) != types.ItemOperationPulling {
		t.Fatal("UI handoff did not claim current flag and queue native validation")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 0 || len(ui.ui) != 1 || f.ui.state.GetItemOperation(currentItem) != types.ItemOperationPulling {
		t.Fatal("native validation did not queue terminal work while holding current flag")
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 1 || f.ui.state.GetItemOperation(currentItem) != types.ItemOperationPulling {
		t.Fatal("terminal did not retain current busy flag")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if f.ui.state.GetItemOperation(currentItem) != types.ItemOperationNone {
		t.Fatal("current flag survived completion")
	}
}

func TestBranchSyncStackUpdatePreflightRejectsWholeBatchOnStaleInput(t *testing.T) {
	for _, change := range []string{"source", "occupied-worktree"} {
		t.Run(change, func(t *testing.T) {
			f, ui, batch := stackUpdateFixture(t, []string{"lower", "bottom"}, false)
			collectStackUpdate(t, f, ui, batch)
			beforeBottom := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/bottom")
			beforeLower := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower")
			if change == "source" {
				branchSyncPushGit(t, f.root, "checkout", "-q", "lower")
				branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "moved upstream")
				branchSyncPushGit(t, f.root, "checkout", "-q", "main")
			} else {
				worktree := filepath.Join(t.TempDir(), "linked")
				branchSyncPushGit(t, f.root, "worktree", "add", "-q", worktree, "lower")
			}
			refsAfterExternalMutation := stackUpdateAllRefs(t, f)
			filesAfterExternalMutation := stackUpdateSnapshot(t, f)
			if err := stackUpdateApprove(t, f); err != nil {
				t.Fatal(err)
			}
			err := ui.runBlockingWorker(t)
			if err == nil {
				t.Fatal("stale batch preflight succeeded")
			}
			if got := stackUpdateAllRefs(t, f); got != refsAfterExternalMutation {
				t.Fatalf("preflight changed refs\nbefore: %s\nafter: %s", refsAfterExternalMutation, got)
			}
			stackUpdateAssertWorktreeSnapshot(t, f, filesAfterExternalMutation)
			if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/bottom"); got != beforeBottom {
				t.Fatal("preflight moved bottom")
			}
			if change == "source" {
				if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got == beforeLower {
					t.Fatal("external source mutation absent")
				}
			} else if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != beforeLower {
				t.Fatal("preflight moved occupied lower branch")
			}
			if len(ui.blockingWorkers) != 0 || len(ui.ui) != 0 {
				t.Fatal("preflight automatically retried or refetched")
			}
		})
	}
}

func TestBranchSyncStackUpdateNativeLockFailureKeepsPriorProgress(t *testing.T) {
	f, ui, batch := stackUpdateFixture(t, []string{"first", "second", "third"}, false)
	collectStackUpdate(t, f, ui, batch)
	for _, plan := range batch.plans {
		if err := plan.check(batch.target); err != nil {
			t.Fatalf("read-only preflight rejected plan before lock: %v", err)
		}
	}
	lock := filepath.Join(f.root, ".git", "refs", "heads", "second.lock")
	if err := os.WriteFile(lock, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	lockBefore, err := os.ReadFile(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := stackUpdateApprove(t, f); err != nil {
		t.Fatal(err)
	}
	err = ui.runBlockingWorker(t)
	if err == nil || !strings.Contains(err.Error(), "first") || !strings.Contains(err.Error(), "second") {
		t.Fatalf("partial failure error=%v", err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/first"); got != batch.plans[0].targetOID() {
		t.Fatalf("first ref=%s want committed approved target %s", got, batch.plans[0].targetOID())
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/second"); got != batch.members[1].OID {
		t.Fatal("failed ref moved")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/third"); got != batch.members[2].OID {
		t.Fatal("third ref attempted after failure")
	}
	if lockAfter, readErr := os.ReadFile(lock); readErr != nil || !bytes.Equal(lockAfter, lockBefore) {
		t.Fatalf("native failure altered lock file: %v", readErr)
	}
	if replay := stackUpdateApprove(t, f); replay == nil {
		t.Fatal("consumed approval replayed")
	}
	for _, member := range batch.members {
		if f.ui.state.GetItemOperation(syncOperationItem(f.git, member.Name)) != types.ItemOperationNone {
			t.Errorf("%s busy after error", member.Name)
		}
	}
}

func TestBranchSyncStackUpdateCurrentNativeValidationReportsEarlierProgress(t *testing.T) {
	f, ui, batch := stackUpdateFixture(t, []string{"lower"}, true)
	collectStackUpdate(t, f, ui, batch)
	if err := stackUpdateApprove(t, f); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	// The native preparation worker runs after the lower ref has committed;
	// moving the current target makes the reviewed integration stale.
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "move current tip after approval")
	err := ui.runBlockingWorker(t)
	if err == nil || !errors.Is(err, git_commands.ErrBranchSyncChanged) {
		t.Fatalf("native current validation error=%v", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("stale current plan launched terminal work")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got == batch.members[1].OID {
		t.Fatal("earlier lower progress was lost")
	}
	if !strings.Contains(err.Error(), "lower") {
		t.Fatalf("error omitted prior progress: %v", err)
	}
	if len(ui.blockingWorkers) != 0 || len(ui.ui) != 0 {
		t.Fatal("current native validation failure queued automatic reinspection/refetch")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("current branch remained busy")
	}
}
