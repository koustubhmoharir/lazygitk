package controllers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchCreateContextStack struct {
	*branchActionRebaseContextStack
	current types.Context
}

func (stack *branchCreateContextStack) Current() types.Context { return stack.current }

type branchCreateUI struct {
	*branchActionMergeUI
	stack *branchCreateContextStack
}

func (ui *branchCreateUI) Context() types.IContextMgr { return ui.stack }

func branchCreateStart(t *testing.T, f *branchSyncPushFixture, ui *branchActionMergeUI) {
	t.Helper()
	if err := helpers.NewRefsHelper(f.sync.c.HelperCommon, nil).NewBranch("refs/heads/topic", "topic", "topic"); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
}

func branchCreateReview(t *testing.T, f *branchSyncPushFixture, ui *branchActionMergeUI, name string) types.CreateMenuOptions {
	t.Helper()
	branchCreateStart(t, f, ui)
	if len(ui.prompts) != 1 {
		t.Fatalf("prompts=%d, want one", len(ui.prompts))
	}
	if err := ui.prompts[0].HandleConfirm(name); err != nil {
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

func TestBranchActionCreateFlowReviewsCapturedSourceAndPolicies(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	branchCreateStart(t, f, ui)
	prompt := ui.prompts[0]
	if !prompt.PreserveWhitespace || prompt.InitialContent != "topic" {
		t.Fatalf("unexpected name prompt: %+v", prompt)
	}
	if err := prompt.HandleConfirm("literal-name"); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	menu := f.ui.menus[len(f.ui.menus)-1]
	if len(menu.Items) != 4 {
		t.Fatalf("policy choices=%d, want four", len(menu.Items))
	}
	want := []string{"branch.create.no-tracking", "branch.create.configured-tracking", "branch.create.direct-tracking", "branch.create.cancel"}
	for i, id := range want {
		if menu.Items[i].CommandID != id {
			t.Fatalf("choice %d ID=%q, want %q", i, menu.Items[i].CommandID, id)
		}
	}
	if !strings.Contains(menu.Prompt, "refs/heads/topic") || !strings.Contains(menu.Prompt, f.sourceOID) || !strings.Contains(menu.Prompt, "refs/heads/main") {
		t.Fatalf("review omitted captured source/head: %q", menu.Prompt)
	}
	if menu.Items[0].Label != f.sync.c.Tr.BranchActionCreateNoTracking || menu.Items[1].Label != f.sync.c.Tr.BranchActionCreateConfiguredTracking {
		t.Fatal("changed literal name did not default to no tracking")
	}
	if menu.Items[2].DisabledReason != nil {
		t.Fatalf("branch source unexpectedly disables direct tracking: %v", menu.Items[2].DisabledReason)
	}
}

func TestBranchActionCreateFlowRejectsInvalidLiteralNameWithoutSanitizing(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	branchCreateStart(t, f, ui)
	if err := ui.prompts[0].HandleConfirm(" literal name "); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err == nil || !strings.Contains(err.Error(), "invalid local branch name") {
		t.Fatalf("invalid literal name error=%v", err)
	}
	if len(f.ui.menus) != 0 {
		t.Fatal("invalid name was sanitized and presented for review")
	}
	if got := branchActionMergeGit(t, f.root, "branch", "--list"); strings.Contains(got, "literal-name") {
		t.Fatalf("sanitized branch was created: %q", got)
	}
}

func TestBranchActionCreateFlowDisablesDirectTrackingForFullOID(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	if err := helpers.NewRefsHelper(f.sync.c.HelperCommon, nil).NewBranch(f.sourceOID, "topic commit", "from-oid"); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.prompts[0].HandleConfirm("from-oid"); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	item := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.create.direct-tracking")
	if item.DisabledReason == nil {
		t.Fatal("direct tracking from full OID was enabled")
	}
	if err := item.OnPress(); err == nil {
		t.Fatal("programmatic direct tracking from full OID was accepted")
	}
}

func TestBranchActionCreateFlowRejectsStaleSourceHeadConfigAndDestination(t *testing.T) {
	for _, stale := range []string{"source", "head", "config", "destination"} {
		t.Run(stale, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			menu := branchCreateReview(t, f, ui, "created")
			var externalDestination string
			if stale == "source" {
				branchActionMergeGit(t, f.root, "update-ref", "refs/heads/topic", "refs/heads/main")
			}
			if stale == "head" {
				branchActionMergeGit(t, f.root, "commit", "--allow-empty", "-qm", "move HEAD")
			}
			if stale == "destination" {
				externalDestination = branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/main")
				branchActionMergeGit(t, f.root, "branch", "created", externalDestination)
			}
			if stale == "config" {
				branchActionMergeGit(t, f.root, "config", "branch.autoSetupMerge", "always")
			}
			if err := menuItem(t, menu, "branch.create.no-tracking").OnPress(); err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); err == nil {
				t.Fatalf("accepted stale %s", stale)
			}
			if ui.subprocessCalls != 0 {
				t.Fatal("stale create launched native command")
			}
			if stale == "destination" {
				if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/created"); got != externalDestination {
					t.Fatalf("stale create changed external destination: got %s want %s", got, externalDestination)
				}
			} else if got := branchActionMergeGit(t, f.root, "branch", "--list", "created"); got != "" {
				t.Fatalf("stale create made ref: %q", got)
			}
		})
	}
}

