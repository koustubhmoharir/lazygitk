package controllers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
)

func TestBranchSyncUpdateSelectedBranchFetchAndFastForward(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f)
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
	configureBranchSyncUpdateStorage(t, f)
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
	if got := f.ui.menus[0].Items[0].Label; got != f.sync.c.Tr.BranchSyncReviewLocalUpdate {
		t.Fatalf("local upstream action label %q", got)
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

func TestBranchSyncUpdateQueuedPreparationCapturesInputs(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	target := newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true})
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncUpdateFlow{target: target, info: info, options: branchSyncUpdateDefaults(info)}
	if err := flow.prepareFetch(); err != nil {
		t.Fatal(err)
	}
	flow.info.Name, flow.info.OID = "main", strings.Repeat("0", len(info.OID))
	flow.options = gitFetchOptionsForTest("invalid-remote", "refs/heads/changed", "refs/cache/changed")
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("confirmations=%d", len(f.ui.confirms))
	}
	prompt := f.ui.confirms[0].Prompt
	for _, expected := range []string{"origin", "refs/heads/topic", "refs/cache/review/topic", "topic"} {
		if !strings.Contains(prompt, expected) {
			t.Errorf("captured review missing %q: %s", expected, prompt)
		}
	}
}

func TestBranchSyncUpdateQueuedLocalPreparationCapturesInputs(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "branch", "local-upstream", "topic")
	branchSyncPushGit(t, f.root, "checkout", "-q", "local-upstream")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "local upstream commit")
	branchSyncPushGit(t, f.root, "checkout", "-q", f.current)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/local-upstream")
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}), info: info, options: branchSyncUpdateDefaults(info)}
	if err := flow.prepareLocalUpdate(); err != nil {
		t.Fatal(err)
	}
	flow.info.Name, flow.info.OID = "main", strings.Repeat("0", len(info.OID))
	flow.options.RemoteRef = "refs/heads/main"
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("confirmations=%d", len(f.ui.confirms))
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, `local branch "topic"`) {
		t.Fatalf("review used mutable flow data: %s", f.ui.confirms[0].Prompt)
	}
}

func TestBranchSyncUpdateShortUpstreamInputRetainsTrackingSuggestion(t *testing.T) {
	f := newBranchSyncPushFixture(t)
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
	if err := menuItem(t, f.ui.menus[0], "branch.update-selected.upstream").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatalf("prompts=%d", len(f.ui.prompts))
	}
	if err := f.ui.prompts[0].HandleConfirm("topic"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.menus[len(f.ui.menus)-1].Prompt, "refs/cache/review/topic") {
		t.Fatal("short-form equivalent cleared the existing tracking storage")
	}
}

func TestBranchSyncUpdateDefaultsUnknownSourceRefAndRejectsRemoteURL(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.menus[0].Prompt, "refs/heads/topic") {
		t.Fatalf("full source ref not visible in default: %s", f.ui.menus[0].Prompt)
	}
	if err := menuItem(t, f.ui.menus[0], "branch.update-selected.remote").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[0].HandleConfirm("https://user:secret@example.invalid/repo"); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("URL rejection error leaked input or was absent: %v", err)
	}
}

func gitFetchOptionsForTest(remote, source, tracking string) git_commands.BranchFetchOptions {
	return git_commands.BranchFetchOptions{Remote: remote, RemoteRef: source, TrackingRef: tracking}
}

func TestBranchSyncUpdateRejectsRepositoryChangeInQueuedCallbacks(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	if err := f.sync.updateBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/replacement/repository")}
	if err := f.ui.runWorker(t); err == nil || !strings.Contains(err.Error(), "repository changed") {
		t.Fatalf("got %v", err)
	}
	if len(f.ui.ui) != 0 {
		t.Fatal("stale worker queued a UI callback")
	}
}

func TestBranchSyncUpdateQueuedCallbackAndConfirmationRejectRepositoryChange(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f)
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
	f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/replacement/repository")}
	if err := f.ui.runUI(t); err == nil || !strings.Contains(err.Error(), "repository changed") {
		t.Fatalf("queued callback returned %v", err)
	}
	if len(f.ui.menus) != 0 {
		t.Fatal("stale callback opened a menu")
	}
	// Recreate the captured target and prepare the fetch confirmation, then
	// change repositories before the user confirms it.
	f = newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f)
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

func TestBranchSyncUpdateDivergenceFetchesOnlyTrackingRef(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	branchSyncPushGit(t, f.root, "branch", "remote-diverged")
	branchSyncPushGit(t, f.root, "checkout", "-q", "remote-diverged")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "diverged remote tip")
	branchSyncPushGit(t, f.root, "push", "origin", "HEAD:refs/heads/topic")
	remoteTip := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	branchSyncPushGit(t, f.root, "branch", "-D", "remote-diverged")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	configureBranchSyncUpdateStorage(t, f)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged before fetch"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged before fetch"), 0o600); err != nil {
		t.Fatal(err)
	}
	indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	worktreeBefore, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	headBefore := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
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
	divergedMenu := f.ui.menus[len(f.ui.menus)-1]
	if !strings.Contains(divergedMenu.Prompt, "diverged") || !strings.Contains(divergedMenu.Prompt, "will not reset") {
		t.Fatalf("divergence reason is unclear: %s", divergedMenu.Prompt)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatal("divergence offered local integration/update confirmation")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/cache/review/topic"); got != remoteTip {
		t.Fatalf("cache tip %s, want fetched %s", got, remoteTip)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatal("divergent fetch moved local branch")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != headBefore {
		t.Fatal("divergent fetch moved HEAD")
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Fatal("divergent fetch changed index")
	}
	worktreeAfter, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(worktreeBefore, worktreeAfter) {
		t.Fatal("divergent fetch changed worktree bytes")
	}
}

