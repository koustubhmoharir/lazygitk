package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchActionMergeUI struct {
	*branchActionRebaseUI
	prompts []types.PromptOpts
}

func (ui *branchActionMergeUI) Prompt(opts types.PromptOpts) {
	ui.prompts = append(ui.prompts, opts)
}

func branchActionMergeFixture(t *testing.T) (*branchSyncPushFixture, *branchActionMergeUI) {
	t.Helper()
	f, rebaseUI := branchActionRebaseFixture(t)
	ui := &branchActionMergeUI{branchActionRebaseUI: rebaseUI}
	f.sync.c.HelperCommon.IGuiCommon = ui
	return f, ui
}

func branchActionMergeStart(t *testing.T, f *branchSyncPushFixture, ui *branchActionMergeUI) types.CreateMenuOptions {
	t.Helper()
	if err := helpers.MergeSelectedIntoCurrent(f.sync.c.HelperCommon, "refs/heads/topic"); err != nil {
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

func branchActionMergeChoose(t *testing.T, ui *branchActionMergeUI, menu types.CreateMenuOptions, id string) {
	t.Helper()
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

func TestBranchActionMergeFlowCapturesFullRefsAndCancelDoesNotMutate(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchActionMergeStart(t, f, ui)
	head := branchActionMergeGit(t, f.root, "rev-parse", "HEAD")
	if len(menu.Items) != 5 {
		t.Fatalf("merge choices=%d, want five", len(menu.Items))
	}
	wantIDs := []string{"branch.merge.regular", "branch.merge.ff", "branch.merge.no-ff", "branch.merge.squash-uncommitted", "branch.merge.squash-committed"}
	for i, id := range wantIDs {
		if menu.Items[i].CommandID != id {
			t.Fatalf("choice %d ID=%q, want %q", i, menu.Items[i].CommandID, id)
		}
	}
	if !strings.Contains(menu.Prompt, "refs/heads/topic") || !strings.Contains(menu.Prompt, head) {
		t.Fatalf("menu omitted captured full source/current identities: %q", menu.Prompt)
	}
	branchActionMergeChoose(t, ui, menu, "branch.merge.regular")
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, "refs/heads/topic") || !strings.Contains(f.ui.confirms[0].Prompt, head) {
		t.Fatalf("confirmation omitted captured full identities: %+v", f.ui.confirms)
	}
	before := branchSyncCurrentPullReadSnapshot(t, f)
	if err := f.ui.confirms[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	after := branchSyncCurrentPullReadSnapshot(t, f)
	if string(before.head) != string(after.head) || string(before.index) != string(after.index) || string(before.tracked) != string(after.tracked) || ui.subprocessCalls != 0 {
		t.Fatal("canceling merge confirmation mutated the repository or ran a subprocess")
	}
}

func TestBranchActionMergeFlowMergesOnlyCurrentAndUsesNativeArgs(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "current-only"), []byte("current commit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionMergeGit(t, f.root, "add", "current-only")
	branchActionMergeGit(t, f.root, "commit", "-qm", "current only")
	initialHead := branchActionMergeGit(t, f.root, "rev-parse", "HEAD")
	source := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/topic")
	menu := branchActionMergeStart(t, f, ui)
	branchActionMergeChoose(t, ui, menu, "branch.merge.no-ff")
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
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
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", "HEAD"); got == initialHead {
		t.Fatal("merge did not move current branch")
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/topic"); got != source {
		t.Fatalf("source ref changed: %s", got)
	}
	if got := branchActionMergeGit(t, f.root, "show", "-s", "--format=%P", "HEAD"); len(strings.Fields(got)) != 2 {
		t.Fatalf("no-ff merge parents=%q", got)
	}
	if !strings.Contains(strings.Join(ui.commandArgs, " "), "--no-ff") || strings.Contains(strings.Join(ui.commandArgs, " "), "--autostash") {
		t.Fatalf("native merge argv unexpected: %v", ui.commandArgs)
	}
}

func TestBranchActionMergeFlowRejectsChangedSourceBeforeNativeCommand(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchActionMergeStart(t, f, ui)
	branchActionMergeChoose(t, ui, menu, "branch.merge.regular")
	branchActionMergeGit(t, f.root, "update-ref", "refs/heads/topic", "refs/heads/main")
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("changed source preflight error=%v", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("stale merge plan launched a native command")
	}
}

func TestBranchActionMergeFlowRejectsRepositoryReplacementBeforeMenuAndConfirmation(t *testing.T) {
	for _, boundary := range []string{"menu", "confirmation"} {
		for _, swap := range []string{"generation", "backend"} {
			t.Run(boundary+"/"+swap, func(t *testing.T) {
				f, ui := branchActionMergeFixture(t)
				var menu types.CreateMenuOptions
				if boundary == "menu" {
					if err := helpers.MergeSelectedIntoCurrent(f.sync.c.HelperCommon, "refs/heads/topic"); err != nil {
						t.Fatal(err)
					}
				} else {
					menu = branchActionMergeStart(t, f, ui)
					if err := menuItem(t, menu, "branch.merge.regular").OnPress(); err != nil {
						t.Fatal(err)
					}
					if err := ui.runWorker(t); err != nil {
						t.Fatal(err)
					}
					if err := ui.runUI(t); err != nil {
						t.Fatal(err)
					}
				}
				checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
				defer checkCaptured()
				checkReplacement := func() {}
				if swap == "generation" {
					f.ui.state.generation++
				} else {
					checkReplacement = branchActionRebaseReplaceRepository(t, ui.branchActionRebaseUI)
					defer checkReplacement()
				}
				if boundary == "menu" {
					if err := ui.runWorker(t); err == nil || !strings.Contains(err.Error(), f.sync.c.Tr.BranchSyncRepositoryChanged) {
						t.Fatalf("menu worker accepted repository replacement: %v", err)
					}
					if len(f.ui.menus) != 0 {
						t.Fatal("stale menu was presented")
					}
				} else if err := f.ui.confirms[0].HandleConfirm(); err == nil || !strings.Contains(err.Error(), f.sync.c.Tr.BranchSyncRepositoryChanged) {
					t.Fatalf("confirmation accepted repository replacement: %v", err)
				}
				if ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
					t.Fatal("repository replacement reached native merge or refresh")
				}
				checkReplacement()
			})
		}
	}
}

func TestBranchActionMergeFlowRejectsRepositoryReplacementAtTerminalBoundary(t *testing.T) {
	for _, swap := range []string{"generation", "backend"} {
		t.Run(swap, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			menu := branchActionMergeStart(t, f, ui)
			branchActionMergeChoose(t, ui, menu, "branch.merge.regular")
			if err := f.ui.confirms[0].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); err != nil {
				t.Fatal(err)
			}
			checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
			defer checkCaptured()
			if swap == "generation" {
				f.ui.state.generation++
			} else {
				checkReplacement := branchActionRebaseReplaceRepository(t, ui.branchActionRebaseUI)
				defer checkReplacement()
			}
			if err := ui.runUI(t); err == nil || !strings.Contains(err.Error(), f.sync.c.Tr.BranchSyncRepositoryChanged) {
				t.Fatalf("terminal callback accepted repository replacement: %v", err)
			}
			if ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
				t.Fatal("replacement repository reached native merge or refresh")
			}
		})
	}
}