func TestBranchActionCreateFlowRejectsRepositoryReplacementAtWorkerAndRefreshThen(t *testing.T) {
	for _, replacement := range []string{"generation", "backend"} {
		t.Run("worker/"+replacement, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			menu := branchCreateReview(t, f, ui, "created")
			if err := menuItem(t, menu, "branch.create.no-tracking").OnPress(); err != nil {
				t.Fatal(err)
			}
			checkOriginal := branchActionRebaseRepositoryUnchanged(t, f)
			defer checkOriginal()
			checkReplacement := func() {}
			if replacement == "generation" {
				f.ui.state.generation++
			} else {
				checkReplacement = branchActionRebaseReplaceRepository(t, ui.branchActionRebaseUI)
				defer checkReplacement()
			}
			if err := ui.runBlockingWorker(t); err == nil {
				t.Fatal("worker accepted replaced repository")
			}
			if ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
				t.Fatal("replacement reached command or refresh")
			}
			checkReplacement()
		})
	}
	for _, replacement := range []string{"generation", "backend"} {
		t.Run("refresh-then/"+replacement, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			branchActionDeleteContexts(f, ui)
			f.sync.c.HelperCommon.IGuiCommon = &branchCreateUI{branchActionMergeUI: ui, stack: &branchCreateContextStack{branchActionRebaseContextStack: &branchActionRebaseContextStack{}, current: ui.ctx.Branches}}
			menu := branchCreateReview(t, f, ui, "created")
			if err := menuItem(t, menu, "branch.create.no-tracking").OnPress(); err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(ui.refresh) != 1 || ui.refresh[0].Then == nil {
				t.Fatal("missing refresh Then callback")
			}
			checkReplacement := func() {}
			if replacement == "generation" {
				f.ui.state.generation++
			} else {
				checkReplacement = branchActionRebaseReplaceRepository(t, ui.branchActionRebaseUI)
				defer checkReplacement()
			}
			if err := ui.refresh[0].Then(); err == nil {
				t.Fatal("refresh Then accepted replacement repository")
			}
			checkReplacement()
		})
	}
}

func TestBranchActionCreateFlowNativeCheckoutAndTracking(t *testing.T) {
	for _, policy := range []struct {
		id       string
		tracking git_commands.BranchActionCreateTracking
	}{
		{"branch.create.no-tracking", git_commands.BranchActionCreateNoTracking},
		{"branch.create.configured-tracking", git_commands.BranchActionCreateConfiguredTracking},
		{"branch.create.direct-tracking", git_commands.BranchActionCreateDirectTracking},
	} {
		t.Run(policy.id, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			if policy.tracking == git_commands.BranchActionCreateConfiguredTracking {
				branchActionMergeGit(t, f.root, "config", "branch.autoSetupMerge", "always")
			}
			branchActionDeleteContexts(f, ui)
			f.sync.c.HelperCommon.IGuiCommon = &branchCreateUI{branchActionMergeUI: ui, stack: &branchCreateContextStack{branchActionRebaseContextStack: &branchActionRebaseContextStack{}, current: ui.ctx.Branches}}
			branchCreateStart(t, f, ui)
			if err := ui.prompts[0].HandleConfirm("created"); err != nil {
				t.Fatal(err)
			}
			if err := ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			menu := f.ui.menus[len(f.ui.menus)-1]
			if err := menuItem(t, menu, policy.id).OnPress(); err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if got := branchActionMergeGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "created" {
				t.Fatalf("HEAD branch=%q", got)
			}
			if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/created"); got != f.sourceOID {
				t.Fatalf("new branch OID=%s", got)
			}
			if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
				t.Fatalf("source ref changed: %s", got)
			}
			if policy.tracking == git_commands.BranchActionCreateNoTracking {
				cmd := exec.Command("git", "-C", f.root, "config", "--get", "branch.created.remote")
				if output, err := cmd.CombinedOutput(); err == nil {
					t.Fatalf("no-tracking remote config=%q", output)
				}
			} else if policy.tracking == git_commands.BranchActionCreateDirectTracking {
				if got := branchActionMergeGit(t, f.root, "config", "--get", "branch.created.remote"); got != "." {
					t.Fatalf("direct tracking remote=%q", got)
				}
				if got := branchActionMergeGit(t, f.root, "config", "--get", "branch.created.merge"); got != "refs/heads/topic" {
					t.Fatalf("direct tracking merge=%q", got)
				}
			} else if got := branchActionMergeGit(t, f.root, "config", "--get", "branch.created.merge"); got != "refs/heads/topic" {
				t.Fatalf("configured tracking merge=%q", got)
			}
			if strings.Contains(strings.Join(ui.commandArgs, " "), "stash") || strings.Contains(strings.Join(ui.commandArgs, " "), "autostash") {
				t.Fatalf("creation invoked stash: %v", ui.commandArgs)
			}
			if len(ui.refresh) != 1 || ui.refresh[0].Then == nil {
				t.Fatal("create completion did not queue refresh")
			}
		})
	}
}

