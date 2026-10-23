package controllers

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/app/daemon"
	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchActionRebaseUI struct {
	*branchSyncCurrentPullUI
	modes          *types.Modes
	ctx            *guicontext.ContextTree
	refresh        []types.RefreshOptions
	logs           int
	stack          *branchActionRebaseContextStack
	commandEnv     []string
	gitReplacement *commands.GitCommand
	commandArgs    []string
	nativeErr      error
}

func (ui *branchActionRebaseUI) Modes() *types.Modes        { return ui.modes }
func (ui *branchActionRebaseUI) Context() types.IContextMgr { return ui.stack }
func (ui *branchActionRebaseUI) Git() *commands.GitCommand {
	if ui.gitReplacement != nil {
		return ui.gitReplacement
	}
	return ui.branchSyncCurrentPullUI.Git()
}
func (ui *branchActionRebaseUI) LogAction(string)                { ui.logs++ }
func (ui *branchActionRebaseUI) PostRefreshUpdate(types.Context) {}
func (ui *branchActionRebaseUI) RunSubprocess(cmd *oscommands.CmdObj) (bool, error) {
	ui.subprocessCalls++
	ui.commandEnv = append([]string(nil), cmd.GetEnvVars()...)
	ui.commandArgs = append([]string(nil), cmd.GetCmd().Args...)
	cmd.RemoveEnvVar("GIT_SEQUENCE_EDITOR")
	cmd.AddEnvVars("GIT_SEQUENCE_EDITOR=true")
	err := cmd.Run()
	ui.nativeErr = err
	return true, err
}

func (ui *branchActionRebaseUI) Refresh(opts types.RefreshOptions) {
	ui.refresh = append(ui.refresh, opts)
}

// This adapter deliberately models the context stack boundary used by rebase completion.
type branchActionRebaseContextStack struct {
	types.IContextMgr
	pushed int
}

func (stack *branchActionRebaseContextStack) Push(types.Context, types.OnFocusOpts) { stack.pushed++ }

func branchActionRebaseFixture(t *testing.T) (*branchSyncPushFixture, *branchActionRebaseUI) {
	t.Helper()
	f, base := newBranchSyncCurrentPullFixture(t)
	local := &guicontext.LocalCommitsContext{LocalCommitsViewModel: &guicontext.LocalCommitsViewModel{}}
	ui := &branchActionRebaseUI{branchSyncCurrentPullUI: base, modes: &types.Modes{}, ctx: &guicontext.ContextTree{LocalCommits: local}, stack: &branchActionRebaseContextStack{}}
	f.sync.c.HelperCommon.IGuiCommon = ui
	f.sync.c.HelperCommon.IGetContexts = branchActionResetContexts{tree: ui.ctx}
	f.sync.c.IGetHelpers = branchSyncCurrentPullHelpers{value: &helpers.Helpers{MergeAndRebase: helpers.NewMergeAndRebaseHelper(f.sync.c.HelperCommon)}}
	return f, ui
}

func branchActionRebaseRepositoryUnchanged(t *testing.T, f *branchSyncPushFixture) func() {
	t.Helper()
	before := branchSyncCurrentPullReadSnapshot(t, f)
	headBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		t.Helper()
		after := branchSyncCurrentPullReadSnapshot(t, f)
		headAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "HEAD"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(headBefore, headAfter) || !bytes.Equal(before.head, after.head) || !bytes.Equal(before.index, after.index) || !bytes.Equal(before.tracked, after.tracked) {
			t.Fatalf("repository %s changed HEAD, raw index or worktree bytes after stale callback", f.root)
		}
	}
}

