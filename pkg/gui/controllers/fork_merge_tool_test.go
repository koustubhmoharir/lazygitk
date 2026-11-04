package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/stretchr/testify/assert"
)

func TestMergeToolSelectionNeedsAConflictedFile(t *testing.T) {
	controller := flatCopyFilesController(t)
	plain := &filetree.FileNode{Node: &filetree.Node[models.File]{File: &models.File{Path: "plain"}}}
	reason := controller.canOpenMergeToolForSelected([]*filetree.FileNode{plain})
	if assert.NotNil(t, reason) {
		assert.Equal(t, "No conflicted files selected", reason.Text)
		assert.True(t, reason.Inapplicable)
	}
}

func TestMergeToolSelectionAcceptsAnyConflictKind(t *testing.T) {
	controller := flatCopyFilesController(t)
	deleteModify := &filetree.FileNode{Node: &filetree.Node[models.File]{File: &models.File{Path: "dm", HasMergeConflicts: true}}}
	assert.Nil(t, controller.canOpenMergeToolForSelected([]*filetree.FileNode{deleteModify}))
}

// The directory is flattened to its files: only the conflicted ones are opened,
// and an unconflicted selection beside them is ignored.
func TestConflictedFilePathsFlattensSelectionAndKeepsConflicts(t *testing.T) {
	files := []*models.File{
		{Path: "dir/both.txt", HasMergeConflicts: true, HasInlineMergeConflicts: true},
		{Path: "dir/gone.txt", HasMergeConflicts: true},
		{Path: "dir/plain.txt"},
		{Path: "top.txt"},
	}
	tree := filetree.NewFileTree(func() []*models.File { return files }, common.NewDummyCommon(), true)
	tree.SetTree()
	root := tree.GetRoot()

	assert.Equal(t, []string{"dir/both.txt", "dir/gone.txt"}, conflictedFilePaths([]*filetree.FileNode{root}))
	assert.Empty(t, conflictedFilePaths([]*filetree.FileNode{
		{Node: &filetree.Node[models.File]{File: files[2]}},
		{Node: &filetree.Node[models.File]{File: files[3]}},
	}))
}
