package controllers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (ui *branchActionMergeUI) LogCommand(string, bool) {}

func branchActionDeleteContexts(f *branchSyncPushFixture, ui *branchActionMergeUI) {
	branches := &guicontext.BranchesContext{FilteredListViewModel: guicontext.NewFilteredListViewModel(
		func() []*models.Branch { return f.ui.model.Branches }, func(branch *models.Branch) []string { return []string{branch.Name} },
	)}
	remoteBranches := &guicontext.RemoteBranchesContext{FilteredListViewModel: guicontext.NewFilteredListViewModel(
		func() []*models.RemoteBranch { return f.ui.model.RemoteBranches }, func(branch *models.RemoteBranch) []string { return []string{branch.Name} },
	)}
	ui.ctx = &guicontext.ContextTree{Branches: branches, RemoteBranches: remoteBranches}
	f.sync.c.HelperCommon.IGetContexts = branchActionResetContexts{tree: ui.ctx}
}

func branchActionDeleteAttachRemote(t *testing.T, f *branchSyncPushFixture) string {
	t.Helper()
	const name = "topic"
	branchActionMergeGit(t, f.root, "push", "-q", "origin", "refs/heads/topic:refs/heads/"+name)
	branchActionMergeGit(t, f.root, "update-ref", "refs/remotes/origin/"+name, f.sourceOID)
	return "refs/remotes/origin/" + name
}

func branchActionDeleteRemoteStart(t *testing.T, f *branchSyncPushFixture, ui *branchActionMergeUI, ref, remote string) {
	t.Helper()
	branchActionDeleteContexts(f, ui)
	if err := helpers.DeleteSelectedRemoteBranches(f.sync.c.HelperCommon, []helpers.RemoteBranchDeleteSource{{Ref: ref, Remote: remote}}); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
}

func branchActionDeleteApprove(t *testing.T, f *branchSyncPushFixture, ui *branchActionMergeUI) error {
	t.Helper()
	if len(f.ui.confirms) != 1 {
		t.Fatalf("confirmations=%d, want one", len(f.ui.confirms))
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		return err
	}
	return ui.runWorker(t)
}

func branchActionDeleteStart(t *testing.T, f *branchSyncPushFixture, ui *branchActionMergeUI) types.CreateMenuOptions {
	t.Helper()
	if err := helpers.DeleteSelectedLocalBranches(f.sync.c.HelperCommon, []string{"refs/heads/topic"}); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	return f.ui.menus[len(f.ui.menus)-1]
}

func branchActionDeleteChoose(t *testing.T, ui *branchActionMergeUI, menu types.CreateMenuOptions) {
	t.Helper()
	const id = "branch.delete-local"
	if err := menuItem(t, menu, id).OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
}

func TestBranchActionDeleteFlowMenuCapturesFullRefsAndCancelPreservesRepository(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchActionDeleteStart(t, f, ui)
	head := branchActionMergeGit(t, f.root, "rev-parse", "HEAD")
	if len(menu.Items) != 3 {
		t.Fatalf("delete choices=%d, want three", len(menu.Items))
	}
	for i, id := range []string{"branch.delete-local", "remote-branch.delete", "branch.delete-local-and-remote"} {
		if menu.Items[i].CommandID != id {
			t.Fatalf("choice %d ID=%q, want %q", i, menu.Items[i].CommandID, id)
		}
	}
	if !strings.Contains(menu.Prompt, "refs/heads/topic") || !strings.Contains(menu.Prompt, f.sourceOID) || !strings.Contains(menu.Prompt, "refs/heads/main") || !strings.Contains(menu.Prompt, head) {
		t.Fatalf("menu omitted captured source/current identities: %q", menu.Prompt)
	}
	if !strings.Contains(menu.Title, "topic") {
		t.Fatalf("delete title did not substitute selected branch: %q", menu.Title)
	}
	branchActionDeleteChoose(t, ui, menu)
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, fmt.Sprintf("%q", "refs/heads/topic")) || !strings.Contains(f.ui.confirms[0].Prompt, f.sourceOID) {
		t.Fatalf("force confirmation omitted source identity: %+v", f.ui.confirms)
	}
	before := branchSyncCurrentPullReadSnapshot(t, f)
	if err := f.ui.confirms[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	after := branchSyncCurrentPullReadSnapshot(t, f)
	if !bytes.Equal(before.head, after.head) || !bytes.Equal(before.index, after.index) || !bytes.Equal(before.tracked, after.tracked) || ui.subprocessCalls != 0 {
		t.Fatal("canceling force-delete confirmation changed repository state")
	}
}

