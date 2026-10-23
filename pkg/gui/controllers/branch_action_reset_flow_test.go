package controllers

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchActionResetContexts struct{ tree *guicontext.ContextTree }

func (contexts branchActionResetContexts) Contexts() *guicontext.ContextTree { return contexts.tree }

type branchActionResetUI struct{ *branchSyncCurrentPullUI }

func (ui *branchActionResetUI) LogAction(string) {}

func branchActionResetFixture(t *testing.T) (*branchSyncPushFixture, *branchSyncCurrentPullUI, *BranchesController) {
	t.Helper()
	f, ui := newBranchSyncCurrentPullFixture(t)
	f.sync.c.HelperCommon.IGuiCommon = &branchActionResetUI{ui}
	localCommits := &guicontext.LocalCommitsContext{LocalCommitsViewModel: &guicontext.LocalCommitsViewModel{}}
	f.sync.c.HelperCommon.IGetContexts = branchActionResetContexts{tree: &guicontext.ContextTree{LocalCommits: localCommits}}
	f.sync.c.IGetHelpers = branchSyncCurrentPullHelpers{value: &helpers.Helpers{}}
	return f, ui, &BranchesController{c: f.sync.c}
}

func branchActionResetOpen(t *testing.T, f *branchSyncPushFixture, ui *branchSyncCurrentPullUI, controller *BranchesController, model *models.Branch) types.CreateMenuOptions {
	t.Helper()
	if err := controller.createResetMenu(model); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 {
		t.Fatalf("reset menu count = %d", len(f.ui.menus))
	}
	return f.ui.menus[0]
}

func branchActionResetChoose(t *testing.T, ui *branchSyncCurrentPullUI, menu types.CreateMenuOptions, id string) {
	t.Helper()
	if err := menuItem(t, menu, id).OnPress(); err != nil {
		t.Fatal(err)
	}
	if id == "branch.reset.hard" {
		return
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
}

func TestBranchActionResetFlowCapturesSourceAndHeadWithoutInspectionWrites(t *testing.T) {
	f, ui, controller := branchActionResetFixture(t)
	model := &models.Branch{Name: "topic"}
	initialHead := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	if err := controller.createResetMenu(model); err != nil {
		t.Fatal(err)
	}
	model.Name = "main"
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	menu := f.ui.menus[0]
	if len(ui.blockingWorkers) != 0 || ui.subprocessCalls != 0 {
		t.Fatal("inspection performed mutation work")
	}
	if !strings.Contains(menu.Prompt, "topic") || !strings.Contains(menu.Prompt, initialHead) || !strings.Contains(menu.Prompt, f.sourceOID) {
		t.Fatalf("review omitted captured identities: %q", menu.Prompt)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != initialHead {
		t.Fatal("inspection moved HEAD")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatal("inspection changed selected source")
	}
}

func TestBranchActionResetFlowNativeUpstream(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(map[bool]string{false: "custom tracking namespace", true: "local upstream"}[local], func(t *testing.T) {
			f, ui, _ := branchActionResetFixture(t)
			f.ui.model.CheckedOutBranch = "main"
			ref := "refs/company/review/topic"
			remote := "origin"
			if local {
				ref, remote = "refs/heads/topic", "."
			} else {
				branchSyncPushGit(t, f.root, "config", "--replace-all", "remote.origin.fetch", "+refs/heads/*:refs/company/review/*")
				branchSyncPushGit(t, f.root, "update-ref", ref, f.sourceOID)
			}
			branchSyncPushGit(t, f.root, "config", "branch.main.remote", remote)
			branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/topic")
			if err := helpers.ResetCurrentToCapturedSource(f.sync.c.HelperCommon, "@{upstream}", "@{upstream}"); err != nil {
				t.Fatal(err)
			}
			// A later cached model refresh must not choose a different branch.
			f.ui.model.CheckedOutBranch = "topic"
			if err := ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			menu := f.ui.menus[0]
			if !strings.Contains(menu.Prompt, ref) || !strings.Contains(menu.Prompt, f.sourceOID) {
				t.Fatalf("review omitted native upstream: %q", menu.Prompt)
			}
			branchActionResetChoose(t, ui, menu, "branch.reset.soft")
			if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != f.sourceOID {
				t.Fatalf("HEAD = %s, want captured upstream %s", got, f.sourceOID)
			}
		})
	}
}

func TestBranchActionResetFlowUpstreamRejectsCheckoutBeforeInspection(t *testing.T) {
	f, ui, _ := branchActionResetFixture(t)
	f.ui.model.CheckedOutBranch = "main"
	branchSyncPushGit(t, f.root, "config", "branch.main.remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/topic")
	if err := helpers.ResetCurrentToCapturedSource(f.sync.c.HelperCommon, "@{upstream}", "@{upstream}"); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	if err := ui.runWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("changed checkout error = %v", err)
	}
	if len(f.ui.menus) != 0 || len(ui.blockingWorkers) != 0 {
		t.Fatal("changed checkout opened or executed reset")
	}
}