func TestBranchActionCreateFlowFixedNameSkipsPrompt(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	if err := helpers.NewRefsHelper(f.sync.c.HelperCommon, nil).NewBranchWithNameFromRef("refs/heads/topic", "topic", "fixed"); err != nil {
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
	if len(ui.prompts) != 0 || len(f.ui.menus) != 1 {
		t.Fatalf("prompts=%d menus=%d", len(ui.prompts), len(f.ui.menus))
	}
	if !strings.Contains(f.ui.menus[0].Prompt, "fixed") {
		t.Fatalf("fixed name missing from review: %q", f.ui.menus[0].Prompt)
	}
	if f.ui.menus[0].Items[0].CommandID != "branch.create.configured-tracking" {
		t.Fatalf("fixed-name default policy=%q", f.ui.menus[0].Items[0].CommandID)
	}
}

func TestBranchActionCreateFlowCancelInvalidatesCallbacks(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	branchCreateStart(t, f, ui)
	prompt := ui.prompts[0]
	if err := prompt.HandleClose(); err != nil {
		t.Fatal(err)
	}
	if err := prompt.HandleConfirm("late"); err == nil {
		t.Fatal("late prompt callback remained valid")
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "refs", "heads", "late")); !os.IsNotExist(err) {
		t.Fatalf("late branch exists or stat failed: %v", err)
	}
}

