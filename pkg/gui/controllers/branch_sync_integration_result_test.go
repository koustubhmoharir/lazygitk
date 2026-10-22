package controllers

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type integrationResultSnapshot struct {
	head, main, topic string
	index, unmerged   []byte
}

func snapshotIntegrationResult(t *testing.T, f *branchSyncPushFixture) integrationResultSnapshot {
	t.Helper()
	git := func(args ...string) []byte {
		cmd := exec.Command("git", append([]string{"-C", f.root}, args...)...)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return out
	}
	indexPath := string(bytes.TrimSpace(git("rev-parse", "--git-path", "index")))
	if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(f.root, indexPath)
	}
	index, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	return integrationResultSnapshot{
		head:     string(bytes.TrimSpace(git("rev-parse", "HEAD"))),
		main:     string(bytes.TrimSpace(git("rev-parse", "refs/heads/main"))),
		topic:    string(bytes.TrimSpace(git("rev-parse", "refs/heads/topic"))),
		index:    index,
		unmerged: git("ls-files", "-u", "-z"),
	}
}

func assertIntegrationResultSnapshot(t *testing.T, before, after integrationResultSnapshot) {
	t.Helper()
	if before.head != after.head || before.main != after.main || before.topic != after.topic ||
		!bytes.Equal(before.index, after.index) || !bytes.Equal(before.unmerged, after.unmerged) {
		t.Fatal("integration result handling changed HEAD, branch tips, index, or unmerged entries")
	}
}

func integrationResultFixture(t *testing.T) (*branchSyncPushFixture, *branchSyncTarget, *branchSyncCurrentPullUI) {
	t.Helper()
	f, ui := newBranchSyncCurrentPullFixture(t)
	return f, newBranchSyncTarget(f.sync.c, branchSyncSource{name: "main"}), ui
}

func execGitExpectedFailure(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	return cmd.Run()
}

func TestBranchSyncIntegrationResultConflictMergeAndRebase(t *testing.T) {
	for _, rebase := range []bool{false, true} {
		name := "merge"
		if rebase {
			name = "rebase"
		}
		t.Run(name, func(t *testing.T) {
			f, target, ui := integrationResultFixture(t)
			branchSyncPushGit(t, f.root, "checkout", "-q", "main")
			if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("main"), 0o600); err != nil {
				t.Fatal(err)
			}
			branchSyncPushGit(t, f.root, "commit", "-qam", "main edit")
			if rebase {
				branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
				if err := execGitExpectedFailure(f.root, "rebase", "main"); err == nil {
					t.Fatal("expected rebase conflict")
				}
			} else if err := execGitExpectedFailure(f.root, "merge", "topic"); err == nil {
				t.Fatal("expected merge conflict")
			}
			original := errors.New("exit status 1")
			before := snapshotIntegrationResult(t, f)
			if got := target.integrationSubprocessResult(original); got != nil {
				t.Fatalf("result = %v, want conflict menu result", got)
			}
			assertIntegrationResultSnapshot(t, before, snapshotIntegrationResult(t, f))
			if len(ui.menus) != 1 || ui.menus[0].Title != f.sync.c.Tr.FoundConflictsTitle {
				t.Fatalf("conflict menu not opened: %#v", ui.menus)
			}
			if !ui.State().GetRepoState().(*branchSyncCurrentPullRepoState).started {
				t.Fatal("paused conflict operation was not marked started")
			}
		})
	}
}