func TestBranchActionMergeFlowRejectsRepositoryReplacementAfterNativeMerge(t *testing.T) {
	for _, boundary := range []string{"postcommit-worker", "refresh-then"} {
		for _, swap := range []string{"generation", "backend"} {
			t.Run(boundary+"/"+swap, func(t *testing.T) {
				f, ui := branchActionMergeFixture(t)
				menu := branchActionMergeStart(t, f, ui)
				branchActionMergeChoose(t, ui, menu, "branch.merge.regular")
				if err := f.ui.confirms[0].HandleConfirm(); err != nil {
					t.Fatal(err)
				}
				if err := ui.runBlockingWorker(t); err != nil {
					t.Fatal(err)
				}
				if err := ui.runUI(t); err != nil {
					t.Fatal(err)
				}
				if boundary == "refresh-then" {
					if err := ui.runBlockingWorker(t); err != nil {
						t.Fatal(err)
					}
					if err := ui.runUI(t); err != nil {
						t.Fatal(err)
					}
					if len(ui.refresh) != 1 || ui.refresh[0].Then == nil {
						t.Fatal("merge completion did not queue refresh callback")
					}
				}
				checkReplacement := func() {}
				if swap == "generation" {
					f.ui.state.generation++
				} else {
					checkReplacement = branchActionRebaseReplaceRepository(t, ui.branchActionRebaseUI)
					defer checkReplacement()
				}
				if boundary == "postcommit-worker" {
					if err := ui.runBlockingWorker(t); err == nil || !strings.Contains(err.Error(), f.sync.c.Tr.BranchSyncRepositoryChanged) {
						t.Fatalf("post-merge worker accepted repository replacement: %v", err)
					}
				} else if err := ui.refresh[0].Then(); err == nil || !strings.Contains(err.Error(), f.sync.c.Tr.BranchSyncRepositoryChanged) {
					t.Fatalf("refresh callback accepted repository replacement: %v", err)
				}
				if boundary == "postcommit-worker" && ui.subprocessCalls != 1 {
					t.Fatalf("stale post-merge worker launched another subprocess: %d", ui.subprocessCalls)
				}
				checkReplacement()
			})
		}
	}
}

