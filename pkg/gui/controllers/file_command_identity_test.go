package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
)

func TestFilePrimaryCommandIdentity(t *testing.T) {
	tests := []struct {
		name        string
		nodes       []*filetree.FileNode
		needsDialog func(*testing.T, *models.File) bool
		wantCalls   int
		wantID      string
		wantVerb    string
	}{
		{
			name: "no nodes",
			needsDialog: func(t *testing.T, _ *models.File) bool {
				t.Helper()
				t.Fatal("dialog predicate called for no nodes")
				return false
			},
			wantCalls: 0,
			wantID:    "file.toggle-stage",
			wantVerb:  "toggle-stage",
		},
		{
			name: "regular file",
			nodes: []*filetree.FileNode{{Node: &filetree.Node[models.File]{File: &models.File{
				Path: "src/one.go", HasUnstagedChanges: true, ShortStatus: " M",
			}}}},
			needsDialog: func(t *testing.T, file *models.File) bool {
				t.Helper()
				if file == nil {
					t.Fatal("regular file predicate received nil")
				}
				return false
			},
			wantCalls: 1,
			wantID:    "file.toggle-stage",
			wantVerb:  "toggle-stage",
		},
		{
			name: "dialog conflict",
			nodes: []*filetree.FileNode{{Node: &filetree.Node[models.File]{File: &models.File{
				Path: "conflict.txt", HasMergeConflicts: true,
			}}}},
			needsDialog: func(t *testing.T, file *models.File) bool {
				t.Helper()
				if file == nil || !file.HasMergeConflicts {
					t.Fatal("predicate did not receive the conflict file")
				}
				return true
			},
			wantCalls: 1,
			wantID:    "conflict.options-menu",
			wantVerb:  "conflict-options",
		},
		{
			name: "multiple nodes",
			nodes: []*filetree.FileNode{
				{Node: &filetree.Node[models.File]{File: &models.File{Path: "a"}}},
				{Node: &filetree.Node[models.File]{File: &models.File{Path: "b"}}},
			},
			needsDialog: func(t *testing.T, _ *models.File) bool {
				t.Helper()
				t.Fatal("dialog predicate called for multiple nodes")
				return false
			},
			wantCalls: 0,
			wantID:    "file.toggle-stage",
			wantVerb:  "toggle-stage",
		},
		{
			name: "directory node",
			nodes: []*filetree.FileNode{{Node: &filetree.Node[models.File]{Children: []*filetree.Node[models.File]{
				{File: &models.File{Path: "dir/child.txt"}},
			}}}},
			needsDialog: func(t *testing.T, file *models.File) bool {
				t.Helper()
				if file != nil {
					t.Fatal("directory predicate received a non-nil file")
				}
				return false
			},
			wantCalls: 1,
			wantID:    "file.toggle-stage",
			wantVerb:  "toggle-stage",
		},
		{
			name: "identity ignores path and contents",
			nodes: []*filetree.FileNode{{Node: &filetree.Node[models.File]{File: &models.File{
				Path: "elsewhere/renamed.bin", PreviousPath: "old/name.bin", ShortStatus: "??",
			}}}},
			needsDialog: func(*testing.T, *models.File) bool { return false },
			wantCalls:   1,
			wantID:      "file.toggle-stage",
			wantVerb:    "toggle-stage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			predicate := func(file *models.File) bool {
				calls++
				if calls > tt.wantCalls {
					t.Fatalf("dialog predicate called more than %d times", tt.wantCalls)
				}
				return tt.needsDialog(t, file)
			}

			id, verb := filePrimaryCommandIdentity(tt.nodes, predicate)
			if calls != tt.wantCalls {
				t.Errorf("dialog predicate calls = %d, want %d", calls, tt.wantCalls)
			}
			if id != tt.wantID {
				t.Errorf("command ID = %q, want %q", id, tt.wantID)
			}
			if verb != tt.wantVerb {
				t.Errorf("command verb = %q, want %q", verb, tt.wantVerb)
			}
		})
	}
}