func TestBranchActionResetFlowSoftPreservesRawDirtyStateAndSourceRef(t *testing.T) {
	f, ui, controller := branchActionResetFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged\x00\xff"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("work\x00\xfe"), 0o600); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	work, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "topic"})
	branchActionResetChoose(t, ui, menu, "branch.reset.soft")
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != f.sourceOID {
		t.Fatalf("soft reset HEAD %s", got)
	}
	gotIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	gotWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(index, gotIndex) || !bytes.Equal(work, gotWork) {
		t.Fatal("soft reset changed dirty index/worktree bytes")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatal("soft reset changed selected source ref")
	}
	if ui.refreshes != 1 {
		t.Fatalf("refresh count = %d, want one captured repository refresh", ui.refreshes)
	}
}

func TestBranchActionResetFlowHardAlwaysConfirmsAndCancellationDoesNotWrite(t *testing.T) {
	f, ui, controller := branchActionResetFixture(t)
	menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "topic"})
	if err := menuItem(t, menu, "branch.reset.hard").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.confirms) != 1 || len(ui.blockingWorkers) != 0 {
		t.Fatal("clean hard reset must show confirmation before work")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") {
		t.Fatal("confirmation changed HEAD")
	}
	if err := ui.confirms[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if len(ui.blockingWorkers) != 0 || ui.refreshes != 0 {
		t.Fatal("cancel queued reset or refresh")
	}
}

func TestBranchActionResetFlowHardIgnoredCollisionWaitsForConfirmation(t *testing.T) {
	f, ui, controller := branchActionResetFixture(t)
	ignoreFile := filepath.Join(f.root, "global-ignore")
	if err := os.WriteFile(ignoreFile, []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "config", "core.excludesFile", ignoreFile)
	if err := os.WriteFile(filepath.Join(f.root, ".gitignore"), []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "ignored"), []byte("committed destination"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", ".gitignore")
	branchSyncPushGit(t, f.root, "add", "-f", "ignored")
	branchSyncPushGit(t, f.root, "commit", "-qm", "ignored destination")
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "ignored"), []byte("collision bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "main"})
	if err := menuItem(t, menu, "branch.reset.hard").OnPress(); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "ignored")); err != nil || string(got) != "collision bytes" {
		t.Fatalf("inspection lost ignored collision: %q %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "tracked")); err != nil || string(got) != "unstaged bytes" {
		t.Fatalf("confirmation changed dirty worktree bytes: %q %v", got, err)
	}
	indexPreConfirm, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil || !bytes.Equal(indexBefore, indexPreConfirm) {
		t.Fatal("confirmation changed the staged index")
	}
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") {
		t.Fatal("hard reset did not move current HEAD to captured destination")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main"); got != branchSyncPushGit(t, f.root, "rev-parse", "HEAD") {
		t.Fatal("hard reset changed the selected destination ref")
	}
	got, err := os.ReadFile(filepath.Join(f.root, "ignored"))
	if err != nil || string(got) != "committed destination" {
		t.Fatalf("confirmed hard reset result = %q, %v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil || string(got) != "base" {
		t.Fatalf("hard reset worktree result = %q, %v", got, err)
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(indexBefore, indexAfter) {
		t.Fatal("hard reset did not replace staged index state")
	}
}

func TestBranchActionResetFlowRejectsChangedSourceOrHeadBeforeLaunch(t *testing.T) {
	for _, change := range []string{"source", "head tip", "same oid head identity"} {
		t.Run(change, func(t *testing.T) {
			f, ui, controller := branchActionResetFixture(t)
			menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "topic"})
			if err := menuItem(t, menu, "branch.reset.soft").OnPress(); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source":
				branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main"))
			case "head tip":
				branchSyncPushGit(t, f.root, "update-ref", "refs/heads/main", f.sourceOID)
			case "same oid head identity":
				branchSyncPushGit(t, f.root, "branch", "other-head", "HEAD")
				branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD", "refs/heads/other-head")
			}
			beforeHead := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
			beforeSymbolicHead := branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD")
			beforeIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			beforeWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
				t.Fatalf("stale prepared reset error = %v, want ErrBranchActionChanged", err)
			}
			if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != beforeHead {
				t.Fatal("stale reset moved HEAD")
			}
			if got := branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD"); got != beforeSymbolicHead {
				t.Fatal("stale reset changed HEAD identity")
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
				t.Fatal("stale reset changed raw index/worktree bytes")
			}
			if change == "source" && branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic") != branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") {
				t.Fatal("stale source changed during reset")
			}
			if change == "same oid head identity" && beforeHead != branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") {
				t.Fatal("same-OID HEAD identity setup changed the commit")
			}
		})
	}
}

