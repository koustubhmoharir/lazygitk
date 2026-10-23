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
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func branchUpstreamMenuSetup(t *testing.T, local bool) (*branchSyncPushFixture, *branchSyncCurrentPullUI, *BranchesController, string) {
	t.Helper()
	f, ui, controller := branchActionResetFixture(t)
	ref, remote := "refs/company/review/base", "origin"
	if local {
		ref, remote = "refs/heads/base", "."
	} else {
		branchSyncPushGit(t, f.root, "config", "--replace-all", "remote.origin.fetch", "+refs/heads/*:refs/company/review/*")
	}
	branchSyncPushGit(t, f.root, "update-ref", ref, f.sourceOID)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", remote)
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/base")
	branchSyncPushGit(t, f.root, "tag", "base", "main")
	return f, ui, controller, ref
}

func branchUpstreamMenuOpen(t *testing.T, f *branchSyncPushFixture, ui *branchSyncCurrentPullUI, controller *BranchesController) types.CreateMenuOptions {
	t.Helper()
	if err := controller.viewUpstreamOptions(&models.Branch{Name: "topic"}); err != nil {
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

func TestBranchUpstreamMenuUsesNativeRefAndCapturedSelection(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(map[bool]string{false: "custom namespace", true: "local upstream"}[local], func(t *testing.T) {
			f, ui, controller, ref := branchUpstreamMenuSetup(t, local)
			model := &models.Branch{Name: "topic", UpstreamRemote: "wrong", UpstreamBranch: "wrong"}
			if err := controller.viewUpstreamOptions(model); err != nil {
				t.Fatal(err)
			}
			model.Name, model.UpstreamRemote, model.UpstreamBranch = "main", "other", "other"
			if err := ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			menu := f.ui.menus[0]
			if !strings.Contains(menu.Prompt, "topic") || !strings.Contains(menu.Prompt, ref) || !strings.Contains(menu.Prompt, f.sourceOID) {
				t.Fatalf("menu omitted captured native identities: %q", menu.Prompt)
			}
			if len(menu.Items) != 6 {
				t.Fatalf("upstream menu has %d rows", len(menu.Items))
			}
			reset := menuItem(t, menu, "branch.reset-selected-upstream-menu")
			if reset.DisabledReason != nil {
				t.Fatalf("native upstream reset disabled: %+v", reset.DisabledReason)
			}
			indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			workBefore, err := os.ReadFile(filepath.Join(f.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}
			if err := reset.OnPress(); err != nil {
				t.Fatal(err)
			}
			// The parent review revalidates on a worker, then the shared reset
			// flow captures its source/current HEAD on its own inspection worker.
			for range 2 {
				if err := ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				if err := ui.runUI(t); err != nil {
					t.Fatal(err)
				}
			}
			resetMenu := f.ui.menus[1]
			if !strings.Contains(resetMenu.Prompt, ref) || !strings.Contains(resetMenu.Prompt, "refs/heads/main") {
				t.Fatalf("reset omitted exact source/current destination: %q", resetMenu.Prompt)
			}
			branchActionResetChoose(t, ui, resetMenu, "branch.reset.soft")
			if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != f.sourceOID {
				t.Fatalf("reset used another source: %s", got)
			}
			if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
				t.Fatal("reset moved selected branch")
			}
			afterIndex, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			afterWork, err := os.ReadFile(filepath.Join(f.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(indexBefore, afterIndex) || !bytes.Equal(workBefore, afterWork) {
				t.Fatal("soft reset changed index/worktree bytes")
			}
		})
	}
}

func TestBranchUpstreamMenuAbsentAndGoneUpstreamsRetainMetadataChoices(t *testing.T) {
	for _, gone := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "gone"}[gone], func(t *testing.T) {
			f, ui, controller := branchActionResetFixture(t)
			if gone {
				branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
				branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/gone")
			}
			menu := branchUpstreamMenuOpen(t, f, ui, controller)
			for _, id := range []string{"branch.view-upstream-divergence", "branch.reset-selected-upstream-menu", "branch.rebase-current-onto-selected-upstream"} {
				item := menuItem(t, menu, id)
				if item.DisabledReason == nil || item.DisabledReason.Text != f.sync.c.Tr.UpstreamNotSetError {
					t.Fatalf("%s missing absent-upstream reason: %+v", id, item.DisabledReason)
				}
			}
			if menuItem(t, menu, "branch.set-upstream").DisabledReason != nil {
				t.Fatal("set upstream must remain available")
			}
			if disabled := menuItem(t, menu, "branch.unset-upstream").DisabledReason != nil; disabled == gone {
				t.Fatal("unset availability did not match saved tracking metadata")
			}
		})
	}
}

func TestBranchUpstreamMenuLabelsRemappedRemoteNamespace(t *testing.T) {
	f, ui, controller, _ := branchUpstreamMenuSetup(t, false)
	ref := "refs/remotes/review/base"
	branchSyncPushGit(t, f.root, "config", "--replace-all", "remote.origin.fetch", "+refs/heads/*:refs/remotes/review/*")
	branchSyncPushGit(t, f.root, "update-ref", ref, f.sourceOID)
	menu := branchUpstreamMenuOpen(t, f, ui, controller)
	if !strings.Contains(menu.Prompt, ref) {
		t.Fatalf("review omitted native ref: %q", menu.Prompt)
	}
	for _, id := range []string{"branch.reset-selected-upstream-menu", "branch.rebase-current-onto-selected-upstream"} {
		label := strings.Join(menuItem(t, menu, id).LabelColumns, " ")
		if !strings.Contains(label, "review/base") || strings.Contains(label, "origin/base") {
			t.Fatalf("%s mislabeled remapped upstream: %q", id, label)
		}
	}
}

