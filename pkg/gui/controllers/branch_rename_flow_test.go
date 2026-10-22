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
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchRenameRecordingUI struct {
	*branchSyncPushTestUI
	refreshOptions []types.RefreshOptions
	actions        []string
}

func (ui *branchRenameRecordingUI) RefreshFromWorker(opts types.RefreshOptions) {
	ui.refreshOptions = append(ui.refreshOptions, opts)
}

func (ui *branchRenameRecordingUI) LogAction(action string) { ui.actions = append(ui.actions, action) }

func branchRenameController(f *branchSyncPushFixture) (*BranchesController, *branchRenameRecordingUI) {
	ui := &branchRenameRecordingUI{branchSyncPushTestUI: f.ui}
	c := NewControllerCommon(&helpers.HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}, nil)
	return &BranchesController{c: c}, ui
}

func TestBranchRenameFlowCapturesLiteralTargetAndAppliesOnce(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	controller, recordingUI := branchRenameController(f)
	if err := controller.renameCapturedBranch("topic"); err != nil {
		t.Fatal(err)
	}
	f.ui.model.Branches = []*models.Branch{{Name: "main"}}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 || f.ui.prompts[0].InitialContent != "topic" {
		t.Fatalf("unexpected rename prompt: %+v", f.ui.prompts)
	}
	// An invalid Git name is passed through verbatim and rejected, never cleaned.
	invalid := " topic "
	if err := f.ui.prompts[0].HandleConfirm(invalid); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("invalid literal branch name was accepted")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatalf("invalid prompt changed source branch: %s", got)
	}

	if err := f.ui.prompts[0].HandleConfirm("renamed"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, "topic") || !strings.Contains(f.ui.confirms[0].Prompt, "renamed") || !strings.Contains(f.ui.confirms[0].Prompt, f.sourceOID) {
		t.Fatalf("confirmation omitted captured rename details: %+v", f.ui.confirms)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/renamed"); got != f.sourceOID {
		t.Fatalf("renamed tip %s, want %s", got, f.sourceOID)
	}
	if len(recordingUI.refreshOptions) != 1 || recordingUI.refreshOptions[0].Then == nil {
		t.Fatal("successful rename did not queue a guarded refresh with reselection")
	}
	if len(recordingUI.actions) != 1 {
		t.Fatalf("rename action log count = %d, want 1", len(recordingUI.actions))
	}
	if got := branchSyncPushGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Fatalf("rename changed current branch to %s", got)
	}
}

func TestBranchRenameFlowRepositoryChangeBlocksPromptCallback(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	controller, _ := branchRenameController(f)
	if err := controller.renameCapturedBranch("topic"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(t.TempDir())}
	if err := f.ui.runUI(t); err == nil {
		t.Fatal("repository replacement was accepted")
	}
	if len(f.ui.prompts) != 0 {
		t.Fatal("stale repository opened a rename prompt")
	}
}

func TestBranchRenameFlowPreservesDirtyBytesAndRejectsStalePlan(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged base update"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged final update"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "binary"), []byte{0, 1, 2, 0xff, 0}, 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "-f", "binary")
	beforeIndex := branchSyncPushGit(t, f.root, "diff", "--cached", "--binary")
	beforeTracked, _ := os.ReadFile(filepath.Join(f.root, "tracked"))
	beforeBinary, _ := os.ReadFile(filepath.Join(f.root, "binary"))
	controller, _ := branchRenameController(f)
	if err := controller.renameCapturedBranch("topic"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[0].HandleConfirm("topic-renamed"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	// Change the source tip after review; confirmation must not apply the old plan.
	changedTip := branchSyncPushGit(t, f.root, "commit-tree", f.sourceOID+"^{tree}", "-p", f.sourceOID, "-m", "concurrent tip")
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", changedTip, f.sourceOID)
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("stale source tip was accepted")
	}
	if got := branchSyncPushGit(t, f.root, "diff", "--cached", "--binary"); got != beforeIndex {
		t.Fatal("rename changed staged bytes")
	}
	afterTracked, _ := os.ReadFile(filepath.Join(f.root, "tracked"))
	afterBinary, _ := os.ReadFile(filepath.Join(f.root, "binary"))
	if !bytes.Equal(afterTracked, beforeTracked) || !bytes.Equal(afterBinary, beforeBinary) {
		t.Fatal("rename changed worktree bytes")
	}
}

