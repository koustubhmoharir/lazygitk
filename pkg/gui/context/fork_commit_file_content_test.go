package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/stretchr/testify/assert"
)

// fileNode is the node of a file in the tree; a nil file gives a folder's node.
func fileNode(file *models.CommitFile) *filetree.CommitFileNode {
	return filetree.NewCommitFileNode(&filetree.Node[models.CommitFile]{File: file})
}

func TestCommitFileShowFor(t *testing.T) {
	tests := []struct {
		name        string
		fullListing bool
		node        *filetree.CommitFileNode
		hasParent   bool
		want        CommitFileShow
	}{
		{"changed-files listing shows the diff", false, fileNode(&models.CommitFile{Path: "a", ChangeStatus: "M"}), true, CommitFileShowDiff},
		{"changed-files listing of a deleted file shows the diff", false, fileNode(&models.CommitFile{Path: "a", ChangeStatus: "D"}), true, CommitFileShowDiff},
		{"changed-files listing of a folder shows the diff", false, fileNode(nil), true, CommitFileShowDiff},
		{"changed-files listing of a submodule shows the diff", false, fileNode(&models.CommitFile{Path: "sub", Submodule: true}), true, CommitFileShowDiff},
		{"no selection shows the diff", true, nil, true, CommitFileShowDiff},
		{"folder shows its listing", true, fileNode(nil), true, CommitFileShowFolder},
		{"folder without a parent shows its listing", true, fileNode(nil), false, CommitFileShowFolder},
		{"unchanged file shows its content", true, fileNode(&models.CommitFile{Path: "a"}), true, CommitFileShowContent},
		{"modified file shows its content", true, fileNode(&models.CommitFile{Path: "a", ChangeStatus: "M"}), true, CommitFileShowContent},
		{"added file shows its content", true, fileNode(&models.CommitFile{Path: "a", ChangeStatus: "A"}), true, CommitFileShowContent},
		{"renamed file shows its content", true, fileNode(&models.CommitFile{Path: "b", PreviousPath: "a", ChangeStatus: "R"}), true, CommitFileShowContent},
		{"deleted file shows its content before the commit", true, fileNode(&models.CommitFile{Path: "a", ChangeStatus: "D"}), true, CommitFileShowDeletedContent},
		{"deleted file without a parent shows the diff", true, fileNode(&models.CommitFile{Path: "a", ChangeStatus: "D"}), false, CommitFileShowDiff},
		{"submodule shows the commit it records", true, fileNode(&models.CommitFile{Path: "sub", Submodule: true}), true, CommitFileShowSubmodule},
		{"changed submodule shows the commit it records", true, fileNode(&models.CommitFile{Path: "sub", ChangeStatus: "M", Submodule: true}), true, CommitFileShowSubmodule},
		{"deleted submodule shows the commit before the commit", true, fileNode(&models.CommitFile{Path: "sub", ChangeStatus: "D", Submodule: true}), true, CommitFileShowSubmodule},
		{"deleted submodule without a parent shows the submodule", true, fileNode(&models.CommitFile{Path: "sub", ChangeStatus: "D", Submodule: true}), false, CommitFileShowSubmodule},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CommitFileShowFor(tt.fullListing, tt.node, tt.hasParent))
		})
	}
}

func TestCommitFileHasParent(t *testing.T) {
	assert.True(t, CommitFileHasParent("abc123^"))
	assert.False(t, CommitFileHasParent(""))
	assert.False(t, CommitFileHasParent(models.EmptyTreeCommitHash))
}
