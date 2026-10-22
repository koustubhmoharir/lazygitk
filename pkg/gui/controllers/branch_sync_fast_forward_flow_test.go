package controllers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

func newBranchSyncFastForwardFixture(t *testing.T) (*branchSyncPushFixture, *branchSyncCurrentPullUI, *BranchesController) {
	t.Helper()
	f, ui := newBranchSyncCurrentPullFixture(t)
	return f, ui, &BranchesController{c: f.sync.c}
}

func branchSyncFastForwardLocalUpstream(t *testing.T, f *branchSyncPushFixture, name, target string) {
	t.Helper()
	branchSyncPushGit(t, f.root, "config", "branch."+name+".remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch."+name+".merge", "refs/heads/"+target)
}

func branchSyncFastForwardCapture(t *testing.T, f *branchSyncPushFixture, ui *branchSyncCurrentPullUI, controller *BranchesController, branches []*models.Branch) error {
	t.Helper()
	if err := controller.fastForward(branches); err != nil {
		return err
	}
	if len(f.ui.workers) == 0 {
		t.Fatal("fast-forward did not queue inspection")
	}
	if err := f.ui.runWorker(t); err != nil {
		return err
	}
	if err := f.ui.runUI(t); err != nil {
		return err
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("inspection started a current-branch subprocess")
	}
	return nil
}