func TestBranchActionDeleteFlowRejectsChangedSourceBeforeExecution(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchActionDeleteStart(t, f, ui)
	branchActionDeleteChoose(t, ui, menu)
	branchActionMergeGit(t, f.root, "update-ref", "refs/heads/topic", "refs/heads/main")
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err == nil {
		t.Fatal("confirmation accepted a changed source ref")
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("stale delete plan launched a terminal subprocess")
	}
}

func TestBranchActionDeleteFlowRejectsRepositoryReplacementAtConfirmation(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchActionDeleteStart(t, f, ui)
	branchActionDeleteChoose(t, ui, menu)
	checkUnchanged := branchActionRebaseRepositoryUnchanged(t, f)
	defer checkUnchanged()
	checkReplacement := branchActionRebaseReplaceRepository(t, ui.branchActionRebaseUI)
	defer checkReplacement()
	if err := f.ui.confirms[0].HandleConfirm(); err == nil || !strings.Contains(err.Error(), f.sync.c.Tr.BranchSyncRepositoryChanged) {
		t.Fatalf("confirmation accepted replacement repository: %v", err)
	}
	if ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
		t.Fatal("repository replacement reached subprocess or refresh")
	}
}

func TestBranchActionDeleteFlowPreservesWorktreeWhenDeletingMergedSource(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	branchActionDeleteContexts(f, ui)
	branchActionMergeGit(t, f.root, "checkout", "topic")
	if err := os.WriteFile(filepath.Join(f.root, "merged-source"), []byte("merged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionMergeGit(t, f.root, "add", "merged-source")
	branchActionMergeGit(t, f.root, "commit", "-qm", "merged source")
	branchActionMergeGit(t, f.root, "checkout", "main")
	branchActionMergeGit(t, f.root, "merge", "--no-ff", "-qm", "merge source", "topic")
	mainOID := branchActionMergeGit(t, f.root, "rev-parse", "HEAD")
	menu := branchActionDeleteStart(t, f, ui)
	if err := menuItem(t, menu, "branch.delete-local").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatalf("merged branch unexpectedly required force confirmation: %+v", f.ui.confirms)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", "HEAD"); got != mainOID {
		t.Fatalf("HEAD changed after ref-only delete: %s", got)
	}
	if _, err := os.Stat(filepath.Join(f.root, "merged-source")); err != nil {
		t.Fatalf("tracked worktree file changed: %v", err)
	}
	cmd := exec.Command("git", "-C", f.root, "show-ref", "--verify", "--quiet", "refs/heads/topic")
	if err := cmd.Run(); err == nil {
		t.Fatal("topic branch still exists")
	}
	if ui.subprocessCalls != 0 {
		t.Fatalf("native command unexpectedly used terminal subprocess path: %d", ui.subprocessCalls)
	}
}

func TestBranchActionDeleteFlowOccupiedLocalOffersRemoteOnly(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	branchActionDeleteContexts(f, ui)
	linked := filepath.Join(t.TempDir(), "linked")
	branchActionMergeGit(t, f.root, "worktree", "add", "-q", linked, "topic")
	menu := branchActionDeleteStart(t, f, ui)
	local := menuItem(t, menu, "branch.delete-local")
	both := menuItem(t, menu, "branch.delete-local-and-remote")
	remote := menuItem(t, menu, "remote-branch.delete")
	if local.DisabledReason == nil || both.DisabledReason == nil || remote.DisabledReason != nil {
		t.Fatalf("occupied branch menu gates local=%v both=%v remote=%v", local.DisabledReason, both.DisabledReason, remote.DisabledReason)
	}
	if err := local.OnPress(); err == nil {
		t.Fatal("programmatic local-delete activation bypassed worktree reservation")
	}
	if err := both.OnPress(); err == nil {
		t.Fatal("programmatic compound-delete activation bypassed worktree reservation")
	}
	if len(f.ui.confirms) != 0 || ui.subprocessCalls != 0 {
		t.Fatal("reserved local deletion reached confirmation or subprocess")
	}
}

func TestBranchActionDeleteFlowRemoteOnlyDeletesCapturedServerRef(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	remoteRef := branchActionDeleteAttachRemote(t, f)
	branchActionDeleteRemoteStart(t, f, ui, remoteRef, "origin")
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, "origin") || !strings.Contains(f.ui.confirms[0].Prompt, "refs/heads/topic") || !strings.Contains(f.ui.confirms[0].Prompt, remoteRef) {
		t.Fatalf("remote confirmation omitted explicit server target: %+v", f.ui.confirms)
	}
	if err := branchActionDeleteApprove(t, f, ui); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchActionMergeGit(t, f.remote, "for-each-ref", "--format=%(refname)", "refs/heads/topic"); got != "" {
		t.Fatalf("server branch survived approved deletion: %q", got)
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatalf("remote-only deletion changed local source: %s", got)
	}
	if ui.subprocessCalls != 0 || len(ui.refresh) != 1 {
		t.Fatalf("remote deletion terminal path/refreshes: %d/%d", ui.subprocessCalls, len(ui.refresh))
	}
}

