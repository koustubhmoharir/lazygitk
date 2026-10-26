package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// branchSyncFastForwardUI records the end-of-run alert. The pull fixture's UI
// is wrapped, not changed, so every other fake behaviour is shared.
type branchSyncFastForwardUI struct {
	*branchSyncCurrentPullUI
	alerts []branchSyncFastForwardAlert
}

type branchSyncFastForwardAlert struct{ title, message string }

func (ui *branchSyncFastForwardUI) Alert(title, message string) {
	ui.alerts = append(ui.alerts, branchSyncFastForwardAlert{title: title, message: message})
}

func newBranchSyncFastForwardFixture(t *testing.T) (*branchSyncPushFixture, *branchSyncFastForwardUI, *BranchesController) {
	t.Helper()
	f, pull := newBranchSyncCurrentPullFixture(t)
	ui := &branchSyncFastForwardUI{branchSyncCurrentPullUI: pull}
	f.sync.c.HelperCommon.IGuiCommon = ui
	return f, ui, &BranchesController{c: f.sync.c}
}

// branchSyncFastForwardLocalUpstream makes name track the local topic branch.
func branchSyncFastForwardLocalUpstream(t *testing.T, f *branchSyncPushFixture, name string) {
	t.Helper()
	branchSyncPushGit(t, f.root, "config", "branch."+name+".remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch."+name+".merge", "refs/heads/topic")
}

// branchSyncFastForwardDrain runs queued work until the flow is waiting for a
// user decision (a menu, which is recorded rather than queued).
func branchSyncFastForwardDrain(t *testing.T, f *branchSyncPushFixture) {
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

func branchSyncFastForwardStart(t *testing.T, f *branchSyncPushFixture, controller *BranchesController, names ...string) {
	t.Helper()
	branches := make([]*models.Branch, 0, len(names))
	for _, name := range names {
		branches = append(branches, &models.Branch{Name: name})
	}
	if err := controller.fastForward(branches); err != nil {
		t.Fatal(err)
	}
	branchSyncFastForwardDrain(t, f)
}

func branchSyncFastForwardAssertSilent(t *testing.T, f *branchSyncPushFixture, ui *branchSyncFastForwardUI) {
	t.Helper()
	if len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 || len(f.ui.prompts) != 0 || len(ui.alerts) != 0 {
		t.Fatalf("fast-forward prompted or reported: menus=%d confirms=%d prompts=%d alerts=%+v", len(f.ui.menus), len(f.ui.confirms), len(f.ui.prompts), ui.alerts)
	}
	if ui.subprocessCalls != 0 {
		t.Fatalf("fast-forward started a terminal subprocess: %d", ui.subprocessCalls)
	}
}

// branchSyncFastForwardSingleAlert returns the lines of the only alert.
func branchSyncFastForwardSingleAlert(t *testing.T, f *branchSyncPushFixture, ui *branchSyncFastForwardUI) []string {
	t.Helper()
	if len(ui.alerts) != 1 {
		t.Fatalf("alerts=%+v, want exactly one", ui.alerts)
	}
	if len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 {
		t.Fatalf("skipped branches opened menus=%d confirms=%d", len(f.ui.menus), len(f.ui.confirms))
	}
	if ui.alerts[0].title != f.sync.c.Tr.BranchSyncFastForwardSkippedTitle {
		t.Fatalf("alert title %q", ui.alerts[0].title)
	}
	return strings.Split(ui.alerts[0].message, "\n")
}

func branchSyncFastForwardWorktreeBytes(t *testing.T, root string) ([]byte, []byte) {
	t.Helper()
	index, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	tracked, err := os.ReadFile(filepath.Join(root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	return index, tracked
}

func TestBranchSyncFastForwardOffBranchBehindMovesWithoutPrompt(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "branch", "lower", "main")
	branchSyncFastForwardLocalUpstream(t, f, "lower")
	topic := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	head := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged bytes\x00\xff"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged bytes\x00\xfe"), 0o600); err != nil {
		t.Fatal(err)
	}
	indexBefore, trackedBefore := branchSyncFastForwardWorktreeBytes(t, f.root)

	branchSyncFastForwardStart(t, f, controller, "lower")

	branchSyncFastForwardAssertSilent(t, f, ui)
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != topic {
		t.Fatalf("lower tip %s, want upstream %s", got, topic)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD changed to %s", got)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != topic {
		t.Fatalf("upstream moved to %s", got)
	}
	indexAfter, trackedAfter := branchSyncFastForwardWorktreeBytes(t, f.root)
	if !bytes.Equal(indexBefore, indexAfter) || !bytes.Equal(trackedBefore, trackedAfter) {
		t.Fatal("fast-forward changed staged or unstaged bytes")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "lower")) != types.ItemOperationNone {
		t.Fatal("fast-forward left the branch busy")
	}
}