func TestBranchSyncFastForwardFlowSingleOffBranchUsesCapturedNameAndPreservesRawState(t *testing.T) {
	f, _, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "branch", "lower", "main")
	branchSyncFastForwardLocalUpstream(t, f, "lower", "topic")
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "newer local upstream")
	targetOID := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	model := &models.Branch{Name: "lower", AheadForPull: "0", BehindForPull: "0"}
	if err := controller.fastForward([]*models.Branch{model}); err != nil {
		t.Fatal(err)
	}
	model.Name = "topic"
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 || f.ui.menus[0].Title != f.sync.c.Tr.BranchSyncFastForwardMenu {
		t.Fatalf("expected fast-forward-only review for captured lower: menus=%+v", f.ui.menus)
	}
	menu := f.ui.menus[0]
	if !strings.Contains(menu.Prompt, "lower") {
		t.Fatalf("review did not retain captured branch name: %q", menu.Prompt)
	}
	beforeHead := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged bytes\x00\xff"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged bytes\x00\xfe"), 0o600); err != nil {
		t.Fatal(err)
	}
	beforeIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	beforeWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, menu, "branch.update-selected.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("expected direct local update review, got %d", len(f.ui.confirms))
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, "lower") {
		t.Fatalf("confirmation lost captured name: %q", f.ui.confirms[0].Prompt)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != targetOID {
		t.Fatalf("lower tip %s, want %s", got, targetOID)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != targetOID {
		t.Fatalf("upstream moved unexpectedly: %s", got)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != beforeHead {
		t.Fatalf("HEAD changed: %s", got)
	}
	afterIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	afterWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeIndex, afterIndex) || !bytes.Equal(beforeWork, afterWork) {
		t.Fatal("fast-forward changed dirty index/worktree bytes")
	}
}

func TestBranchSyncFastForwardFlowCurrentBranchUsesFFOnlyWithoutAutostashOrRebase(t *testing.T) {
	f, ui, controller := newBranchSyncFastForwardFixture(t)
	branchSyncFastForwardLocalUpstream(t, f, "main", "incoming")
	branchSyncPushGit(t, f.root, "config", "branch.main.rebase", "true")
	branchSyncPushGit(t, f.root, "config", "rebase.autoStash", "true")
	originalTopic := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "checkout", "-qb", "incoming", "main")
	if err := os.WriteFile(filepath.Join(f.root, "incoming-new"), []byte("new upstream file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "incoming-new")
	branchSyncPushGit(t, f.root, "commit", "-qm", "incoming file")
	incomingOID := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "main")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged\x00\xff"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged\x00\xfe"), 0o600); err != nil {
		t.Fatal(err)
	}
	untrackedBytes := []byte("untracked\x00\xfd")
	if err := os.WriteFile(filepath.Join(f.root, "fast-forward-untracked"), untrackedBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	configBefore := branchSyncPushGit(t, f.root, "config", "--local", "--list", "--null")
	stagedTrackedOID := branchSyncPushGit(t, f.root, "rev-parse", ":tracked")
	stageBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	workBefore, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if err := branchSyncFastForwardCapture(t, f, ui, controller, []*models.Branch{{Name: "main"}}); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 || f.ui.menus[0].Title != f.sync.c.Tr.BranchSyncFastForwardMenu {
		t.Fatal("current path did not begin with the fast-forward-only menu")
	}
	if err := menuItem(t, f.ui.menus[0], "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("local upstream unexpectedly requested network fetch approval")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 {
		t.Fatalf("ff-only flow offered integration menu: %d", len(f.ui.menus))
	}
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, f.sync.c.Tr.BranchSyncIntegrationFFOnly) {
		t.Fatal("missing final fast-forward-only confirmation")
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatalf("native fast-forward validation: %v", err)
	}
	indexBeforeTerminal, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stageBefore, indexBeforeTerminal) {
		t.Fatal("index changed before terminal execution")
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatalf("native fast-forward completion: %v", err)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("current fast-forward helper calls=%d", ui.subprocessCalls)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != incomingOID {
		t.Fatalf("HEAD %s, want reviewed incoming commit %s", got, incomingOID)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != originalTopic {
		t.Fatal("unrelated topic ref changed")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/incoming"); got != incomingOID {
		t.Fatal("reviewed incoming branch ref changed")
	}
	if got := branchSyncPushGit(t, f.root, "config", "--local", "--list", "--null"); got != configBefore {
		t.Fatal("fast-forward changed saved configuration")
	}
	afterIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	afterWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(stageBefore, afterIndex) {
		t.Fatal("index did not acquire the incoming file entry")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", ":tracked"); got != stagedTrackedOID {
		t.Fatalf("staged tracked blob %s, want %s", got, stagedTrackedOID)
	}
	if got, want := branchSyncPushGit(t, f.root, "rev-parse", ":incoming-new"), branchSyncPushGit(t, f.root, "rev-parse", incomingOID+":incoming-new"); got != want {
		t.Fatalf("incoming file index blob %s, want %s", got, want)
	}
	if !bytes.Equal(workBefore, afterWork) {
		t.Fatal("current fast-forward altered unstaged tracked bytes")
	}
	if got, readErr := os.ReadFile(filepath.Join(f.root, "fast-forward-untracked")); readErr != nil || !bytes.Equal(got, untrackedBytes) {
		t.Fatalf("untracked bytes changed: %q, err=%v", got, readErr)
	}
	if got := branchSyncPushGit(t, f.root, "stash", "list", "--format=%gd"); got != "" {
		t.Fatalf("autostash created stash: %s", got)
	}
}

func TestBranchSyncFastForwardFlowOffBranchDivergenceOffersOnlyCancel(t *testing.T) {
	f, _, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "main sibling commit")
	branchSyncFastForwardLocalUpstream(t, f, "topic", "main")
	head := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	topic := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	main := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	index, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	work, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.fastForward([]*models.Branch{{Name: "topic"}}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 || f.ui.menus[0].Title != f.sync.c.Tr.BranchSyncFastForwardMenu {
		t.Fatal("missing ff-only divergence action menu")
	}
	if err := menuItem(t, f.ui.menus[0], "branch.update-selected.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 || len(f.ui.menus) != 2 || len(f.ui.menus[1].Items) != 1 || f.ui.menus[1].Items[0].Verb != "cancel" {
		t.Fatal("divergence did not produce a cancel-only explanation")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != head {
		t.Fatal("HEAD moved")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != topic {
		t.Fatal("divergent branch rewound or moved")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main"); got != main {
		t.Fatal("divergent upstream moved")
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	workAfter, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(index, indexAfter) || !bytes.Equal(work, workAfter) {
		t.Fatal("divergence changed current index/worktree")
	}
}

func TestBranchSyncFastForwardFlowOccupiedWorktreeBlocksAdvanceWithoutMutation(t *testing.T) {
	f, _, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "branch", "lower", "main")
	branchSyncFastForwardLocalUpstream(t, f, "lower", "topic")
	baseOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower")
	targetOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	linked := filepath.Join(t.TempDir(), "occupied-lower")
	branchSyncPushGit(t, f.root, "worktree", "add", "-q", linked, "lower")
	rootIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	linkedIndexPath := branchSyncPushGit(t, linked, "rev-parse", "--git-path", "index")
	if !filepath.IsAbs(linkedIndexPath) {
		linkedIndexPath = filepath.Join(linked, linkedIndexPath)
	}
	linkedIndex, err := os.ReadFile(linkedIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	linkedTracked, err := os.ReadFile(filepath.Join(linked, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.fastForward([]*models.Branch{{Name: "lower"}}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	menu := f.ui.menus[0]
	if menu.Title != f.sync.c.Tr.BranchSyncFastForwardMenu {
		t.Fatalf("menu title %q", menu.Title)
	}
	if err := menuItem(t, menu, "branch.update-selected.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	advanceErr := f.ui.runWorker(t)
	if advanceErr == nil || !strings.Contains(advanceErr.Error(), "reserved by another worktree") {
		t.Fatalf("occupied branch was not blocked during local update preparation: %v", advanceErr)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("occupied branch reached update approval")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/lower"); got != baseOID {
		t.Fatalf("occupied lower moved to %s", got)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != targetOID {
		t.Fatal("upstream ref changed")
	}
	if got := branchSyncPushGit(t, linked, "rev-parse", "HEAD"); got != baseOID {
		t.Fatal("occupied worktree HEAD changed")
	}
	rootIndexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	linkedIndexAfter, err := os.ReadFile(linkedIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	linkedTrackedAfter, err := os.ReadFile(filepath.Join(linked, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rootIndex, rootIndexAfter) {
		t.Fatal("current worktree index changed")
	}
	if !bytes.Equal(linkedIndex, linkedIndexAfter) || !bytes.Equal(linkedTracked, linkedTrackedAfter) {
		t.Fatal("occupied worktree index or files changed")
	}
}

func TestBranchSyncFastForwardFlowMultiCaptureAndReadOnlyFirstMemberReview(t *testing.T) {
	f, _, controller := newBranchSyncFastForwardFixture(t)
	branchSyncPushGit(t, f.root, "branch", "bottom", "main")
	branchSyncFastForwardLocalUpstream(t, f, "main", "topic")
	branchSyncFastForwardLocalUpstream(t, f, "bottom", "topic")
	branchSyncPushGit(t, f.root, "checkout", "topic")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-qm", "upstream")
	branchSyncPushGit(t, f.root, "checkout", "main")
	mainOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	bottomOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/bottom")
	topicOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	names := []string{"main", "bottom"}
	if err := controller.fastForwardCapturedBranches(names); err != nil {
		t.Fatal(err)
	}
	names[0] = "topic"
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.workers) != 1 {
		t.Fatalf("expected queued first-member inspection, workers=%d", len(f.ui.workers))
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 || f.ui.menus[0].Title != f.sync.c.Tr.BranchSyncFastForwardMenu || !strings.Contains(f.ui.menus[0].Prompt, "main") || f.ui.menus[0].Items[0].CommandID != "branch.pull-current.fetch" {
		t.Fatalf("first member not captured: %+v", f.ui.menus)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("batch approved before member review")
	}
	if err := menuItem(t, f.ui.menus[0], "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatal("missing first member native review")
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
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 2 || !strings.Contains(f.ui.menus[1].Prompt, "bottom") {
		t.Fatalf("second captured member not reviewed: %+v", f.ui.menus)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main"); got != mainOID {
		t.Fatal("collection moved current ref before final batch approval")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/bottom"); got != bottomOID {
		t.Fatal("collection moved off-branch ref before final batch approval")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != topicOID {
		t.Fatal("collection moved upstream ref")
	}
	for _, bad := range [][]string{{"lower", "lower"}, {"lower", "bad name"}, {"lower", "-x"}} {
		before := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)")
		if err := controller.fastForwardCapturedBranches(bad); err != nil {
			t.Fatalf("capture %q: %v", bad, err)
		}
		if err := f.ui.runWorker(t); err == nil {
			t.Fatalf("worker accepted invalid names %q", bad)
		}
		if got := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)"); got != before {
			t.Fatal("invalid names changed refs")
		}
	}
}

func TestBranchSyncFastForwardFlowGenerationChangeBeforeInspectionBlocksUI(t *testing.T) {
	f, _, controller := newBranchSyncFastForwardFixture(t)
	if err := controller.fastForward([]*models.Branch{{Name: "topic"}}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation++
	if err := f.ui.runUI(t); err == nil {
		t.Fatal("stale queued inspection callback succeeded")
	}
	if len(f.ui.ui) != 0 || len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 || f.ui.refreshes != 0 {
		t.Fatal("stale generation prompted, refreshed, or queued UI")
	}

	f, _, controller = newBranchSyncFastForwardFixture(t)
	if err := controller.fastForwardCapturedBranches([]string{"topic", "also-topic"}); err != nil {
		t.Fatal(err)
	}
	f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(f.root)}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("multi-member worker accepted replaced Git owner")
	}
	if len(f.ui.ui) != 0 || len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 || f.ui.refreshes != 0 {
		t.Fatal("stale multi-member worker prompted, refreshed, or queued UI")
	}
}