func TestBranchActionDeleteFlowBothWarnsAndKeepsLocalOnRemoteRejection(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	remoteRef := branchActionDeleteAttachRemote(t, f)
	branchActionMergeGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchActionMergeGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	hook := filepath.Join(f.remote, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	menu := branchActionDeleteStart(t, f, ui)
	if err := menuItem(t, menu, "branch.delete-local-and-remote").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.workers) > 0 {
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("compound delete confirmations=%d", len(f.ui.confirms))
	}
	prompt := f.ui.confirms[0].Prompt
	if !strings.Contains(prompt, fmt.Sprintf("%q", "refs/heads/topic")) || !strings.Contains(prompt, f.sourceOID) || !strings.Contains(prompt, "origin -> refs/heads/topic") {
		t.Fatalf("compound approval omitted force or remote destination: %q", prompt)
	}
	err := branchActionDeleteApprove(t, f, ui)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "pre-receive") {
		t.Fatalf("remote rejection error=%v", err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatalf("local branch was removed despite remote rejection: %s", got)
	}
	if got := branchActionMergeGit(t, f.remote, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatalf("rejected server branch changed: %s", got)
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", remoteRef); got != f.sourceOID {
		t.Fatalf("rejected remote operation changed local cache ref: %s", got)
	}
	if ui.subprocessCalls != 0 || len(ui.refresh) != 1 {
		t.Fatalf("native rejection subprocess/refreshes: %d/%d", ui.subprocessCalls, len(ui.refresh))
	}
}