func TestBranchSyncFastForwardCurrentBehindFastForwardsInProcessWithoutPrompt(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncFastForwardLocalUpstream(t, f, "main")
	topic := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")

	branchSyncFastForwardStart(t, f, controller, "main")

	branchSyncFastForwardAssertSilent(t, f, ui)
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != topic {
		t.Fatalf("HEAD %s, want upstream %s", got, topic)
	}
	if got := branchSyncPushGit(t, f.root, "status", "--porcelain"); got != "" {
		t.Fatalf("fast-forward left the worktree dirty: %q", got)
	}
}

func TestBranchSyncFastForwardUpToDateAndAheadAreSilent(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "branch", "equal", "topic")
	branchSyncFastForwardLocalUpstream(t, f, "equal")
	branchSyncPushGit(t, f.root, "branch", "ahead", "topic")
	branchSyncPushGit(t, f.root, "checkout", "-q", "ahead")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "ahead of upstream")
	ahead := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	branchSyncFastForwardLocalUpstream(t, f, "ahead")
	equal := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/equal")

	branchSyncFastForwardStart(t, f, controller, "equal", "ahead")

	branchSyncFastForwardAssertSilent(t, f, ui)
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/equal"); got != equal {
		t.Fatal("up-to-date branch moved")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/ahead"); got != ahead {
		t.Fatal("ahead branch was rewound")
	}
}

func TestBranchSyncFastForwardSkippedBranchesShareOneAlert(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "checkout", "-q", "-b", "diverged", "main")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "diverged commit")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	branchSyncFastForwardLocalUpstream(t, f, "diverged")
	branchSyncPushGit(t, f.root, "branch", "plain", "main")
	branchSyncPushGit(t, f.root, "branch", "orphan", "main")
	branchSyncPushGit(t, f.root, "config", "branch.orphan.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.orphan.merge", "refs/heads/gone")
	before := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)")

	branchSyncFastForwardStart(t, f, controller, "diverged", "plain", "orphan")

	lines := branchSyncFastForwardSingleAlert(t, f, ui)
	want := []string{
		"diverged: " + fmt.Sprintf(f.sync.c.Tr.BranchSyncFastForwardDiverged, "refs/heads/topic"),
		"plain: " + f.sync.c.Tr.BranchSyncFastForwardNoUpstream,
		"orphan: " + fmt.Sprintf(f.sync.c.Tr.BranchSyncFastForwardMissingRef, "refs/remotes/origin/gone"),
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("alert lines %q, want %q", lines, want)
	}
	if got := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)"); got != before {
		t.Fatal("skipped branches changed refs")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "plain")) != types.ItemOperationNone {
		t.Fatal("skipped branch remained busy")
	}
}

func TestBranchSyncFastForwardOccupiedWorktreeIsListedWithoutMutation(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "branch", "lower", "main")
	branchSyncFastForwardLocalUpstream(t, f, "lower")
	base := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower")
	linked := filepath.Join(t.TempDir(), "occupied-lower")
	branchSyncPushGit(t, f.root, "worktree", "add", "-q", linked, "lower")

	branchSyncFastForwardStart(t, f, controller, "lower")

	lines := branchSyncFastForwardSingleAlert(t, f, ui)
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "lower: ") || !strings.Contains(lines[0], "reserved by another worktree") {
		t.Fatalf("occupied branch alert %q", lines)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != base {
		t.Fatalf("occupied lower moved to %s", got)
	}
	if got := branchSyncPushGit(t, linked, "rev-parse", "HEAD"); got != base {
		t.Fatal("occupied worktree HEAD changed")
	}
}