func TestBranchUpstreamMenuRejectsChangedReviewedUpstream(t *testing.T) {
	f, ui, controller, ref := branchUpstreamMenuSetup(t, false)
	menu := branchUpstreamMenuOpen(t, f, ui, controller)
	initialHead := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "update-ref", ref, initialHead)
	for _, id := range []string{"branch.reset-selected-upstream-menu", "branch.rebase-current-onto-selected-upstream"} {
		if err := menuItem(t, menu, id).OnPress(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
			t.Fatalf("%s stale upstream error = %v", id, err)
		}
	}
	if len(ui.workers) != 0 || len(f.ui.menus) != 1 || branchSyncPushGit(t, f.root, "rev-parse", "HEAD") != initialHead {
		t.Fatal("stale upstream opened or executed a mutation")
	}
}

func TestBranchUpstreamMenuMutationReviewRejectsChangedBranchHeadAndMapping(t *testing.T) {
	for _, change := range []string{"selected tip", "HEAD identity", "upstream mapping", "repository during worker", "repository before submenu"} {
		t.Run(change, func(t *testing.T) {
			f, ui, controller, _ := branchUpstreamMenuSetup(t, true)
			menu := branchUpstreamMenuOpen(t, f, ui, controller)
			if err := menuItem(t, menu, "branch.reset-selected-upstream-menu").OnPress(); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "selected tip":
				branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", branchSyncPushGit(t, f.root, "rev-parse", "HEAD"))
			case "HEAD identity":
				branchSyncPushGit(t, f.root, "branch", "other-head", "HEAD")
				branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD", "refs/heads/other-head")
			case "upstream mapping":
				branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/main")
			case "repository during worker":
				f.ui.state.generation++
			}
			err := ui.runWorker(t)
			if change == "repository before submenu" {
				if err != nil {
					t.Fatal(err)
				}
				f.ui.state.generation++
				err = ui.runUI(t)
			}
			if err == nil {
				t.Fatal("changed review opened a reset submenu")
			}
			if len(ui.workers) != 0 || len(f.ui.menus) != 1 || len(ui.blockingWorkers) != 0 {
				t.Fatal("changed review queued a reset or opened another menu")
			}
		})
	}
}

func TestBranchUpstreamMenuPreservesNativeInspectionErrors(t *testing.T) {
	t.Run("non-commit upstream is not missing", func(t *testing.T) {
		f, ui, controller, ref := branchUpstreamMenuSetup(t, false)
		blob := branchSyncPushGit(t, f.root, "hash-object", "-w", "tracked")
		branchSyncPushGit(t, f.root, "update-ref", ref, blob)
		menu := branchUpstreamMenuOpen(t, f, ui, controller)
		for _, id := range []string{"branch.view-upstream-divergence", "branch.reset-selected-upstream-menu", "branch.rebase-current-onto-selected-upstream"} {
			reason := menuItem(t, menu, id).DisabledReason
			if reason == nil || reason.Text == f.sync.c.Tr.UpstreamNotSetError {
				t.Fatalf("%s swallowed a native inspection error: %+v", id, reason)
			}
		}
		if menuItem(t, menu, "branch.set-upstream").DisabledReason != nil || menuItem(t, menu, "branch.unset-upstream").DisabledReason != nil {
			t.Fatal("invalid upstream must not disable metadata repair choices")
		}
	})
	t.Run("malformed native config", func(t *testing.T) {
		f, ui, controller, _ := branchUpstreamMenuSetup(t, false)
		if err := controller.viewUpstreamOptions(&models.Branch{Name: "topic"}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, ".git", "config"), []byte("[unterminated\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err == nil || !strings.Contains(err.Error(), "config") {
			t.Fatalf("native config error = %v", err)
		}
		if len(f.ui.menus) != 0 || len(ui.ui) != 0 {
			t.Fatal("failed native inspection opened an upstream menu")
		}
	})
}

func TestBranchUpstreamMenuRepositoryBoundaries(t *testing.T) {
	for _, boundary := range []string{"inspection", "queued UI", "callbacks"} {
		for _, change := range []string{"pointer", "generation"} {
			t.Run(boundary+" "+change, func(t *testing.T) {
				f, ui, controller, _ := branchUpstreamMenuSetup(t, true)
				switchRepository := func() {
					if change == "pointer" {
						f.ui.git = newBranchSyncPushFixture(t).git
					} else {
						f.ui.state.generation++
					}
				}
				if err := controller.viewUpstreamOptions(&models.Branch{Name: "topic"}); err != nil {
					t.Fatal(err)
				}
				if boundary == "inspection" {
					switchRepository()
					if err := ui.runWorker(t); err == nil {
						t.Fatal("inspection accepted repository generation change")
					}
					return
				}
				if err := ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				if boundary == "queued UI" {
					switchRepository()
					if err := ui.runUI(t); err == nil {
						t.Fatal("queued menu accepted repository generation change")
					}
					return
				}
				if err := ui.runUI(t); err != nil {
					t.Fatal(err)
				}
				menu := f.ui.menus[0]
				switchRepository()
				for _, item := range menu.Items {
					if item.DisabledReason == nil && item.OnPress != nil {
						if err := item.OnPress(); err == nil {
							t.Fatalf("%s accepted repository change", item.CommandID)
						}
					}
				}
				if menu.OnCancel == nil || menu.OnCancel() == nil {
					t.Fatal("cancel did not retain original repository")
				}
				if len(ui.workers) != 0 || len(f.ui.menus) != 1 {
					t.Fatal("stale callback queued work or another menu")
				}
			})
		}
	}
}