func branchActionRebaseReplaceRepository(t *testing.T, ui *branchActionRebaseUI) func() {
	t.Helper()
	replacement := newBranchSyncPushFixture(t)
	if err := os.WriteFile(filepath.Join(replacement.root, "tracked"), []byte("replacement staged\x00\xff"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, replacement.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(replacement.root, "tracked"), []byte("replacement work\x00\xfe"), 0o600); err != nil {
		t.Fatal(err)
	}
	checkUnchanged := branchActionRebaseRepositoryUnchanged(t, replacement)
	ui.gitReplacement = replacement.git
	return checkUnchanged
}

func branchActionRebaseStart(t *testing.T, f *branchSyncPushFixture, ui *branchActionRebaseUI) types.CreateMenuOptions {
	t.Helper()
	if err := helpers.RebaseCurrentOntoCapturedRef(f.sync.c.HelperCommon, "refs/heads/topic"); err != nil {
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

func branchActionRebaseChoose(t *testing.T, ui *branchActionRebaseUI, menu types.CreateMenuOptions, id string) {
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

func branchActionRebaseGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	return branchSyncPushGit(t, root, args...)
}

func TestBranchActionRebaseFlowCapturedRefAndModes(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	// A separate branch is the literal source; the current branch is main.
	branchActionRebaseGit(t, f.root, "checkout", "-q", "topic")
	branchActionRebaseGit(t, f.root, "checkout", "-q", "main")
	f.ui.model.MainBranches = nil
	ui.modes.MarkedBaseCommit.SetHash(f.sourceOID)
	menu := branchActionRebaseStart(t, f, ui)
	head := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD")
	if !strings.Contains(menu.Prompt, `"main"`) || !strings.Contains(menu.Prompt, head) || !strings.Contains(menu.Prompt, "refs/heads/topic") || !strings.Contains(menu.Prompt, f.sourceOID) {
		t.Fatalf("menu prompt omitted captured source/current identities: %q", menu.Prompt)
	}
	if !strings.Contains(menu.Prompt, "Marked range base:") || !strings.Contains(menu.Prompt, f.sourceOID) {
		t.Fatalf("menu prompt omitted captured marked base: %q", menu.Prompt)
	}
	if len(menu.Items) != 3 || menu.Items[0].CommandID != "branch.rebase-current-onto-selected.simple" || menu.Items[1].CommandID != "branch.rebase-current-onto-selected.interactive" || menu.Items[2].CommandID != "branch.rebase-current-onto-base" {
		t.Fatalf("unexpected rebase menu IDs: %+v", menu.Items)
	}
	if menu.Items[2].DisabledReason == nil {
		t.Fatal("base choice should start disabled when main branches are unavailable")
	}
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	confirmation := ui.confirms[0]
	if !strings.Contains(confirmation.Prompt, head) || !strings.Contains(confirmation.Prompt, "refs/heads/topic") || !strings.Contains(confirmation.Prompt, f.sourceOID) {
		t.Fatalf("confirmation omitted captured identities: %q", confirmation.Prompt)
	}
	if err := ui.confirms[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("closing confirmation ran rebase")
	}
}

func TestBranchActionRebaseFlowRebasesOnlyCurrentAndPreservesOtherRefs(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	// Add a unique current-branch commit so this verifies a replay, not only a fast-forward.
	if err := os.WriteFile(filepath.Join(f.root, "current-only"), []byte("local commit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "add", "current-only")
	branchActionRebaseGit(t, f.root, "commit", "-qm", "current only")
	initialHead := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD")
	other := branchActionRebaseGit(t, f.root, "rev-parse", "refs/heads/topic")
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.ui) != 1 {
		t.Fatalf("terminal callbacks=%d", len(ui.ui))
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
	if got := branchActionRebaseGit(t, f.root, "rev-parse", "refs/heads/topic"); got != other {
		t.Fatalf("unrelated source ref changed: %s", got)
	}
	if got := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD"); got == initialHead {
		t.Fatal("rebase did not move current branch")
	}
	if got := branchActionRebaseGit(t, f.root, "show", "-s", "--format=%s", "HEAD"); got != "current only" {
		t.Fatalf("rebased current tip subject=%q", got)
	}
	if len(ui.refresh) != 1 {
		t.Fatalf("refreshes=%d", len(ui.refresh))
	}
	joinedArgs := strings.Join(ui.commandArgs, " ")
	if !strings.Contains(joinedArgs, "--no-autostash") || strings.Contains(joinedArgs, "git stash") {
		t.Fatalf("rebase command did not explicitly avoid autostash: %s", joinedArgs)
	}
	if ui.refresh[0].Then != nil {
		if err := ui.refresh[0].Then(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBranchActionRebaseFlowInteractiveChoiceUsesCapturedPlan(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.interactive")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
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
		t.Fatal("interactive rebase did not queue its refresh completion")
	}
	if err := ui.refresh[0].Then(); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 1 || ui.stack.pushed != 1 {
		t.Fatalf("interactive terminal calls=%d context pushes=%d", ui.subprocessCalls, ui.stack.pushed)
	}
	if len(ui.commandEnv) == 0 {
		t.Fatal("interactive command's original environment was not captured before the editor override")
	}
	var sawBreakInstruction bool
	for _, entry := range ui.commandEnv {
		if strings.HasPrefix(entry, "GIT_SEQUENCE_EDITOR=") && !strings.HasSuffix(entry, "=true") {
			sawBreakInstruction = true
		}
	}
	if !sawBreakInstruction {
		t.Fatalf("original interactive-break sequence editor missing before test override: %v", ui.commandEnv)
	}
	for _, env := range daemon.ToEnvVars(daemon.NewInsertBreakInstruction()) {
		if !slices.Contains(ui.commandEnv, env) {
			t.Fatalf("original interactive-break env %q missing: %v", env, ui.commandEnv)
		}
	}
}

func TestBranchActionRebaseFlowRejectsChangedMarkedBaseAndReplay(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	ui.modes.MarkedBaseCommit.SetHash(f.sourceOID)
	if err := ui.confirms[0].HandleConfirm(); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("marked base change error=%v", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("stale marked base launched rebase")
	}
}

func TestBranchActionRebaseFlowRejectsRepositorySwapBeforeMenu(t *testing.T) {
	for _, swap := range []string{"generation", "git-pointer"} {
		t.Run(swap, func(t *testing.T) {
			f, ui := branchActionRebaseFixture(t)
			if err := helpers.RebaseCurrentOntoCapturedRef(f.sync.c.HelperCommon, "refs/heads/topic"); err != nil {
				t.Fatal(err)
			}
			if err := ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
			defer checkCaptured()
			if swap == "generation" {
				f.ui.state.generation++
			} else {
				checkReplacement := branchActionRebaseReplaceRepository(t, ui)
				defer checkReplacement()
			}
			if err := ui.runUI(t); err == nil {
				t.Fatal("stale menu callback accepted replacement repository")
			}
			if len(f.ui.menus) != 0 || ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
				t.Fatal("stale menu callback presented or ran work")
			}
		})
	}
}

func TestBranchActionRebaseFlowRepositorySwapBeforeInspection(t *testing.T) {
	for _, swap := range []string{"generation", "git-pointer"} {
		t.Run(swap, func(t *testing.T) {
			f, ui := branchActionRebaseFixture(t)
			if err := helpers.RebaseCurrentOntoCapturedRef(f.sync.c.HelperCommon, "refs/heads/topic"); err != nil {
				t.Fatal(err)
			}
			checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
			defer checkCaptured()
			if swap == "generation" {
				f.ui.state.generation++
			} else {
				checkReplacement := branchActionRebaseReplaceRepository(t, ui)
				defer checkReplacement()
			}
			if err := ui.runWorker(t); err == nil {
				t.Fatal("inspection accepted replacement repository")
			}
			if len(f.ui.menus) != 0 || ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
				t.Fatal("stale inspection presented UI or ran work")
			}
		})
	}
}

func TestBranchActionRebaseFlowRejectsChangedSourceAtPreflight(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", "refs/heads/main")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("preflight error=%v, want stale captured source", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("changed source launched terminal rebase")
	}
	if got := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD"); got != branchActionRebaseGit(t, f.root, "rev-parse", "refs/heads/main") {
		t.Fatal("failed preflight changed current HEAD")
	}
}

func TestBranchActionRebaseFlowRejectsChangedSourceAtTerminalBoundary(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", "refs/heads/main")
	if err := ui.runUI(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("terminal boundary error=%v, want stale captured source", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("stale terminal callback launched rebase")
	}
}

func TestBranchActionRebaseFlowRejectsHeadIdentityChangeAtPreflight(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	branchActionRebaseGit(t, f.root, "branch", "same-tip")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "checkout", "-q", "same-tip")
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("same-OID HEAD identity error=%v", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("changed HEAD identity launched rebase")
	}
}

func TestBranchActionRebaseFlowRejectsCurrentTipChangeAtPreflight(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "update-ref", "refs/heads/main", "refs/heads/topic")
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("changed current tip error=%v", err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("changed current tip launched rebase")
	}
}

func TestBranchActionRebaseFlowRepositorySwapAtPreflightAndTerminal(t *testing.T) {
	for _, boundary := range []string{"preflight", "terminal"} {
		for _, swap := range []string{"generation", "git-pointer"} {
			t.Run(boundary+"/"+swap, func(t *testing.T) {
				f, ui := branchActionRebaseFixture(t)
				menu := branchActionRebaseStart(t, f, ui)
				branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
				if err := ui.confirms[0].HandleConfirm(); err != nil {
					t.Fatal(err)
				}
				if boundary == "terminal" {
					if err := ui.runBlockingWorker(t); err != nil {
						t.Fatal(err)
					}
				}
				checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
				defer checkCaptured()
				if swap == "generation" {
					f.ui.state.generation++
				} else {
					checkReplacement := branchActionRebaseReplaceRepository(t, ui)
					defer checkReplacement()
				}
				if boundary == "preflight" {
					if err := ui.runBlockingWorker(t); err == nil {
						t.Fatal("preflight accepted replacement repository")
					}
				} else if err := ui.runUI(t); err == nil {
					t.Fatal("terminal callback accepted replacement repository")
				}
				if ui.subprocessCalls != 0 || len(ui.refresh) != 0 {
					t.Fatal("repository swap launched or refreshed work")
				}
			})
		}
	}
}

func TestBranchActionRebaseFlowConsumesConfirmationOnce(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	confirmation := ui.confirms[0]
	if err := confirmation.HandleConfirm(); err != nil {
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
	if err := confirmation.HandleConfirm(); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("replayed confirmation error=%v", err)
	}
	if ui.subprocessCalls != 1 || len(ui.blockingWorkers) != 0 {
		t.Fatal("replayed confirmation queued another execution")
	}
}

func TestBranchActionRebaseFlowRepositorySwapBeforeFinish(t *testing.T) {
	for _, swap := range []string{"generation", "git-pointer"} {
		t.Run(swap, func(t *testing.T) {
			f, ui := branchActionRebaseFixture(t)
			menu := branchActionRebaseStart(t, f, ui)
			branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
			if err := ui.confirms[0].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if err := ui.runBlockingWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
			defer checkCaptured()
			if swap == "generation" {
				f.ui.state.generation++
			} else {
				checkReplacement := branchActionRebaseReplaceRepository(t, ui)
				defer checkReplacement()
			}
			if err := ui.runBlockingWorker(t); err == nil {
				t.Fatal("finish accepted replacement repository")
			}
			if len(ui.refresh) != 0 || ui.subprocessCalls != 1 {
				t.Fatal("stale finish refreshed replacement repository or launched another subprocess")
			}
		})
	}
}

func TestBranchActionRebaseFlowRepositorySwapBeforeRefreshThen(t *testing.T) {
	for _, swap := range []string{"generation", "git-pointer"} {
		t.Run(swap, func(t *testing.T) {
			f, ui := branchActionRebaseFixture(t)
			menu := branchActionRebaseStart(t, f, ui)
			branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
			if err := ui.confirms[0].HandleConfirm(); err != nil {
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
				t.Fatal("missing queued refresh completion")
			}
			checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
			defer checkCaptured()
			if swap == "generation" {
				f.ui.state.generation++
			} else {
				checkReplacement := branchActionRebaseReplaceRepository(t, ui)
				defer checkReplacement()
			}
			if err := ui.refresh[0].Then(); err == nil {
				t.Fatal("refresh completion accepted replacement repository")
			}
			if len(ui.refresh) != 1 || ui.subprocessCalls != 1 {
				t.Fatal("stale refresh completion refreshed replacement repository or launched another subprocess")
			}
			if ui.stack.pushed != 0 {
				t.Fatal("stale completion changed context stack")
			}
		})
	}
}

func TestBranchActionRebaseFlowNativeIndexLockErrorIsPreserved(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(f.root, ".git", "index.lock")
	if err := os.WriteFile(lock, []byte("held"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(lock)
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil || !strings.Contains(strings.ToLower(err.Error()), "lock") {
		t.Fatalf("native index-lock error=%v", err)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("native attempts=%d, want one", ui.subprocessCalls)
	}
}

func TestBranchActionRebaseFlowSigningFailureLeavesExplicitPausedRebase(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	base := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(f.root, "current-only"), []byte("replayed commit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "add", "current-only")
	branchActionRebaseGit(t, f.root, "commit", "-qm", "unsigned setup commit")
	currentTip := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD")
	topicTip := branchActionRebaseGit(t, f.root, "rev-parse", "refs/heads/topic")
	branchActionRebaseGit(t, f.root, "config", "commit.gpgSign", "true")
	branchActionRebaseGit(t, f.root, "config", "gpg.format", "openpgp")
	branchActionRebaseGit(t, f.root, "config", "gpg.program", "false")
	ui.modes.MarkedBaseCommit.SetHash(base)
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err == nil || !strings.Contains(strings.ToLower(err.Error()), "rebase is paused") {
		t.Fatalf("signing failure result=%v, want paused rebase message", err)
	} else if !errors.Is(err, ui.nativeErr) {
		t.Fatalf("wrapped result %v does not preserve native error %v", err, ui.nativeErr)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.nativeErr == nil {
		t.Fatal("native signing failure was not recorded")
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("native attempts=%d, want exactly one", ui.subprocessCalls)
	}
	if rebasing, err := f.git.Status.IsInRebase(); err != nil || !rebasing {
		t.Fatalf("rebase active=%v err=%v", rebasing, err)
	}
	if unmerged, err := f.git.Rebase.HasUnmergedFiles(); err != nil || unmerged {
		t.Fatalf("unmerged=%v err=%v; signing failure should pause without conflicts", unmerged, err)
	}
	if got := branchActionRebaseGit(t, f.root, "rev-parse", "refs/heads/main"); got != currentTip {
		t.Fatalf("current branch ref moved despite signing failure: %s", got)
	}
	if got := branchActionRebaseGit(t, f.root, "rev-parse", "refs/heads/topic"); got != topicTip {
		t.Fatalf("unrelated selected ref changed: %s", got)
	}
	state := ui.branchSyncCurrentPullUI.state.(*branchSyncCurrentPullState)
	if !state.repo.started {
		t.Fatal("repo operation state did not record active paused rebase")
	}
	if len(ui.refresh) == 0 || ui.refresh[len(ui.refresh)-1].Then == nil {
		t.Fatal("paused rebase did not queue refresh completion")
	}
	if err := ui.refresh[len(ui.refresh)-1].Then(); err != nil {
		t.Fatal(err)
	}
	if ui.modes.MarkedBaseCommit.GetHash() != "" {
		t.Fatal("marked base was retained after Git started the reviewed rebase")
	}
	joinedArgs := strings.Join(ui.commandArgs, " ")
	if strings.Contains(joinedArgs, "--skip") || strings.Contains(joinedArgs, "--continue") {
		t.Fatalf("rebase silently chose a recovery action: %s", joinedArgs)
	}
}

func branchActionRebaseConflictMenu(t *testing.T) (*branchSyncPushFixture, *branchActionRebaseUI, string, types.CreateMenuOptions) {
	t.Helper()
	f, ui := branchActionRebaseFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("main conflicting change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "commit", "-qam", "main conflicting change")
	mainTip := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD")
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
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
	if len(ui.refresh) > 0 && ui.refresh[len(ui.refresh)-1].Then != nil {
		if err := ui.refresh[len(ui.refresh)-1].Then(); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.ui.menus) < 2 {
		t.Fatal("rebase conflict did not open the captured conflict menu after refresh")
	}
	conflict := f.ui.menus[len(f.ui.menus)-1]
	return f, ui, mainTip, conflict
}

func TestBranchActionRebaseFlowConflictOffersCapturedAbort(t *testing.T) {
	f, ui, mainTip, conflict := branchActionRebaseConflictMenu(t)
	if menuItem(t, conflict, "operation.abort") == nil {
		t.Fatal("captured abort missing")
	}
	if err := menuItem(t, conflict, "operation.abort").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.refresh) < 2 {
		t.Fatal("abort did not refresh captured repository")
	}
	if got := branchActionRebaseGit(t, f.root, "rev-parse", "HEAD"); got != mainTip {
		t.Fatalf("abort HEAD=%s, want original %s", got, mainTip)
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "rebase-merge")); err == nil {
		t.Fatal("abort left rebase in progress")
	}
}

func TestBranchActionRebaseFlowAbortRefusesMismatchedMetadata(t *testing.T) {
	f, ui, _, conflict := branchActionRebaseConflictMenu(t)
	metadata := filepath.Join(f.root, ".git", "rebase-merge", "head-name")
	if err := os.WriteFile(metadata, []byte("refs/heads/replaced\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, conflict, "operation.abort").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("replaced operation abort error=%v", err)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("subprocess attempts=%d, want only original rebase", ui.subprocessCalls)
	}
}

func TestBranchActionRebaseFlowAbortRefusesActualReplacementOperation(t *testing.T) {
	f, ui, _, conflict := branchActionRebaseConflictMenu(t)
	base := branchActionRebaseGit(t, f.root, "merge-base", "refs/heads/topic", "refs/heads/main")
	branchActionRebaseGit(t, f.root, "rebase", "--abort")
	branchActionRebaseGit(t, f.root, "checkout", "--detach", base)
	branchActionRebaseGit(t, f.root, "checkout", "-qb", "replacement-target")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("replacement target conflict\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "add", "tracked")
	branchActionRebaseGit(t, f.root, "commit", "-qm", "replacement target")
	branchActionRebaseGit(t, f.root, "checkout", "-q", "main")
	if err := exec.Command("git", "-C", f.root, "rebase", "--onto", "replacement-target", base).Run(); err == nil {
		t.Fatal("expected replacement rebase to conflict")
	}
	if err := menuItem(t, conflict, "operation.abort").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("replaced operation abort error=%v", err)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("subprocess attempts=%d, want only original captured rebase", ui.subprocessCalls)
	}
}

func TestBranchActionRebaseFlowAbortRefusesRepositorySwap(t *testing.T) {
	f, ui, _, conflict := branchActionRebaseConflictMenu(t)
	refreshesBefore := len(ui.refresh)
	checkCaptured := branchActionRebaseRepositoryUnchanged(t, f)
	defer checkCaptured()
	checkReplacement := branchActionRebaseReplaceRepository(t, ui)
	defer checkReplacement()
	if err := menuItem(t, conflict, "operation.abort").OnPress(); err == nil {
		t.Fatal("abort callback accepted replacement repository")
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("subprocess attempts=%d, want only original rebase", ui.subprocessCalls)
	}
	if len(ui.refresh) != refreshesBefore || len(ui.blockingWorkers) != 0 {
		t.Fatal("stale abort refreshed replacement repository or queued work")
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "rebase-merge")); err != nil {
		t.Fatal("captured rebase was aborted after repository swap")
	}
}

func TestBranchActionRebaseFlowIgnoredBytesRequireFreshApproval(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	ignore := filepath.Join(f.root, "global-ignore")
	if err := os.WriteFile(ignore, []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "config", "core.excludesFile", ignore)
	branchActionRebaseGit(t, f.root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(f.root, "ignored"), []byte("destination\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "add", "-f", "ignored")
	branchActionRebaseGit(t, f.root, "commit", "-qm", "ignored destination")
	branchActionRebaseGit(t, f.root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(f.root, "ignored"), []byte("risk bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) < 2 {
		t.Fatal("ignored collision did not open approval menu")
	}
	risk := f.ui.menus[len(f.ui.menus)-1]
	if len(risk.Items) == 0 || risk.Items[0].CommandID != "branch.rebase-current-onto-selected.simple.cancel-risk" {
		t.Fatalf("first ignored-risk row isn't cancel: %+v", risk.Items)
	}
	var pathRow *types.MenuItem
	for _, item := range risk.Items {
		if item.CommandID == "branch.rebase-current-onto-selected.simple.inspect-ignored-path" {
			pathRow = item
			break
		}
	}
	if pathRow == nil {
		t.Fatal("ignored risk path row missing")
	}
	menusBefore := len(f.ui.menus)
	if err := pathRow.OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != menusBefore+1 || len(ui.blockingWorkers) != 0 || ui.subprocessCalls != 0 {
		t.Fatal("opening ignored path approved or executed the rebase")
	}
	risk = f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, risk, "branch.rebase-current-onto-selected.simple.cancel-risk").OnPress(); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("risk cancellation launched rebase")
	}
}

func TestBranchActionRebaseFlowChangedIgnoredBytesAfterApprovalRejects(t *testing.T) {
	f, ui := branchActionRebaseFixture(t)
	ignore := filepath.Join(f.root, "global-ignore")
	if err := os.WriteFile(ignore, []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "config", "core.excludesFile", ignore)
	branchActionRebaseGit(t, f.root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(f.root, "ignored"), []byte("tracked destination\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchActionRebaseGit(t, f.root, "add", "-f", "ignored")
	branchActionRebaseGit(t, f.root, "commit", "-qm", "ignored destination")
	branchActionRebaseGit(t, f.root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(f.root, "ignored"), []byte("reviewed bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	menu := branchActionRebaseStart(t, f, ui)
	branchActionRebaseChoose(t, ui, menu, "branch.rebase-current-onto-selected.simple")
	if err := ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	risk := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, risk, "branch.rebase-current-onto-selected.simple.approve-ignored-overwrite").OnPress(); err != nil {
		t.Fatal(err)
	}
	approval := menuItem(t, risk, "branch.rebase-current-onto-selected.simple.approve-ignored-overwrite")
	if err := os.WriteFile(filepath.Join(f.root, "ignored"), []byte("changed after approval\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("changed ignored bytes error=%v", err)
	}
	if err := approval.OnPress(); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("replayed approval error=%v", err)
	}
	if len(ui.blockingWorkers) != 0 {
		t.Fatal("replayed approval queued additional work")
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("stale ignored-file approval launched rebase")
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "ignored")); err != nil || string(got) != "changed after approval\n" {
		t.Fatalf("ignored bytes=%q err=%v", got, err)
	}
}