func TestBranchSyncFastForwardMixedSelectionMovesBehindAndListsRest(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "branch", "bottom", "main")
	branchSyncFastForwardLocalUpstream(t, f, "bottom")
	branchSyncFastForwardLocalUpstream(t, f, "main")
	branchSyncPushGit(t, f.root, "branch", "equal", "topic")
	branchSyncFastForwardLocalUpstream(t, f, "equal")
	branchSyncPushGit(t, f.root, "branch", "plain", "main")
	topic := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")

	// The checked-out branch is selected first on purpose: order is decided by
	// the flow, not by selection order.
	branchSyncFastForwardStart(t, f, controller, "main", "plain", "equal", "bottom")

	lines := branchSyncFastForwardSingleAlert(t, f, ui)
	if len(lines) != 1 || lines[0] != "plain: "+f.sync.c.Tr.BranchSyncFastForwardNoUpstream {
		t.Fatalf("summary %q", lines)
	}
	for _, ref := range []string{"HEAD", "refs/heads/bottom", "refs/heads/main", "refs/heads/equal"} {
		if got := branchSyncPushGit(t, f.root, "rev-parse", ref); got != topic {
			t.Fatalf("%s at %s, want %s", ref, got, topic)
		}
	}
}

func TestBranchSyncFastForwardUntrackedFileInTheWayLeavesBranchAndShowsGitError(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(f.root, "clash"), []byte("from topic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "clash")
	branchSyncPushGit(t, f.root, "commit", "-qm", "topic clash")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(f.root, "clash"), []byte("untracked local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncFastForwardLocalUpstream(t, f, "main")
	head := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")

	branchSyncFastForwardStart(t, f, controller, "main")

	lines := branchSyncFastForwardSingleAlert(t, f, ui)
	// Git's multi-line message keeps its layout, indented under the branch.
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "main: ") || !strings.Contains(lines[0], "would be overwritten by merge") ||
		!slices.ContainsFunc(lines[1:], func(line string) bool { return strings.HasPrefix(line, "  ") && strings.Contains(line, "clash") }) {
		t.Fatalf("alert did not carry Git's error: %q", lines)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != head {
		t.Fatal("refused merge moved HEAD")
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "clash")); err != nil || string(got) != "untracked local\n" {
		t.Fatalf("untracked file changed: %q, err=%v", got, err)
	}
}

// branchSyncFastForwardIgnoredCollision makes the topic commit add a file that
// the checked-out main ignores locally, so Git would overwrite ignored content.
func branchSyncFastForwardIgnoredCollision(t *testing.T, f *branchSyncPushFixture) {
	t.Helper()
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(f.root, "collision"), []byte("from topic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "collision")
	branchSyncPushGit(t, f.root, "commit", "-qm", "topic collision")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(f.root, "collision"), []byte("ignored original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, ".git", "info", "exclude"), []byte("collision\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "check-ignore", "-q", "collision")
	branchSyncFastForwardLocalUpstream(t, f, "main")
}

func TestBranchSyncFastForwardIgnoredFileCancelRecordsNothing(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncFastForwardIgnoredCollision(t, f)
	head := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")

	branchSyncFastForwardStart(t, f, controller, "main")

	if len(f.ui.menus) != 1 || f.ui.menus[0].Title != f.sync.c.Tr.BranchSyncIntegrationIgnoredTitle {
		t.Fatalf("ignored-file approval menu missing: %+v", f.ui.menus)
	}
	if err := menuItem(t, f.ui.menus[0], "branch.pull-current.cancel-ignored-risk").OnPress(); err != nil {
		t.Fatal(err)
	}
	branchSyncFastForwardDrain(t, f)

	if len(f.ui.menus) != 1 || len(f.ui.confirms) != 0 || len(ui.alerts) != 0 {
		t.Fatalf("cancel prompted again or reported: menus=%d confirms=%d alerts=%+v", len(f.ui.menus), len(f.ui.confirms), ui.alerts)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != head {
		t.Fatal("cancelled ignored-file review moved HEAD")
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "collision")); err != nil || string(got) != "ignored original\n" {
		t.Fatalf("ignored file changed: %q, err=%v", got, err)
	}
}

func TestBranchSyncFastForwardIgnoredFileApprovalMovesCurrentBranch(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncFastForwardIgnoredCollision(t, f)
	topic := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")

	branchSyncFastForwardStart(t, f, controller, "main")

	if len(f.ui.menus) != 1 {
		t.Fatalf("approval menu missing: %+v", f.ui.menus)
	}
	if err := menuItem(t, f.ui.menus[0], "branch.pull-current.approve-ignored-risk").OnPress(); err != nil {
		t.Fatal(err)
	}
	branchSyncFastForwardDrain(t, f)

	if len(f.ui.menus) != 1 || len(f.ui.confirms) != 0 || len(ui.alerts) != 0 || ui.subprocessCalls != 0 {
		t.Fatalf("approved fast-forward prompted, reported or used a terminal: menus=%d confirms=%d alerts=%+v subprocess=%d", len(f.ui.menus), len(f.ui.confirms), ui.alerts, ui.subprocessCalls)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != topic {
		t.Fatalf("HEAD %s, want %s", got, topic)
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "collision")); err != nil || string(got) != "from topic\n" {
		t.Fatalf("approved overwrite did not apply: %q, err=%v", got, err)
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("approved fast-forward left the branch busy")
	}
}

func TestBranchSyncFastForwardDoesNotFetchFromRemote(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "push", "-q", "origin", "topic")
	tracking := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/topic")
	clone := filepath.Join(t.TempDir(), "remote-work")
	branchSyncPushGit(t, t.TempDir(), "clone", "-q", "--branch", "topic", f.remote, clone)
	branchSyncPushGit(t, clone, "config", "user.name", "Remote Test")
	branchSyncPushGit(t, clone, "config", "user.email", "remote@example.invalid")
	branchSyncPushGit(t, clone, "commit", "--allow-empty", "-qm", "newer remote commit")
	newer := branchSyncPushGit(t, clone, "rev-parse", "HEAD")
	branchSyncPushGit(t, clone, "push", "-q", "origin", "topic")
	branchSyncPushGit(t, f.root, "branch", "lower", "main")
	branchSyncPushGit(t, f.root, "config", "branch.lower.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.lower.merge", "refs/heads/topic")

	branchSyncFastForwardStart(t, f, controller, "lower")

	branchSyncFastForwardAssertSilent(t, f, ui)
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != tracking {
		t.Fatalf("lower at %s, want local tracking ref %s", got, tracking)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/topic"); got != tracking {
		t.Fatalf("tracking ref was fetched to %s", got)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got == newer {
		t.Fatal("remote commit reached the local branch")
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/topic"); got != newer {
		t.Fatalf("test remote did not receive its newer commit: %s", got)
	}
}

func TestBranchSyncFastForwardInvalidNamesNeverMutate(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	if err := controller.fastForwardCapturedBranches([]string{"lower", "lower"}); !errors.Is(err, git_commands.ErrBranchSyncChanged) {
		t.Fatalf("duplicate capture error %v", err)
	}
	before := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)")
	for _, names := range [][]string{{"lower", "bad name"}, {"lower", "-x"}} {
		if err := controller.fastForwardCapturedBranches(names); err != nil {
			t.Fatalf("capture %q: %v", names, err)
		}
		branchSyncFastForwardDrain(t, f)
		if len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 {
			t.Fatalf("invalid names %q prompted", names)
		}
		if got := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)"); got != before {
			t.Fatalf("invalid names %q changed refs", names)
		}
	}
	if len(ui.alerts) != 2 {
		t.Fatalf("invalid names were not each listed: %+v", ui.alerts)
	}
}

