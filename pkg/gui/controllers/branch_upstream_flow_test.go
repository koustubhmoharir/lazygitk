package controllers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchUpstreamSwitchUI struct {
	*branchSyncPushTestUI
	activeGit                  *commands.GitCommand
	oldGit, newGit             *commands.GitCommand
	oldRefreshes, newRefreshes int
}

func (ui *branchUpstreamSwitchUI) Git() *commands.GitCommand { return ui.activeGit }
func (ui *branchUpstreamSwitchUI) RefreshFromWorker(types.RefreshOptions) {
	if ui.activeGit == ui.oldGit {
		ui.oldRefreshes++
	}
	if ui.activeGit == ui.newGit {
		ui.newRefreshes++
	}
}

func TestBranchUpstreamFlowSetUsesExactNativeRefAndPreservesCheckout(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "update-ref", "refs/cache/review/topic", f.sourceOID)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("staged then dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "tracked")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("unstaged"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/old")
	branchSyncPushGit(t, f.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/cache/review/*")
	indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	headBefore := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	refBefore := branchSyncPushGit(t, f.root, "show-ref", "--hash", "refs/cache/review/topic")
	flow := newBranchUpstreamFlow(f.sync.c, "topic")
	if err := flow.set(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 || f.ui.prompts[0].InitialContent != "refs/cache/review/old" {
		t.Fatalf("prompt did not use fresh native upstream info: %+v", f.ui.prompts)
	}
	if !f.ui.prompts[0].PreserveWhitespace {
		t.Fatal("full ref prompt must preserve whitespace")
	}
	if err := f.ui.prompts[0].HandleConfirm("refs/cache/review/topic"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatalf("reviews=%d", len(f.ui.confirms))
	}
	for _, expected := range []string{"refs/cache/review/topic", refBefore, "origin", "refs/heads/old"} {
		if !strings.Contains(f.ui.confirms[0].Prompt, expected) {
			t.Errorf("review missing %q: %s", expected, f.ui.confirms[0].Prompt)
		}
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "config", "--get", "branch.topic.merge"); got != "refs/heads/topic" {
		t.Fatalf("merge=%q", got)
	}
	if got := branchSyncPushGit(t, f.root, "config", "--get", "branch.topic.remote"); got != "origin" {
		t.Fatalf("remote=%q", got)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != headBefore {
		t.Fatal("HEAD changed")
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Fatal("index changed")
	}
	contents, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "unstaged" {
		t.Fatalf("worktree=%q", contents)
	}
}

func TestBranchUpstreamFlowUnsetReviewsThenUnsets(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/topic")
	flow := newBranchUpstreamFlow(f.sync.c, "topic")
	if err := flow.unset(); err != nil {
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
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, "refs/heads/topic") {
		t.Fatalf("missing unset review: %+v", f.ui.confirms)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "config")); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "config", "--local", "--list"); strings.Contains(got, "branch.topic.remote") || strings.Contains(got, "branch.topic.merge") {
		t.Fatalf("upstream config remains: %s", got)
	}
}

func TestBranchUpstreamFlowCancelAndRepositoryGuards(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	flow := newBranchUpstreamFlow(f.sync.c, "topic")
	if err := flow.set(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.workers) != 0 {
		t.Fatal("prompt cancel queued a write")
	}
	if err := f.ui.prompts[0].HandleConfirm("refs/heads/main"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation++
	if err := f.ui.confirms[0].HandleClose(); err == nil {
		t.Fatal("stale confirm close was accepted")
	}
	if err := f.ui.confirms[0].HandleConfirm(); err == nil {
		t.Fatal("stale confirmation was accepted")
	}
	if len(f.ui.workers) != 0 {
		t.Fatal("stale confirmation queued a worker")
	}
}

func TestBranchUpstreamFlowRejectsStaleCallbacksAtEveryBoundary(t *testing.T) {
	stages := []string{"constructor-action", "inspect-worker-ui", "prompt-prepare-worker", "review-final", "queued-execute-worker"}
	for _, stage := range stages {
		for _, change := range []string{"pointer", "generation"} {
			t.Run(stage+"/"+change, func(t *testing.T) {
				f := newBranchSyncPushFixture(t)
				replacement := newBranchSyncPushFixture(t)
				switchUI := &branchUpstreamSwitchUI{branchSyncPushTestUI: f.ui, activeGit: f.git, oldGit: f.git, newGit: replacement.git}
				f.sync.c.HelperCommon.IGuiCommon = switchUI
				flow := newBranchUpstreamFlow(f.sync.c, "topic")
				makeStale := func() {
					if change == "pointer" {
						switchUI.activeGit = replacement.git
					} else {
						f.ui.state.generation++
					}
				}
				advanceToReview := func() {
					if err := flow.set(); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.runWorker(t); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.runUI(t); err != nil {
						t.Fatal(err)
					}
					if len(f.ui.prompts) != 1 {
						t.Fatalf("prompts=%d", len(f.ui.prompts))
					}
					if err := f.ui.prompts[0].HandleConfirm("refs/heads/main"); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.runWorker(t); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.runUI(t); err != nil {
						t.Fatal(err)
					}
					if len(f.ui.confirms) != 1 {
						t.Fatalf("reviews=%d", len(f.ui.confirms))
					}
				}
				switch stage {
				case "constructor-action":
					makeStale()
					if err := flow.set(); err == nil {
						t.Fatal("stale action accepted")
					}
				case "inspect-worker-ui":
					if err := flow.set(); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.runWorker(t); err != nil {
						t.Fatal(err)
					}
					makeStale()
					if err := f.ui.runUI(t); err == nil {
						t.Fatal("stale inspection callback accepted")
					}
				case "prompt-prepare-worker":
					if err := flow.set(); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.runWorker(t); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.runUI(t); err != nil {
						t.Fatal(err)
					}
					if err := f.ui.prompts[0].HandleConfirm("refs/heads/main"); err != nil {
						t.Fatal(err)
					}
					makeStale()
					if err := f.ui.runWorker(t); err == nil {
						t.Fatal("stale preparation worker accepted")
					}
				case "review-final":
					advanceToReview()
					makeStale()
					if err := f.ui.confirms[0].HandleConfirm(); err == nil {
						t.Fatal("stale final callback accepted")
					}
				case "queued-execute-worker":
					advanceToReview()
					if err := f.ui.confirms[0].HandleConfirm(); err != nil {
						t.Fatal(err)
					}
					makeStale()
					if err := f.ui.runWorker(t); err == nil {
						t.Fatal("stale execute worker accepted")
					}
				}
				if switchUI.oldRefreshes != 0 || switchUI.newRefreshes != 0 {
					t.Fatalf("stale flow refreshed old=%d replacement=%d", switchUI.oldRefreshes, switchUI.newRefreshes)
				}
				if len(f.ui.confirms) > 0 && change == "generation" && stage != "constructor-action" {
					config := branchSyncPushGit(t, f.root, "config", "--local", "--list")
					if strings.Contains(config, "branch.topic.merge") || strings.Contains(config, "branch.topic.remote") {
						t.Fatalf("stale flow wrote tracking config: %s", config)
					}
				}
			})
		}
	}
}

func TestBranchUpstreamFlowCapturedBranchAndMutableModelDoNotRetarget(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/other", f.sourceOID)
	flow := newBranchUpstreamFlow(f.sync.c, "topic")
	if err := flow.set(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	f.ui.model.Branches = nil
	if err := f.ui.prompts[0].HandleConfirm("refs/heads/other"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, "topic") || !strings.Contains(f.ui.confirms[0].Prompt, "refs/heads/other") {
		t.Fatalf("review retargeted: %s", f.ui.confirms[0].Prompt)
	}
}

func TestBranchUpstreamFlowRefusesStaleReviewedInputs(t *testing.T) {
	for _, mutate := range []struct {
		name string
		fn   func(*testing.T, *branchSyncPushFixture)
	}{
		{"source tip", func(t *testing.T, f *branchSyncPushFixture) {
			t.Helper()
			branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "move topic")
			branchSyncPushGit(t, f.root, "branch", "-f", "topic", "HEAD")
		}},
		{"tracking tip", func(t *testing.T, f *branchSyncPushFixture) {
			t.Helper()
			mainOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
			branchSyncPushGit(t, f.root, "update-ref", "refs/heads/other", mainOID)
		}},
		{"remote profile", func(t *testing.T, f *branchSyncPushFixture) {
			t.Helper()
			branchSyncPushGit(t, f.root, "remote", "add", "other", filepath.Join(t.TempDir(), "other.git"))
		}},
		{"configuration", func(t *testing.T, f *branchSyncPushFixture) {
			t.Helper()
			branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
		}},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			branchSyncPushGit(t, f.root, "update-ref", "refs/heads/other", f.sourceOID)
			flow := newBranchUpstreamFlow(f.sync.c, "topic")
			if err := flow.set(); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.prompts[0].HandleConfirm("refs/heads/other"); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			mutate.fn(t, f)
			if err := f.ui.confirms[0].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err == nil {
				t.Fatal("stale reviewed plan executed")
			}
			config := branchSyncPushGit(t, f.root, "config", "--local", "--list")
			if strings.Contains(config, "branch.topic.merge") {
				t.Fatalf("tracking config changed: %s", config)
			}
		})
	}
}

func TestBranchUpstreamFlowAmbiguousConfigurationIsNotRepaired(t *testing.T) {
	for _, kind := range []string{"global", "include", "multiple-merge"} {
		for _, operation := range []string{"set", "unset"} {
			t.Run(kind+"/"+operation, func(t *testing.T) {
				globalConfig := filepath.Join(t.TempDir(), "global-config")
				if err := os.WriteFile(globalConfig, nil, 0o600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
				f := newBranchSyncPushFixture(t)
				switch kind {
				case "global":
					branchSyncPushGit(t, f.root, "config", "--global", "branch.topic.remote", "origin")
					branchSyncPushGit(t, f.root, "config", "--global", "branch.topic.merge", "refs/heads/topic")
				case "include":
					included := filepath.Join(t.TempDir(), "included-config")
					if err := os.WriteFile(included, []byte("[branch \"topic\"]\n remote = origin\n merge = refs/heads/topic\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					branchSyncPushGit(t, f.root, "config", "--local", "include.path", included)
				case "multiple-merge":
					branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "origin")
					branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/topic")
					branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/other")
				}
				if kind != "multiple-merge" {
					branchSyncPushGit(t, f.root, "config", "--local", "branch.topic.remote", "origin")
					branchSyncPushGit(t, f.root, "config", "--local", "branch.topic.merge", "refs/heads/topic")
				}
				before, err := os.ReadFile(filepath.Join(f.root, ".git", "config"))
				if err != nil {
					t.Fatal(err)
				}
				flow := newBranchUpstreamFlow(f.sync.c, "topic")
				var callErr error
				if operation == "unset" {
					callErr = flow.unset()
				} else {
					callErr = flow.set()
				}
				if callErr != nil {
					t.Fatal(callErr)
				}
				if err := f.ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				if err := f.ui.runUI(t); err != nil {
					t.Fatal(err)
				}
				if operation == "set" {
					if err := f.ui.prompts[0].HandleConfirm("refs/heads/main"); err != nil {
						t.Fatal(err)
					}
				}
				if err := f.ui.runWorker(t); err == nil {
					t.Fatal("ambiguous/inherited upstream was accepted")
				}
				if len(f.ui.confirms) != 0 {
					t.Fatal("ambiguous upstream reached review")
				}
				after, err := os.ReadFile(filepath.Join(f.root, ".git", "config"))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, after) {
					t.Fatal("ambiguous operation changed config bytes")
				}
			})
		}
	}
}

func TestBranchUpstreamFlowSetAcceptsDirectLocalTarget(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	flow := newBranchUpstreamFlow(f.sync.c, "topic")
	if err := flow.set(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[0].HandleConfirm("refs/heads/main"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, "refs/heads/main") {
		t.Fatalf("local full ref absent: %s", f.ui.confirms[0].Prompt)
	}
}

func TestBranchUpstreamFlowReviewRedactsPriorRemoteWithoutChangingConfig(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	raw := "https://user:secret@example.com/r?token=secret#private"
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", raw)
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/main")
	before, err := os.ReadFile(filepath.Join(f.root, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	flow := newBranchUpstreamFlow(f.sync.c, "topic")
	if err := flow.set(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[0].HandleConfirm("refs/heads/main"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	prompt := f.ui.confirms[0].Prompt
	for _, secret := range []string{"secret", "token=", "private", "user:"} {
		if strings.Contains(prompt, secret) {
			t.Fatalf("review exposed %q: %s", secret, prompt)
		}
	}
	after, err := os.ReadFile(filepath.Join(f.root, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("review modified native config")
	}
}