func TestBranchActionCreateFlowCancelMenuInvalidatesAndDuplicateCallbacksRunOnce(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchCreateReview(t, f, ui, "created")
	if err := menu.OnCancel(); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, menu, "branch.create.no-tracking").OnPress(); err == nil {
		t.Fatal("cancelled menu callback remained valid")
	}
	if len(ui.blockingWorkers) != 0 {
		t.Fatal("cancelled menu queued a worker")
	}

	f, ui = branchActionMergeFixture(t)
	branchCreateStart(t, f, ui)
	prompt := ui.prompts[0]
	if err := prompt.HandleConfirm("once"); err != nil {
		t.Fatal(err)
	}
	if err := prompt.HandleConfirm("twice"); err == nil {
		t.Fatal("duplicate prompt submission was accepted")
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	menu = f.ui.menus[len(f.ui.menus)-1]
	item := menuItem(t, menu, "branch.create.no-tracking")
	if err := item.OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := item.OnPress(); err == nil {
		t.Fatal("duplicate policy selection was accepted")
	}
	if len(ui.blockingWorkers) != 1 {
		t.Fatalf("blocking workers=%d, want one", len(ui.blockingWorkers))
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchActionMergeGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "once" {
		t.Fatalf("checked out branch=%q", got)
	}
}

func TestBranchActionCreateFlowCarriesDirtyWorktreeAndRejectsIgnoredCollision(t *testing.T) {
	t.Run("dirty success", func(t *testing.T) {
		f, ui := branchActionMergeFixture(t)
		branchActionMergeGit(t, f.root, "checkout", "-q", "topic")
		if err := os.WriteFile(filepath.Join(f.root, "shared"), []byte("shared base"), 0o600); err != nil {
			t.Fatal(err)
		}
		branchActionMergeGit(t, f.root, "add", "shared")
		branchActionMergeGit(t, f.root, "commit", "-qm", "shared file")
		branchActionMergeGit(t, f.root, "checkout", "-q", "main")
		branchActionMergeGit(t, f.root, "merge", "--no-ff", "-qm", "merge topic", "topic")
		if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("dirty bytes\x00\xff"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, "shared"), []byte("dirty shared bytes\x00\xfe"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, "untracked"), []byte("untracked bytes"), 0o600); err != nil {
			t.Fatal(err)
		}
		branchCreateChoose := branchCreateReview(t, f, ui, "dirty-created")
		if err := menuItem(t, branchCreateChoose, "branch.create.no-tracking").OnPress(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runBlockingWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(f.root, "tracked")); err != nil || string(got) != "dirty bytes\x00\xff" {
			t.Fatalf("tracked dirty bytes=%q err=%v", got, err)
		}
		if got, err := os.ReadFile(filepath.Join(f.root, "shared")); err != nil || string(got) != "dirty shared bytes\x00\xfe" {
			t.Fatalf("shared dirty bytes=%q err=%v", got, err)
		}
		if got, err := os.ReadFile(filepath.Join(f.root, "untracked")); err != nil || string(got) != "untracked bytes" {
			t.Fatalf("untracked bytes=%q err=%v", got, err)
		}
	})
	t.Run("ignored collision", func(t *testing.T) {
		f, ui := branchActionMergeFixture(t)
		branchActionMergeGit(t, f.root, "checkout", "-q", "topic")
		if err := os.WriteFile(filepath.Join(f.root, "ignored-incoming"), []byte("source version"), 0o600); err != nil {
			t.Fatal(err)
		}
		branchActionMergeGit(t, f.root, "add", "ignored-incoming")
		branchActionMergeGit(t, f.root, "commit", "-qm", "add incoming")
		branchActionMergeGit(t, f.root, "checkout", "-q", "main")
		branchActionMergeGit(t, f.root, "config", "core.excludesFile", filepath.Join(f.root, "exclude-file"))
		if err := os.WriteFile(filepath.Join(f.root, "exclude-file"), []byte("ignored-incoming\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, "ignored-incoming"), []byte("keep ignored bytes"), 0o600); err != nil {
			t.Fatal(err)
		}
		menu := branchCreateReview(t, f, ui, "collision")
		if err := menuItem(t, menu, "branch.create.no-tracking").OnPress(); err != nil {
			t.Fatal(err)
		}
		err := ui.runBlockingWorker(t)
		if err == nil {
			t.Fatal("ignored collision unexpectedly checked out")
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(f.root, "ignored-incoming")); err != nil || string(got) != "keep ignored bytes" {
			t.Fatalf("ignored bytes=%q err=%v", got, err)
		}
		if ui.subprocessCalls != 0 {
			t.Fatalf("collision command count/args=%d %v", ui.subprocessCalls, ui.commandArgs)
		}
		if got := branchActionMergeGit(t, f.root, "stash", "list"); got != "" {
			t.Fatalf("creation stashed changes: %q", got)
		}
		if got := branchActionMergeGit(t, f.root, "branch", "--list", "collision"); got != "" {
			t.Fatalf("failed checkout left new ref: %q", got)
		}
		if len(ui.refresh) != 1 || ui.refresh[0].BranchSelection != types.KeepBranchSelectionByName {
			t.Fatalf("failed creation refresh=%+v", ui.refresh)
		}
		if got := branchActionMergeGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "main" {
			t.Fatalf("failed checkout changed selection to %q", got)
		}
	})
}

func TestBranchActionCreateFlowConfiguredAndDirectTrackingMetadata(t *testing.T) {
	for _, policy := range []struct {
		id         string
		config     bool
		wantRemote string
	}{
		{"configured", true, "."}, {"direct", false, "."},
	} {
		t.Run(policy.id, func(t *testing.T) {
			f, ui := branchActionMergeFixture(t)
			if policy.config {
				branchActionMergeGit(t, f.root, "config", "branch.autoSetupMerge", "always")
			}
			menu := branchCreateReview(t, f, ui, "created")
			id := "branch.create.configured-tracking"
			if !policy.config {
				id = "branch.create.direct-tracking"
			}
			if err := menuItem(t, menu, id).OnPress(); err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if got := branchActionMergeGit(t, f.root, "config", "--get", "branch.created.remote"); got != policy.wantRemote {
				t.Fatalf("tracking remote=%q", got)
			}
			if got := branchActionMergeGit(t, f.root, "config", "--get", "branch.created.merge"); got != "refs/heads/topic" {
				t.Fatalf("tracking merge=%q", got)
			}
			if got := branchActionMergeGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "created" {
				t.Fatalf("HEAD=%q", got)
			}
			if got := branchActionMergeGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
				t.Fatalf("source changed: %s", got)
			}
		})
	}
}

func TestBranchActionCreateFlowNoTrackingAndRefreshFailureSelection(t *testing.T) {
	f, ui := branchActionMergeFixture(t)
	menu := branchCreateReview(t, f, ui, "untracked")
	if err := menuItem(t, menu, "branch.create.no-tracking").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", f.root, "config", "--get", "branch.untracked.remote")
	if output, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("no-tracking remote config=%q", output)
	}
}
