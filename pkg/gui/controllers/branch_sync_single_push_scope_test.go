package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchSyncSinglePushScope(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   string
		selected bool
	}{
		{name: "selected local topic", source: "topic", selected: true},
		{name: "explicitly current main", source: "main"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			if tc.source == "main" {
				// Put two real local branch tips in the selected source's ancestry.
				if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("main middle"), 0o600); err != nil {
					t.Fatal(err)
				}
				branchSyncPushGit(t, f.root, "commit", "-qam", "main middle")
				middle := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
				branchSyncPushGit(t, f.root, "branch", "main-lower", middle)
				if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("main selected"), 0o600); err != nil {
					t.Fatal(err)
				}
				branchSyncPushGit(t, f.root, "commit", "-qam", "main selected")
				branchSyncPushGit(t, f.root, "branch", "main-bottom", "HEAD^")
			}

			localBefore := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
			remoteBefore := branchSyncPushGit(t, f.remote, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
			headBefore := branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD")
			indexBefore := branchSyncPushGit(t, f.root, "write-tree")
			indexBytesBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			trackedBefore, err := os.ReadFile(filepath.Join(f.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}

			if err := f.sync.pushBranchSource(branchSyncSource{name: tc.source, selected: tc.selected}); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(f.ui.menus) != 1 {
				t.Fatalf("expected one normal push menu, got %d", len(f.ui.menus))
			}
			menu := f.ui.menus[0]
			menuText := menu.Prompt
			for _, item := range menu.Items {
				menuText += "\n" + item.CommandID + " " + item.Verb + " " + item.Label
			}
			for _, id := range []string{"branch.push-selected.execute", "branch.push-selected.remote", "branch.push-selected.destination", "branch.push-selected.set-upstream", "branch.push-selected.force-with-lease"} {
				menuItem(t, menu, id)
			}
			lowerMenuText := strings.ToLower(menuText)
			if strings.Contains(menuText, "branch.push-stack") || strings.Contains(menuText, "push-stack") || strings.Contains(lowerMenuText, "push stack") || strings.Contains(lowerMenuText, "stacked below") {
				t.Fatalf("single-branch push menu exposed stack push: %q", menuText)
			}
			if len(f.ui.prompts) != 0 || len(f.ui.confirms) != 0 || len(f.ui.workers) != 0 || len(f.ui.ui) != 0 {
				t.Fatalf("opening menu continued into prompt/review/transport: prompts=%d confirms=%d workers=%d ui=%d", len(f.ui.prompts), len(f.ui.confirms), len(f.ui.workers), len(f.ui.ui))
			}
			if got := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads"); got != localBefore {
				t.Fatalf("local branch refs changed while opening menu:\nbefore %s\nafter %s", localBefore, got)
			}
			if got := branchSyncPushGit(t, f.remote, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads"); got != remoteBefore {
				t.Fatalf("remote refs changed while opening menu:\nbefore %s\nafter %s", remoteBefore, got)
			}
			if got := branchSyncPushGit(t, f.root, "symbolic-ref", "HEAD"); got != headBefore {
				t.Fatalf("HEAD changed from %s to %s", headBefore, got)
			}
			if got := branchSyncPushGit(t, f.root, "write-tree"); got != indexBefore {
				t.Fatalf("index tree changed from %s to %s", indexBefore, got)
			}
			indexBytesAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			if string(indexBytesAfter) != string(indexBytesBefore) {
				t.Fatal("raw index bytes changed while opening menu")
			}
			trackedAfter, err := os.ReadFile(filepath.Join(f.root, "tracked"))
			if err != nil {
				t.Fatal(err)
			}
			if string(trackedAfter) != string(trackedBefore) {
				t.Fatalf("worktree bytes changed: before %q, after %q", trackedBefore, trackedAfter)
			}
		})
	}
}