func TestBranchActionResetFlowHardApprovalRejectsChangedSourceAndHead(t *testing.T) {
	for _, change := range []string{"source tip", "head tip", "same oid head identity"} {
		t.Run(change, func(t *testing.T) {
			f, ui, controller := branchActionResetFixture(t)
			menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "topic"})
			if err := menuItem(t, menu, "branch.reset.hard").OnPress(); err != nil {
				t.Fatal(err)
			}
			approval := ui.confirms[len(ui.confirms)-1]
			if err := approval.HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if err := approval.HandleConfirm(); err == nil {
				t.Fatal("consumed hard approval was replayed")
			}
			switch change {
			case "source tip":
				branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main"))
			case "head tip":
				branchSyncPushGit(t, f.root, "update-ref", "refs/heads/main", f.sourceOID)
			case "same oid head identity":
				branchSyncPushGit(t, f.root, "branch", "other-head", "HEAD")
				branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD", "refs/heads/other-head")
			}
			beforeHead := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
			beforeSymbolicHead := branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD")
			beforeIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			beforeWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
				t.Fatalf("hard stale reset error = %v, want ErrBranchActionChanged", err)
			}
			if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != beforeHead {
				t.Fatal("stale hard reset moved HEAD")
			}
			if got := branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD"); got != beforeSymbolicHead {
				t.Fatal("stale hard reset changed HEAD identity")
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
				t.Fatal("stale hard reset changed raw index/worktree bytes")
			}
			if ui.refreshes != 0 {
				t.Fatal("stale hard approval refreshed after no reset")
			}
		})
	}
}