func TestBranchSyncIntegrationResultPausedWithoutConflictPreservesError(t *testing.T) {
	for _, mode := range []string{"merge-hook", "rebase-exec"} {
		t.Run(mode, func(t *testing.T) {
			f, target, ui := integrationResultFixture(t)
			branchSyncPushGit(t, f.root, "checkout", "-q", "main")
			if err := os.WriteFile(filepath.Join(f.root, "independent"), []byte("main"), 0o600); err != nil {
				t.Fatal(err)
			}
			branchSyncPushGit(t, f.root, "add", "independent")
			branchSyncPushGit(t, f.root, "commit", "-qm", "independent main change")
			if mode == "merge-hook" {
				hooks := filepath.Join(t.TempDir(), "hooks")
				if err := os.MkdirAll(hooks, 0o700); err != nil {
					t.Fatal(err)
				}
				hook := filepath.Join(hooks, "pre-merge-commit")
				if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := execGitExpectedFailure(f.root, "-c", "core.hooksPath="+hooks, "merge", "topic"); err == nil {
					t.Fatal("expected pre-merge hook failure")
				}
			} else {
				branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
				if err := execGitExpectedFailure(f.root, "rebase", "--exec", "false", "main"); err == nil {
					t.Fatal("expected stopped exec command")
				}
			}
			sentinel := errors.New("native operation stopped")
			got := target.integrationSubprocessResult(sentinel)
			if !errors.Is(got, sentinel) {
				t.Fatalf("result %v does not preserve sentinel", got)
			}
			if len(ui.menus) != 0 {
				t.Fatalf("unexpected conflict menu: %d", len(ui.menus))
			}
			if !ui.State().GetRepoState().(*branchSyncCurrentPullRepoState).started {
				t.Fatal("paused operation was not recorded")
			}
		})
	}
}

func TestBranchSyncIntegrationResultSuccessClearsOperationFlag(t *testing.T) {
	_, target, ui := integrationResultFixture(t)
	repoState := ui.State().GetRepoState().(*branchSyncCurrentPullRepoState)
	repoState.started = true
	if err := target.integrationSubprocessResult(nil); err != nil {
		t.Fatalf("result = %v", err)
	}
	if repoState.started {
		t.Fatal("operation flag was not cleared")
	}
	if len(ui.menus) != 0 {
		t.Fatalf("unexpected menu: %d", len(ui.menus))
	}
}

func TestBranchSyncIntegrationResultStaleRepositoryDoesNotTouchState(t *testing.T) {
	_, target, ui := integrationResultFixture(t)
	wrongRepoState := &branchSyncCurrentPullRepoState{started: true}
	state := &branchSyncCurrentPullState{branchSyncTestState: ui.state.(*branchSyncCurrentPullState).branchSyncTestState, repo: wrongRepoState}
	ui.state = state
	state.generation++
	sentinel := errors.New("native operation failed")
	got := target.integrationSubprocessResult(sentinel)
	if !errors.Is(got, sentinel) {
		t.Fatalf("result %v does not preserve sentinel", got)
	}
	if wrongRepoState.setCalls != 0 || !wrongRepoState.started {
		t.Fatalf("wrong repository state changed: calls=%d started=%v", wrongRepoState.setCalls, wrongRepoState.started)
	}
	if len(ui.menus) != 0 {
		t.Fatalf("stale result invoked UI: %d menus", len(ui.menus))
	}
}

func TestBranchSyncIntegrationResultUnmergedQueryFailurePreservesErrors(t *testing.T) {
	f, target, ui := integrationResultFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("main"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "commit", "-qam", "main edit")
	if err := execGitExpectedFailure(f.root, "merge", "topic"); err == nil {
		t.Fatal("expected merge conflict")
	}
	index := branchSyncPushGit(t, f.root, "rev-parse", "--git-path", "index")
	if !filepath.IsAbs(index) {
		index = filepath.Join(f.root, index)
	}
	if err := os.WriteFile(index, []byte("not a git index"), 0o600); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("native operation failed")
	got := target.integrationSubprocessResult(sentinel)
	if !errors.Is(got, sentinel) {
		t.Fatalf("result %v does not preserve original error", got)
	}
	if got == nil || !strings.Contains(strings.ToLower(got.Error()), "index") {
		t.Fatalf("result %v does not include native index query failure", got)
	}
	if len(ui.menus) != 0 {
		t.Fatalf("query failure opened conflict menu: %d", len(ui.menus))
	}
}