func TestBranchActionDeleteFlowBothExcludesScheduledCacheAndPreservesCheckout(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	branchActionDeleteContexts(f, ui)
	remoteRef := branchActionDeleteAttachRemote(t, f)
	branchActionMergeGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchActionMergeGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	branchActionMergeGit(t, f.root, "tag", "topic", "refs/heads/main")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged raw bytes\x00\xff"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionMergeGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("work bytes\xfe\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "untracked-delete-check"), []byte("untracked\x00\xfd"), 0o600); err != nil {
		t.Fatal(err)
	}
	untrackedBefore, err := os.ReadFile(filepath.Join(f.root, "untracked-delete-check"))
	if err != nil {
		t.Fatal(err)
	}
	initial := branchActionRebaseRepositoryUnchanged(t, f)
	menu := branchActionDeleteStart(t, f, ui)
	if err := menuItem(t, menu, "branch.delete-local-and-remote").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("compound confirmations=%d", len(f.ui.confirms))
	}
	prompt := f.ui.confirms[0].Prompt
	if !strings.Contains(prompt, "history not preserved") || !strings.Contains(prompt, "Delete these local branches anyway?") {
		t.Fatalf("compound warning omitted force-delete rationale: %q", prompt)
	}
	for _, part := range []string{fmt.Sprintf("%q", "refs/heads/topic"), f.sourceOID, "origin -> refs/heads/topic"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("compound warning omitted %q: %q", part, prompt)
		}
	}
	if err := branchActionDeleteApprove(t, f, ui); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	initial()
	if got, err := os.ReadFile(filepath.Join(f.root, "untracked-delete-check")); err != nil || !bytes.Equal(got, untrackedBefore) {
		t.Fatalf("untracked worktree bytes changed: %v", err)
	}
	if got := branchActionMergeGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Fatalf("compound deletion changed checkout: %s", got)
	}
	if got := branchActionMergeGit(t, f.remote, "for-each-ref", "--format=%(refname)", "refs/heads/topic"); got != "" {
		t.Fatalf("server destination remains: %s", got)
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/tags/topic"); got != branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/main") {
		t.Fatalf("same-named tag changed: %s", got)
	}
	if err := exec.Command("git", "-C", f.root, "show-ref", "--verify", "--quiet", remoteRef).Run(); err == nil {
		t.Fatalf("scheduled tracking cache ref remains: %s", remoteRef)
	}
	if ui.subprocessCalls != 0 || len(ui.refresh) != 1 {
		t.Fatalf("native compound path/refreshes: %d/%d", ui.subprocessCalls, len(ui.refresh))
	}
}

func TestBranchActionDeleteFlowRejectsCurrentHeadChangeAtQueuedExecution(t *testing.T) {
	for _, mutation := range []string{"head", "config"} {
		t.Run(mutation, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			menu := branchActionDeleteStart(t, f, ui)
			branchActionDeleteChoose(t, ui, menu)
			if err := f.ui.confirms[0].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "head":
				branchActionMergeGit(t, f.root, "checkout", "--detach", "-q", f.sourceOID)
			case "config":
				branchActionMergeGit(t, f.root, "config", "branch.topic.pushRemote", "changed-remote")
			}
			checkUnchanged := branchActionRebaseRepositoryUnchanged(t, f)
			defer checkUnchanged()
			if err := ui.runWorker(t); err == nil {
				t.Fatal("queued delete accepted changed current HEAD/config")
			}
			if ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
				t.Fatal("stale queued delete reached subprocess or refresh")
			}
			if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
				t.Fatalf("stale queued delete changed source ref: %s", got)
			}
		})
	}
}