func TestBranchActionMergeFlowStagedAndIgnoredRisksRequireExplicitChoices(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	branchActionMergeGit(t, f.root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(f.root, "ignored-incoming"), []byte("source bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionMergeGit(t, f.root, "add", "ignored-incoming")
	branchActionMergeGit(t, f.root, "commit", "-qm", "incoming ignored path")
	branchActionMergeGit(t, f.root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(f.root, ".git", "info", "exclude"), []byte("ignored-incoming\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ignoredPath := filepath.Join(f.root, "ignored-incoming")
	if err := os.WriteFile(ignoredPath, []byte("preserve ignored bytes\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged change"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionMergeGit(t, f.root, "add", "tracked")
	indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	ignoredBefore, err := os.ReadFile(ignoredPath)
	if err != nil {
		t.Fatal(err)
	}
	menu := branchActionMergeStart(t, f, ui)
	if err := menuItem(t, menu, "branch.merge.squash-committed").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.prompts[0].HandleConfirm("approved test message"); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	stagedReview := f.ui.menus[len(f.ui.menus)-1]
	if len(stagedReview.Items) == 0 || stagedReview.Items[0].CommandID != "branch.merge.squash-committed.cancel-staged-risk" {
		t.Fatal("staged risk menu must lead with Cancel")
	}
	if menuItem(t, stagedReview, "branch.merge.squash-committed.cancel-staged-risk") == nil {
		t.Fatal("staged risk review omitted cancel")
	}
	pathItem := menuItem(t, stagedReview, "branch.merge.squash-committed.inspect-staged-path")
	if pathItem == nil || !pathItem.OpensMenu {
		t.Fatal("staged path row is not inspection-only")
	}
	if err := pathItem.OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, stagedReview, "branch.merge.squash-committed.approve-staged").OnPress(); err != nil {
		t.Fatal(err)
	}
	ignoredReview := f.ui.menus[len(f.ui.menus)-1]
	pathItem = menuItem(t, ignoredReview, "branch.merge.squash-committed.inspect-ignored-path")
	if pathItem == nil || !pathItem.OpensMenu {
		t.Fatal("ignored path row is not inspection-only")
	}
	if err := pathItem.OnPress(); err != nil {
		t.Fatal(err)
	}
	ignoredReview = f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, ignoredReview, "branch.merge.squash-committed.cancel-ignored-risk").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ignoredReview.Items) == 0 || ignoredReview.Items[0].CommandID != "branch.merge.squash-committed.cancel-ignored-risk" {
		t.Fatal("ignored risk menu must lead with Cancel")
	}
	// Reopen the same captured plan, approve ignored files, then change the
	// approved bytes before native execution; the fingerprint must reject it.
	if err := menuItem(t, stagedReview, "branch.merge.squash-committed.approve-staged").OnPress(); err != nil {
		t.Fatal(err)
	}
	ignoredReview = f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, ignoredReview, "branch.merge.squash-committed.approve-ignored").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ignoredPath, []byte("changed after approval\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil {
		t.Fatal("native merge accepted ignored bytes changed after approval")
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	ignoredAfter, err := os.ReadFile(ignoredPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(indexBefore) != string(indexAfter) || string(ignoredAfter) != "changed after approval\x00" || string(ignoredBefore) == string(ignoredAfter) || ui.subprocessCalls != 0 {
		t.Fatal("risk review changed index, accepted stale ignored approval, or ran native merge")
	}
}

func TestBranchActionMergeFlowRegularAndSquashConflictsDoNotAutocommit(t *testing.T) {
	for _, mode := range []string{"branch.merge.regular", "branch.merge.squash-uncommitted"} {
		t.Run(mode, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("current conflicting change\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			branchActionMergeGit(t, f.root, "add", "tracked")
			branchActionMergeGit(t, f.root, "commit", "-qm", "current conflicting change")
			menu := branchActionMergeStart(t, f, ui)
			branchActionMergeChoose(t, ui, menu, mode)
			if err := f.ui.confirms[0].HandleConfirm(); err != nil {
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
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(ui.refresh) != 1 || ui.refresh[0].Then == nil {
				t.Fatal("conflict did not queue a refresh completion")
			}
			if err := ui.refresh[0].Then(); err != nil {
				t.Fatal(err)
			}
			conflictMenu := f.ui.menus[len(f.ui.menus)-1]
			view := menuItem(t, conflictMenu, "conflict.view")
			if view == nil {
				t.Fatal("conflict choices omitted view-conflicts")
			}
			if mode == "branch.merge.regular" {
				menuItem(t, conflictMenu, "operation.abort")
			} else {
				if len(conflictMenu.Items) != 1 {
					t.Fatalf("squash conflict unexpectedly offered abort/autocommit: %+v", conflictMenu.Items)
				}
			}
			if got := branchActionMergeGit(t, f.root, "status", "--porcelain"); !strings.HasPrefix(got, "UU tracked") {
				t.Fatalf("conflict state was not preserved: %q", got)
			}
			if strings.Contains(strings.Join(ui.commandArgs, " "), " commit") {
				t.Fatalf("merge conflict unexpectedly invoked commit: %v", ui.commandArgs)
			}
		})
	}
}

func TestBranchActionMergeFlowSquashCommittedConflictDoesNotStartCommitPhase(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("current conflicting change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionMergeGit(t, f.root, "add", "tracked")
	branchActionMergeGit(t, f.root, "commit", "-qm", "current conflicting change")
	headBefore := branchActionMergeGit(t, f.root, "rev-parse", "HEAD")
	menu := branchActionMergeStart(t, f, ui)
	if err := menuItem(t, menu, "branch.merge.squash-committed").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.prompts) != 1 {
		t.Fatal("squash-committed conflict path skipped message prompt")
	}
	if err := ui.prompts[0].HandleConfirm("must not be committed"); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
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
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.refresh) != 1 || ui.refresh[0].Then == nil {
		t.Fatal("squash conflict did not queue refresh completion")
	}
	if err := ui.refresh[0].Then(); err != nil {
		t.Fatal(err)
	}
	if got := branchActionMergeGit(t, f.root, "rev-parse", "HEAD"); got != headBefore {
		t.Fatalf("conflicted squash moved HEAD: %s -> %s", headBefore, got)
	}
	if got := branchActionMergeGit(t, f.root, "status", "--porcelain"); !strings.HasPrefix(got, "UU tracked") {
		t.Fatalf("squash conflict state was not preserved: %q", got)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("conflicted squash invoked an automatic commit subprocess: calls=%d args=%v", ui.subprocessCalls, ui.commandArgs)
	}
	conflictMenu := f.ui.menus[len(f.ui.menus)-1]
	if len(conflictMenu.Items) != 1 || conflictMenu.Items[0].CommandID != "conflict.view" {
		t.Fatalf("squash conflict offered unexpected actions: %+v", conflictMenu.Items)
	}
}

func TestBranchActionMergeFlowAbortIsBoundToOriginalNativeMerge(t *testing.T) {
	for _, replace := range []bool{false, true} {
		name := "original operation"
		if replace {
			name = "replaced MERGE_HEAD"
		}
		t.Run(name, func(t *testing.T) {
			f, ui, conflictMenu := branchActionMergeCreateRegularConflict(t)
			if err := menuItem(t, conflictMenu, "operation.abort").OnPress(); err != nil {
				t.Fatal(err)
			}
			if replace {
				currentHead := branchActionMergeGit(t, f.root, "rev-parse", "HEAD")
				if err := os.WriteFile(filepath.Join(f.root, ".git", "MERGE_HEAD"), []byte(currentHead+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
					t.Fatalf("abort accepted replaced native operation: %v", err)
				}
				if ui.subprocessCalls != 1 {
					t.Fatalf("stale abort invoked subprocess; calls=%d", ui.subprocessCalls)
				}
				if _, err := os.Stat(filepath.Join(f.root, ".git", "MERGE_HEAD")); err != nil {
					t.Fatalf("stale abort changed native merge state: %v", err)
				}
				return
			}
			if err := ui.runBlockingWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(f.root, ".git", "MERGE_HEAD")); !os.IsNotExist(err) {
				t.Fatalf("abort did not clear original MERGE_HEAD: %v", err)
			}
			if got := branchActionMergeGit(t, f.root, "status", "--porcelain"); got != "" {
				t.Fatalf("original merge abort left changes: %q", got)
			}
		})
	}
}

func branchActionMergeCreateRegularConflict(t *testing.T) (*branchSyncPushFixture, *branchActionMergeUI, types.CreateMenuOptions) {
	t.Helper()
	f, ui := branchActionMergeFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("current conflicting change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionMergeGit(t, f.root, "add", "tracked")
	branchActionMergeGit(t, f.root, "commit", "-qm", "current conflicting change")
	menu := branchActionMergeStart(t, f, ui)
	branchActionMergeChoose(t, ui, menu, "branch.merge.regular")
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
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
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.refresh) != 1 || ui.refresh[0].Then == nil {
		t.Fatal("merge conflict did not queue refresh completion")
	}
	if err := ui.refresh[0].Then(); err != nil {
		t.Fatal(err)
	}
	return f, ui, f.ui.menus[len(f.ui.menus)-1]
}

func TestBranchActionMergeFlowPromptsForCapturedSquashCommitMessage(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchActionMergeStart(t, f, ui)
	if err := menuItem(t, menu, "branch.merge.squash-committed").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.prompts) != 1 {
		t.Fatalf("message prompts=%d", len(ui.prompts))
	}
	message := "Reviewed squash message\nsecond line"
	if err := ui.prompts[0].HandleConfirm(message); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("confirmation omitted captured message: %+v", f.ui.confirms)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, strconv.Quote(message)) {
		t.Fatalf("confirmation omitted quoted multiline message: %q", f.ui.confirms[0].Prompt)
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
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchActionMergeGit(t, f.root, "show", "-s", "--format=%B", "HEAD"); strings.TrimSpace(got) != message {
		t.Fatalf("squash commit message=%q, want %q", got, message)
	}
	if got := branchActionMergeGit(t, f.root, "show", "-s", "--format=%P", "HEAD"); len(strings.Fields(got)) != 1 {
		t.Fatalf("squash commit parents=%q, want one", got)
	}
}

func branchActionMergeGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	return branchSyncPushGit(t, root, args...)
}
