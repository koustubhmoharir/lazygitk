package controllers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchSyncUpdateSelectedBranchFetchAndFastForward(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged and dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged current worktree"), 0o600); err != nil {
		t.Fatal(err)
	}
	indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	headBefore := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 {
		t.Fatalf("menus=%d", len(f.ui.menus))
	}
	if err := menuItem(t, f.ui.menus[0], "branch.update-selected.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("fetch ran before confirmation")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("fetch confirmations=%d", len(f.ui.confirms))
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, "refs/cache/review/topic") {
		t.Fatalf("custom tracking namespace missing: %s", f.ui.confirms[0].Prompt)
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
	if len(f.ui.confirms) != 2 {
		t.Fatalf("update confirmations=%d", len(f.ui.confirms))
	}
	branchTipBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	if err := f.ui.confirms[1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	if got == branchTipBefore {
		t.Fatal("selected branch did not advance")
	}
	if got != branchSyncPushGit(t, f.root, "rev-parse", "refs/cache/review/topic") {
		t.Fatal("selected branch did not reach reviewed tracking commit")
	}
	if branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD") != "refs/heads/main" {
		t.Fatal("HEAD moved")
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Fatal("index bytes changed")
	}
	worktreeAfter, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if string(worktreeAfter) != "unstaged current worktree" {
		t.Fatalf("worktree changed: %q", worktreeAfter)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != headBefore {
		t.Fatalf("HEAD changed from %s to %s", headBefore, got)
	}
}

func TestBranchSyncUpdateRefusesCurrentBranch(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	err := f.sync.updateBranchSource(branchSyncSource{name: "main", selected: true})
	if err == nil {
		err = f.ui.runWorker(t)
	}
	if err == nil {
		err = f.ui.runUI(t)
	}
	if err == nil || !strings.Contains(err.Error(), "checked out") {
		t.Fatalf("got %v", err)
	}
	if len(f.ui.menus) != 0 {
		t.Fatal("opened a menu for the current branch")
	}
}

func TestBranchSyncUpdateLocalUpstreamSkipsTransport(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "branch", "local-upstream", "topic")
	branchSyncPushGit(t, f.root, "checkout", "-q", "local-upstream")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "local upstream update")
	branchSyncPushGit(t, f.root, "checkout", "-q", f.current)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/local-upstream")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
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
	if len(f.ui.confirms) != 1 {
		t.Fatalf("expected direct update review; confirmations=%d", len(f.ui.confirms))
	}
	if strings.Contains(f.ui.confirms[0].Prompt, "Fetch remote") {
		t.Fatal("local upstream used network fetch review")
	}
}

func TestBranchSyncUpdateRejectsRepositoryChangeInQueuedCallbacks(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation++
	if err := f.ui.runWorker(t); err == nil || !strings.Contains(err.Error(), "repository changed") {
		t.Fatalf("got %v", err)
	}
	if len(f.ui.ui) != 0 {
		t.Fatal("stale worker queued a UI callback")
	}
}

func TestBranchSyncUpdateQueuedCallbackAndConfirmationRejectRepositoryChange(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation++
	if err := f.ui.runUI(t); err == nil || !strings.Contains(err.Error(), "repository changed") {
		t.Fatalf("queued callback returned %v", err)
	}
	if len(f.ui.menus) != 0 {
		t.Fatal("stale callback opened a menu")
	}

	// Recreate the captured target and prepare the fetch confirmation, then
	// change repositories before the user confirms it.
	f = newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
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
	f.ui.state.generation++
	if err := f.ui.confirms[0].HandleConfirm(); err == nil || !strings.Contains(err.Error(), "repository changed") {
		t.Fatalf("confirmation returned %v", err)
	}
	if len(f.ui.workers) != 0 {
		t.Fatal("stale confirmation queued a fetch worker")
	}
}

func TestBranchSyncUpdateNoopOffersOnlyCancel(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "push", "origin", "refs/heads/topic:refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
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
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("no-op unexpectedly requested update confirmation: %d", len(f.ui.confirms))
	}
	noop := f.ui.menus[len(f.ui.menus)-1]
	if len(noop.Items) != 1 || noop.Items[0].Verb != "cancel" {
		t.Fatalf("no-op menu has actions: %#v", noop.Items)
	}
	if branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic") != f.sourceOID {
		t.Fatal("no-op changed the branch")
	}
}

func TestBranchSyncUpdateStaleFetchSourceDoesNotFetch(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, f.ui.menus[0], "branch.update-selected.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "move selected branch")
	branchSyncPushGit(t, f.root, "checkout", "-q", f.current)
	if err := f.ui.runWorker(t); err != nil {
		t.Fatalf("stale preparation should return to UI for fresh review: %v", err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "FETCH_HEAD")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("stale source reached fetch confirmation")
	}
}

func TestBranchSyncUpdateFetchCancellationKeepsLocalBranch(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
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
	localBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	if err := f.ui.confirms[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != localBefore {
		t.Fatal("cancel before fetch changed the local branch")
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "FETCH_HEAD")); !os.IsNotExist(err) {
		t.Fatalf("FETCH_HEAD exists or could not be checked: %v", err)
	}
}

func TestBranchSyncUpdateCancelAfterFetchDoesNotAdvanceBranch(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
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
	localBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 2 {
		t.Fatalf("expected separate update review, got %d confirmations", len(f.ui.confirms))
	}
	trackingAfter := branchSyncPushGit(t, f.root, "rev-parse", "refs/cache/review/topic")
	if trackingAfter == localBefore {
		t.Fatal("fetch did not update the chosen tracking ref")
	}
	if err := f.ui.confirms[1].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != localBefore {
		t.Fatal("cancel after fetch advanced the local branch")
	}
}

func TestBranchSyncUpdateBranchMoveBeforeConfirmIsRefused(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f, "topic")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
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
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", "refs/heads/main")
	if err := f.ui.confirms[1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") {
		t.Fatal("stale confirmation changed the moved branch")
	}
	if len(f.ui.ui) == 0 {
		t.Fatal("stale update did not queue a fresh review")
	}
}

func seedBranchSyncUpdateRemote(t *testing.T, f *branchSyncPushFixture, branch string) {
	t.Helper()
	branchSyncPushGit(t, f.root, "checkout", "-q", branch)
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "remote update")
	branchSyncPushGit(t, f.root, "push", "origin", "HEAD:refs/heads/"+branch)
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/"+branch, f.sourceOID)
	branchSyncPushGit(t, f.root, "checkout", "-q", f.current)
}