func TestBranchActionDeleteFlowManualRemoteFallbackAndMultipleChoicesCancel(t *testing.T) {
	t.Run("manual fallback", func(t *testing.T) {
		f, ui := branchActionMergeFixture(t)
		branchActionMergeGit(t, f.root, "config", "branch.topic.remote", ".")
		branchActionDeleteStart(t, f, ui)
		if err := menuItem(t, f.ui.menus[0], "remote-branch.delete").OnPress(); err != nil {
			t.Fatal(err)
		}
		if len(f.ui.menus) < 2 {
			t.Fatalf("manual remote chooser not shown: %+v", f.ui.menus)
		}
		manual := f.ui.menus[len(f.ui.menus)-1]
		if err := menuItem(t, manual, "branch.delete.choose-remote-ref").OnPress(); err != nil {
			t.Fatal(err)
		}
		if len(ui.prompts) != 1 {
			t.Fatalf("remote name prompts=%d; menu title=%q prompt=%q items=%+v", len(ui.prompts), manual.Title, manual.Prompt, manual.Items)
		}
		if err := ui.prompts[0].HandleConfirm("origin"); err != nil {
			t.Fatal(err)
		}
		if len(ui.prompts) != 2 {
			t.Fatalf("remote ref prompts=%d", len(ui.prompts))
		}
		if err := ui.prompts[1].HandleConfirm("refs/heads/topic"); err != nil {
			t.Fatal(err)
		}
		if len(f.ui.workers) > 0 {
			if err := ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
		}
		if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, "origin") || !strings.Contains(f.ui.confirms[0].Prompt, "refs/heads/topic") {
			t.Fatalf("manual target not captured in approval: %+v", f.ui.confirms)
		}
	})
	t.Run("cancel first candidate", func(t *testing.T) {
		f, ui := branchActionMergeFixture(t)
		branchActionMergeGit(t, f.root, "remote", "add", "origin-two", filepath.Join(t.TempDir(), "other.git"))
		branchActionMergeGit(t, f.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/custom/cache/*")
		branchActionMergeGit(t, f.root, "config", "remote.origin-two.fetch", "+refs/heads/*:refs/custom/cache/*")
		branchActionMergeGit(t, f.root, "config", "branch.topic.remote", "origin")
		branchActionMergeGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
		branchActionMergeGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/other")
		branchActionMergeGit(t, f.root, "update-ref", "refs/custom/cache/topic", f.sourceOID)
		branchActionMergeGit(t, f.root, "update-ref", "refs/custom/cache/other", f.sourceOID)
		branchActionDeleteStart(t, f, ui)
		if err := menuItem(t, f.ui.menus[0], "remote-branch.delete").OnPress(); err != nil {
			t.Fatal(err)
		}
		chooser := f.ui.menus[len(f.ui.menus)-1]
		if len(chooser.Items) < 3 || chooser.Items[1].Label != "refs/heads/topic" || chooser.Items[2].Label != "refs/heads/other" {
			t.Fatalf("remote chooser did not expose exact candidate refs: %+v", chooser.Items)
		}
		if err := chooser.Items[0].OnPress(); err != nil {
			t.Fatal(err)
		}
		if len(f.ui.confirms) != 0 {
			t.Fatal("canceling first candidate silently selected another destination")
		}
		if ui.subprocessCalls != 0 {
			t.Fatal("canceling candidate launched native command")
		}
	})
}

func TestBranchActionDeleteFlowRepeatedConfirmationAndDirectRemoteCacheSource(t *testing.T) {
	t.Run("confirmation once", func(t *testing.T) {
		f, ui := branchActionMergeFixture(t)
		menu := branchActionDeleteStart(t, f, ui)
		branchActionDeleteChoose(t, ui, menu)
		confirm := f.ui.confirms[0].HandleConfirm
		if err := confirm(); err != nil {
			t.Fatal(err)
		}
		if err := confirm(); err == nil {
			t.Fatal("repeated approval accepted after callbacks were consumed")
		}
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(f.root, ".git", "refs", "heads", "topic")); err == nil {
			t.Fatal("approved local branch was not deleted")
		}
	})
	t.Run("remote cache source", func(t *testing.T) {
		f, ui := branchActionMergeFixture(t)
		ref := branchActionDeleteAttachRemote(t, f)
		branchActionDeleteContexts(f, ui)
		if err := helpers.DeleteSelectedRemoteBranches(f.sync.c.HelperCommon, []helpers.RemoteBranchDeleteSource{{Ref: ref, Remote: "origin"}}); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := branchActionDeleteApprove(t, f, ui); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if _, err := exec.Command("git", "-C", f.root, "show-ref", "--verify", "--quiet", ref).Output(); err == nil {
			t.Fatalf("selected remote cache ref survived deletion: %s", ref)
		}
		if got := branchActionMergeGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "main" {
			t.Fatalf("remote-pane delete changed checkout: %s", got)
		}
		if len(ui.refresh) != 1 || ui.subprocessCalls != 0 {
			t.Fatalf("remote cache deletion refresh/subprocess: %d/%d", len(ui.refresh), ui.subprocessCalls)
		}
	})
}