func TestBranchRenameFlowWarnsForGoneConfiguredUpstream(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	controller, _ := branchRenameController(f)
	if err := controller.renameCapturedBranch("topic"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 || f.ui.confirms[0].Prompt != controller.c.Tr.RenameBranchWarning {
		t.Fatal("gone configured upstream did not retain tracking warning")
	}
}

func TestBranchRenameFlowCancelDoesNotWrite(t *testing.T) {
	for _, cancelAtWarning := range []bool{true, false} {
		t.Run(map[bool]string{true: "warning", false: "name prompt"}[cancelAtWarning], func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			if cancelAtWarning {
				branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
				branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
			}
			controller, _ := branchRenameController(f)
			if err := controller.renameCapturedBranch("topic"); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if cancelAtWarning {
				if err := f.ui.confirms[0].HandleClose(); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := f.ui.prompts[0].HandleClose(); err != nil {
					t.Fatal(err)
				}
			}
			if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
				t.Fatalf("cancel changed source tip: %s", got)
			}
			if f.ui.refreshes != 0 {
				t.Fatal("cancel queued a refresh")
			}
		})
	}
}

func TestBranchRenameFlowRefreshesAfterNativePartialFailure(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	controller, recorder := branchRenameController(f)
	if err := controller.renameCapturedBranch("topic"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[0].HandleConfirm("renamed"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(f.root, ".git", "config.lock")
	if err := os.WriteFile(lockPath, []byte("held"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("native partial failure was swallowed")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/renamed"); got != f.sourceOID {
		t.Fatalf("native ref rename was rolled back or lost: %s", got)
	}
	if len(recorder.refreshOptions) != 1 || recorder.refreshOptions[0].Then != nil {
		t.Fatal("partial failure did not queue refresh without reselection")
	}
}

func TestBranchRenameFlowDestinationAppearingAfterReviewIsRefused(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	controller, _ := branchRenameController(f)
	if err := controller.renameCapturedBranch("topic"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[0].HandleConfirm("new-destination"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "branch", "new-destination", f.sourceOID)
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("appeared destination was replaced")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatalf("source changed after refusal: %s", got)
	}
}

func TestBranchRenameFlowFinalWorkerAndThenRejectChangedRepository(t *testing.T) {
	for _, changeAt := range []string{"final", "worker", "then"} {
		t.Run(changeAt, func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			controller, recorder := branchRenameController(f)
			if err := controller.renameCapturedBranch("topic"); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.prompts[0].HandleConfirm("renamed"); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if changeAt == "final" {
				f.ui.state.generation++
				if err := f.ui.confirms[0].HandleConfirm(); err == nil {
					t.Fatal("stale confirmation accepted")
				}
				if len(f.ui.workers) != 0 {
					t.Fatal("stale confirmation queued a worker")
				}
				return
			}
			if err := f.ui.confirms[0].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if changeAt == "worker" {
				f.ui.state.generation++
			}
			workerErr := f.ui.runWorker(t)
			if changeAt == "worker" {
				if workerErr == nil {
					t.Fatal("worker accepted changed repository generation")
				}
				if len(recorder.refreshOptions) != 0 {
					t.Fatal("stale worker refreshed another repository")
				}
				return
			}
			if workerErr != nil {
				t.Fatal(workerErr)
			}
			f.ui.state.generation++
			if err := recorder.refreshOptions[0].Then(); err == nil {
				t.Fatal("stale Then callback accepted changed generation")
			}
		})
	}
}

func TestBranchRenameFlowPointerChangesBlockEveryQueuedStage(t *testing.T) {
	for _, changeAt := range []string{"prompt", "final", "worker", "then"} {
		t.Run(changeAt, func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			controller, recorder := branchRenameController(f)
			if err := controller.renameCapturedBranch("topic"); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if changeAt == "prompt" {
				f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(t.TempDir())}
				if err := f.ui.prompts[0].HandleConfirm("renamed"); err == nil {
					t.Fatal("stale prompt accepted changed Git pointer")
				}
				return
			}
			if err := f.ui.prompts[0].HandleConfirm("renamed"); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if changeAt == "final" {
				f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(t.TempDir())}
				if err := f.ui.confirms[0].HandleConfirm(); err == nil {
					t.Fatal("stale final confirmation accepted changed Git pointer")
				}
				return
			}
			if err := f.ui.confirms[0].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if changeAt == "worker" {
				f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(t.TempDir())}
			}
			workerErr := f.ui.runWorker(t)
			if changeAt == "worker" {
				if workerErr == nil {
					t.Fatal("worker accepted changed Git pointer")
				}
				if len(recorder.refreshOptions) != 0 {
					t.Fatal("stale worker refreshed a replacement repository")
				}
				return
			}
			if workerErr != nil {
				t.Fatal(workerErr)
			}
			f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(t.TempDir())}
			if err := recorder.refreshOptions[0].Then(); err == nil {
				t.Fatal("Then accepted changed Git pointer")
			}
		})
	}
}
