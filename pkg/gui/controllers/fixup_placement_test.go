package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func fixupPlacementController(t *testing.T, basic *BasicCommitsController, ui *selectedRebaseFlowUI) *LocalCommitsController {
	t.Helper()
	ui.git.Version = &git_commands.GitVersion{Major: 2, Minor: 51, Patch: 1}
	ui.git.Config = git_commands.NewConfigCommands(common.NewDummyCommon(), git_config.NewFakeGitConfig(map[string]string{"rebase.updateRefs": "true"}))
	basic.c.IGuiCommon = ui
	return &LocalCommitsController{c: basic.c}
}

func fixupPlacementBranches(id string) []*models.Branch {
	return []*models.Branch{{Name: "topic", CommitHash: id}}
}

func TestFixupPlacementRejectsDirtyBeforeCreate(t *testing.T) {
	for _, tc := range []struct {
		name               string
		includeFileChanges bool
		staged             bool
	}{
		{name: "unstaged tracked changes", includeFileChanges: true},
		{name: "staged changes with message only", includeFileChanges: false, staged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			basic, ui, root, ids := selectedRebaseFlowFixture(t)
			c := fixupPlacementController(t, basic, ui)
			path := filepath.Join(root, "source")
			if err := os.WriteFile(path, []byte("edited\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if tc.staged {
				selectedRebaseFlowGit(t, root, "add", "source")
			}
			head := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
			index := selectedRebaseFlowGit(t, root, "write-tree")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			created := 0
			err = c.createCommitWithFixupPlacement(historicalTestCommit(ids[1]), 2, []*models.Commit{historicalTestCommit(ids[0]), historicalTestCommit(ids[1]), historicalTestCommit(ids[2])}, fixupPlacementBranches(ids[1]), tc.includeFileChanges, func() error { created++; return nil })
			wantErr := git_commands.ErrRebaseWouldRetainTrackedChanges
			if !tc.includeFileChanges {
				wantErr = git_commands.ErrRebaseRequiresClean
			}
			if !assert.ErrorIs(t, err, wantErr) {
				return
			}
			if created != 0 {
				t.Fatalf("create callback called %d times", created)
			}
			if got := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD"); got != head {
				t.Fatalf("HEAD changed: %s", got)
			}
			if got := selectedRebaseFlowGit(t, root, "write-tree"); got != index {
				t.Fatalf("index changed: %s", got)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != string(content) {
				t.Fatalf("content changed: %q, %v", got, err)
			}
		})
	}
}

func TestFixupPlacementCleanOrStagedAllowedBeforeRebaseCallback(t *testing.T) {
	for _, tc := range []struct {
		name               string
		staged             bool
		includeFileChanges bool
	}{
		{name: "clean", includeFileChanges: true},
		{name: "clean message only", includeFileChanges: false},
		{name: "staged index changes", staged: true, includeFileChanges: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			basic, ui, root, ids := selectedRebaseFlowFixture(t)
			c := fixupPlacementController(t, basic, ui)
			if tc.staged {
				if err := os.WriteFile(filepath.Join(root, "source"), []byte("staged edit\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				selectedRebaseFlowGit(t, root, "add", "source")
			}
			beforeHead := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
			beforeIndex := selectedRebaseFlowGit(t, root, "write-tree")
			sentinel := errors.New("stop before rebase")
			created := 0
			commits := []*models.Commit{historicalTestCommit(ids[0]), historicalTestCommit(ids[1]), historicalTestCommit(ids[2])}
			err := c.createCommitWithFixupPlacement(commits[1], 2, commits, fixupPlacementBranches(ids[1]), tc.includeFileChanges, func() error { created++; return sentinel })
			if !errors.Is(err, sentinel) {
				t.Fatalf("error = %v, want sentinel", err)
			}
			if created != 1 {
				t.Fatalf("create callback calls = %d, want 1", created)
			}
			if got := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD"); got != beforeHead {
				t.Fatalf("HEAD changed: %s", got)
			}
			if got := selectedRebaseFlowGit(t, root, "write-tree"); got != beforeIndex {
				t.Fatalf("index changed: %s", got)
			}
		})
	}
}

func TestFixupPlacementDirtyWorkAllowedWhenRebaseIsNotPlanned(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, *selectedRebaseFlowUI, string, []string, []*models.Commit) ([]*models.Branch, int)
	}{
		{name: "updateRefs disabled", setup: func(_ *testing.T, ui *selectedRebaseFlowUI, _ string, _ []string, _ []*models.Commit) ([]*models.Branch, int) {
			ui.git.Config = git_commands.NewConfigCommands(common.NewDummyCommon(), git_config.NewFakeGitConfig(map[string]string{"rebase.updateRefs": "false"}))
			return fixupPlacementBranches("owner"), 1
		}},
		{name: "no owner", setup: func(_ *testing.T, _ *selectedRebaseFlowUI, _ string, _ []string, _ []*models.Commit) ([]*models.Branch, int) {
			return nil, 1
		}},
		{name: "merged target", setup: func(_ *testing.T, _ *selectedRebaseFlowUI, _ string, _ []string, commits []*models.Commit) ([]*models.Branch, int) {
			commits[1].Status = models.StatusMerged
			return fixupPlacementBranches(commits[1].Hash()), 1
		}},
		{name: "old Git", setup: func(_ *testing.T, ui *selectedRebaseFlowUI, _ string, _ []string, _ []*models.Commit) ([]*models.Branch, int) {
			ui.git.Version = &git_commands.GitVersion{Major: 2, Minor: 37, Patch: 9}
			return fixupPlacementBranches("owner"), 1
		}},
		{name: "operation in progress", setup: func(t *testing.T, _ *selectedRebaseFlowUI, root string, _ []string, _ []*models.Commit) ([]*models.Branch, int) {
			t.Helper()
			if err := os.Mkdir(filepath.Join(root, ".git", "rebase-merge"), 0o700); err != nil {
				t.Fatal(err)
			}
			return fixupPlacementBranches("owner"), 1
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			basic, ui, root, ids := selectedRebaseFlowFixture(t)
			c := fixupPlacementController(t, basic, ui)
			path := filepath.Join(root, "source")
			if err := os.WriteFile(path, []byte("dirty tracked edit\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			commits := []*models.Commit{historicalTestCommit(ids[0]), historicalTestCommit(ids[1]), historicalTestCommit(ids[2])}
			branches, selectedIdx := tc.setup(t, ui, root, ids, commits)
			// Ensure owner cases point at the commit index considered by the helper.
			if branches != nil && branches[0].CommitHash == "owner" {
				branches[0].CommitHash = commits[1].Hash()
			}
			created := 0
			err := c.createCommitWithFixupPlacement(commits[1], selectedIdx, commits, branches, true, func() error { created++; return nil })
			if err != nil {
				t.Fatalf("dirty fixup failed without planned rebase: %v", err)
			}
			if created != 1 {
				t.Fatalf("create callback calls = %d, want 1", created)
			}
		})
	}
}