func TestBranchActionResetFlowRepositoryChangesBlockInspectionUIAndPrelaunch(t *testing.T) {
	for _, boundary := range []string{"inspection worker pointer", "queued UI pointer", "queued UI generation", "prelaunch pointer", "prelaunch generation"} {
		t.Run(boundary, func(t *testing.T) {
			f, ui, controller := branchActionResetFixture(t)
			other, _, _ := branchActionResetFixture(t)
			otherHead := branchSyncPushGit(t, other.root, "rev-parse", "HEAD")
			otherIndex, err := os.ReadFile(filepath.Join(other.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			otherWork, err := os.ReadFile(filepath.Join(other.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}
			if err := controller.createResetMenu(&models.Branch{Name: "topic"}); err != nil {
				t.Fatal(err)
			}
			if boundary == "inspection worker pointer" {
				ui.git = other.git
				if err := ui.runWorker(t); err == nil {
					t.Fatal("stale inspection worker accepted repository change")
				}
				if len(ui.ui) != 0 || len(ui.menus) != 0 {
					t.Fatal("stale inspection queued menu/UI")
				}
				branchActionResetAssertOtherRepo(t, other, otherHead, otherIndex, otherWork)
				return
			}
			if err := ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(boundary, "queued UI") {
				if strings.HasSuffix(boundary, "pointer") {
					ui.git = other.git
				} else {
					f.ui.state.generation++
				}
				if err := ui.runUI(t); err == nil {
					t.Fatal("stale UI callback accepted generation change")
				}
				if len(ui.menus) != 0 {
					t.Fatal("stale UI callback opened reset menu")
				}
				branchActionResetAssertOtherRepo(t, other, otherHead, otherIndex, otherWork)
				return
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if err := menuItem(t, ui.menus[0], "branch.reset.soft").OnPress(); err != nil {
				t.Fatal(err)
			}
			if strings.HasSuffix(boundary, "pointer") {
				ui.git = other.git
			} else {
				f.ui.state.generation++
			}
			if err := ui.runBlockingWorker(t); err == nil {
				t.Fatal("stale prelaunch worker accepted generation change")
			}
			if ui.refreshes != 0 {
				t.Fatal("stale prelaunch worker refreshed a repository")
			}
		})
	}
}

func branchActionResetAssertOtherRepo(t *testing.T, f *branchSyncPushFixture, head string, index, expectedWork []byte) {
	t.Helper()
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != head {
		t.Fatal("replacement repository HEAD changed")
	}
	got, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, index) {
		t.Fatal("replacement repository index changed")
	}
	actualWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil || !bytes.Equal(actualWork, expectedWork) {
		t.Fatalf("replacement repository worktree changed: %q %v", actualWork, err)
	}
}

func TestBranchActionResetFlowConsumedCallbackCannotReplay(t *testing.T) {
	f, ui, controller := branchActionResetFixture(t)
	menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "main"})
	item := menuItem(t, menu, "branch.reset.soft")
	if err := item.OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.blockingWorkers) != 1 {
		t.Fatal("initial reset did not queue one worker")
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := item.OnPress(); err == nil {
		t.Fatal("consumed menu callback was replayed")
	}
	if len(ui.blockingWorkers) != 0 {
		t.Fatal("replayed menu callback queued another reset")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main") {
		t.Fatalf("same-tip reset changed HEAD unexpectedly: %s", got)
	}
}

func TestBranchActionResetFlowPostlaunchRepositoryChangeBlocksRefresh(t *testing.T) {
	for _, change := range []string{"pointer", "generation"} {
		t.Run(change, func(t *testing.T) {
			f, ui, controller := branchActionResetFixture(t)
			other, _, _ := branchActionResetFixture(t)
			otherHead := branchSyncPushGit(t, other.root, "rev-parse", "HEAD")
			otherIndex, err := os.ReadFile(filepath.Join(other.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			otherWork, err := os.ReadFile(filepath.Join(other.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}
			menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "topic"})
			if err := menuItem(t, menu, "branch.reset.soft").OnPress(); err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); err != nil {
				t.Fatal(err)
			}
			if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != f.sourceOID {
				t.Fatal("native reset result was not retained")
			}
			if change == "pointer" {
				ui.git = other.git
			} else {
				f.ui.state.generation++
			}
			if len(ui.ui) != 1 {
				t.Fatal("expected queued postlaunch refresh")
			}
			if err := ui.runUI(t); err == nil {
				t.Fatal("stale postlaunch refresh accepted repository change")
			}
			if ui.refreshes != 0 {
				t.Fatal("stale callback refreshed the replacement repository")
			}
			if change == "pointer" {
				branchActionResetAssertOtherRepo(t, other, otherHead, otherIndex, otherWork)
			}
		})
	}
}

func TestBranchActionResetFlowPreservesNativeIndexLockFailure(t *testing.T) {
	f, ui, controller := branchActionResetFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("dirty bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("work bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	workBefore, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(f.root, ".git", "index.lock")
	if err := os.WriteFile(lock, []byte("owned lock"), 0o600); err != nil {
		t.Fatal(err)
	}
	menu := branchActionResetOpen(t, f, ui, controller, &models.Branch{Name: "topic"})
	if err := menuItem(t, menu, "branch.reset.mixed").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil || !strings.Contains(strings.ToLower(err.Error()), "lock") {
		t.Fatalf("native index.lock error not preserved: %v", err)
	}
	lockBytes, err := os.ReadFile(lock)
	if err != nil || string(lockBytes) != "owned lock" {
		t.Fatalf("native lock was removed or repaired: %q %v", lockBytes, err)
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	workAfter, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(indexBefore, indexAfter) || !bytes.Equal(workBefore, workAfter) {
		t.Fatal("failed reset changed raw dirty state")
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("reset failure retried through another subprocess path")
	}
}