func TestBranchSyncFastForwardBusyBranchRejectedBeforeDispatch(t *testing.T) {
	f, _, controller := newBranchSyncFastForwardFixture(t)
	branchSyncFastForwardLocalUpstream(t, f, "main")
	f.ui.state.SetItemOperation(syncOperationItem(f.git, "main"), types.ItemOperationPulling)
	if err := controller.fastForward([]*models.Branch{{Name: "main"}}); err == nil {
		t.Fatal("busy branch was accepted")
	}
	if len(f.ui.workers) != 0 {
		t.Fatal("busy branch queued a worker")
	}
}

func TestBranchSyncFastForwardGenerationChangeBlocksUI(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	if err := controller.fastForward([]*models.Branch{{Name: "topic"}}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	refreshes := f.ui.refreshes
	f.ui.state.generation++
	if err := f.ui.runUI(t); err == nil {
		t.Fatal("stale completion callback succeeded")
	}
	if len(ui.alerts) != 0 || len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 || f.ui.refreshes != refreshes {
		t.Fatal("stale completion prompted, refreshed, or reported")
	}

	f, ui, controller = newBranchSyncFastForwardFixture(t)
	if err := controller.fastForwardCapturedBranches([]string{"topic", "also-topic"}); err != nil {
		t.Fatal(err)
	}
	f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(f.root)}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("multi-member worker accepted replaced Git owner")
	}
	if len(ui.alerts) != 0 || len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 || f.ui.refreshes != 0 {
		t.Fatal("stale multi-member worker prompted, refreshed, or reported")
	}
}