func TestBranchSyncUpdateOccupiedLinkedWorktreeFetchDoesNotAdvance(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "push", "origin", "refs/heads/topic:refs/heads/topic")
	linked := filepath.Join(t.TempDir(), "linked")
	branchSyncPushGit(t, f.root, "worktree", "add", "-q", linked, "topic")
	branchSyncPushGit(t, f.root, "branch", "remote-tmp", "topic")
	branchSyncPushGit(t, f.root, "checkout", "-q", "remote-tmp")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "advance occupied remote")
	branchSyncPushGit(t, f.root, "push", "origin", "HEAD:refs/heads/topic")
	remoteTip := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	branchSyncPushGit(t, f.root, "branch", "-D", "remote-tmp")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	configureBranchSyncUpdateStorage(t, f)
	linkedHead := branchSyncPushGit(t, linked, "rev-parse", "HEAD")
	linkedWorktree, err := os.ReadFile(filepath.Join(linked, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	indexPath := branchSyncPushGit(t, linked, "rev-parse", "--git-path", "index")
	if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(linked, indexPath)
	}
	linkedIndex, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
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
	if err := f.ui.runWorker(t); err == nil || !strings.Contains(strings.ToLower(err.Error()), "worktree") {
		t.Fatalf("occupied update returned %v", err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/cache/review/topic"); got != remoteTip {
		t.Fatalf("cache tip %s, want fetched %s", got, remoteTip)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatal("occupied local branch advanced")
	}
	if got := branchSyncPushGit(t, linked, "rev-parse", "HEAD"); got != linkedHead {
		t.Fatal("linked worktree HEAD changed")
	}
	linkedWorktreeAfter, err := os.ReadFile(filepath.Join(linked, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(linkedWorktree, linkedWorktreeAfter) {
		t.Fatal("linked worktree files changed")
	}
	linkedIndexAfter, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(linkedIndex, linkedIndexAfter) {
		t.Fatal("linked worktree index changed")
	}
}

func configureBranchSyncUpdateStorage(t *testing.T, f *branchSyncPushFixture) {
	t.Helper()
	branchSyncPushGit(t, f.root, "config", "--unset-all", "remote.origin.fetch")
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
}

func TestBranchSyncUpdateStaleRemoteProfileDoesNotFetch(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	seedBranchSyncUpdateRemote(t, f)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	configureBranchSyncUpdateStorage(t, f)
	branchSyncPushGit(t, f.root, "update-ref", "refs/cache/review/topic", f.sourceOID)
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
	otherRemote := filepath.Join(t.TempDir(), "other.git")
	branchSyncPushGit(t, t.TempDir(), "init", "--bare", "-q", otherRemote)
	branchSyncPushGit(t, f.root, "remote", "set-url", "origin", otherRemote)
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/cache/review/topic"); got != f.sourceOID {
		t.Fatal("stale remote profile changed tracking cache")
	}
	if len(f.ui.confirms) != 1 {
		t.Fatal("stale profile proceeded to fetch confirmation")
	}
}

func TestBranchSyncUpdateReinspectRetainsManualChoicesAndRequiresReview(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchSyncPushGit(t, f.root, "remote", "add", "manual", f.remote)
	branchSyncPushGit(t, f.root, "remote", "add", "changed", f.remote)
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncUpdateFlow{
		target:  newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}),
		info:    info,
		options: gitFetchOptionsForTest("manual", "refs/heads/topic", "refs/cache/manual/topic"),
	}
	plan, err := f.git.Sync.PrepareBranchFastForward("topic", f.sourceOID, f.sourceOID)
	if err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", "refs/heads/main")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "changed")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/changed-source")
	if err := flow.applyUpdate(plan); err != nil {
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
	menu := f.ui.menus[len(f.ui.menus)-1]
	for _, expected := range []string{"manual", "refs/heads/topic", "refs/cache/manual/topic"} {
		if !strings.Contains(menu.Prompt, expected) {
			t.Errorf("reinspection lost %q: %s", expected, menu.Prompt)
		}
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("stale plan automatically proceeded to confirmation")
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
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, "manual") {
		t.Fatal("fresh fetch did not require an explicit confirmation with retained options")
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
	seedBranchSyncUpdateRemote(t, f)
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
	seedBranchSyncUpdateRemote(t, f)
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
	menusBefore := len(f.ui.menus)
	if err := f.ui.confirms[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != menusBefore+1 || f.ui.menus[len(f.ui.menus)-1].Title != f.sync.c.Tr.BranchSyncUpdateMenu {
		t.Fatal("Escape on the fetch review did not reopen the update menu")
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
	seedBranchSyncUpdateRemote(t, f)
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
	seedBranchSyncUpdateRemote(t, f)
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

func seedBranchSyncUpdateRemote(t *testing.T, f *branchSyncPushFixture) {
	t.Helper()
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "remote update")
	branchSyncPushGit(t, f.root, "push", "origin", "HEAD:refs/heads/topic")
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", f.sourceOID)
	branchSyncPushGit(t, f.root, "checkout", "-q", f.current)
}
