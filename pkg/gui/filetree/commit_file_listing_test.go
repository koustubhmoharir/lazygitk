package filetree

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func newListingTestViewModel(files []*models.CommitFile) *CommitFileTreeViewModel {
	viewModel := NewCommitFileTreeViewModel(
		func() []*models.CommitFile { return files },
		common.NewDummyCommon(),
		true, // tree mode
	)
	viewModel.SetTree()
	return viewModel
}

func listingNodePaths(viewModel *CommitFileTreeViewModel) []string {
	paths := make([]string, 0, viewModel.Len())
	for i := 0; i < viewModel.Len(); i++ {
		paths = append(paths, viewModel.Get(i).GetInternalPath())
	}
	return paths
}

func TestCommitFileTreeCollapseDirectoriesBelowRootKeepsRootAndTopLevelEntries(t *testing.T) {
	viewModel := newListingTestViewModel([]*models.CommitFile{
		{Path: "a/one.txt"},
		{Path: "a/b/two.txt"},
		{Path: "c/three.txt"},
		{Path: "top.txt"},
	})
	viewModel.CollapseDirectoriesBelowRoot()

	assert.Equal(t, []string{".", "./a", "./c", "./top.txt"}, listingNodePaths(viewModel))
	assert.Equal(t, 4, viewModel.Len())
}

func TestCommitFileTreeSetFullListingKeepsChangedFoldersSeparate(t *testing.T) {
	viewModel := newListingTestViewModel([]*models.CommitFile{
		{Path: "a/one.txt"},
		{Path: "b/two.txt"},
	})
	assert.False(t, viewModel.IsFullListing())

	// Collapse a folder in the changed-files listing.
	viewModel.ToggleCollapsed("./a")
	changedLen := viewModel.Len()

	// Entering the full listing starts with a fresh folder state.
	viewModel.SetFullListing(true)
	assert.True(t, viewModel.IsFullListing())
	assert.False(t, viewModel.IsCollapsed("./a"))

	viewModel.CollapseDirectoriesBelowRoot()
	assert.True(t, viewModel.IsCollapsed("./a"))

	// Back to the changed-files listing restores its own folder state.
	viewModel.SetFullListing(false)
	assert.False(t, viewModel.IsFullListing())
	assert.True(t, viewModel.IsCollapsed("./a"))
	assert.Equal(t, changedLen, viewModel.Len())

	// Entering the full listing again starts it collapsed afresh, not as it was left.
	viewModel.SetFullListing(true)
	assert.False(t, viewModel.IsCollapsed("./b"))
}

func TestCommitFileTreeFullListingEntryDoesNotCollapseRefreshedTree(t *testing.T) {
	files := []*models.CommitFile{{Path: "a/one.txt"}}
	viewModel := newListingTestViewModel(files)
	viewModel.SetFullListing(true)
	viewModel.CollapseDirectoriesBelowRoot()

	// A refresh rebuilds the tree; the collapsed state of the entry stays as it was.
	files = append(files, &models.CommitFile{Path: "b/two.txt"})
	viewModel.SetTree()
	assert.True(t, viewModel.IsCollapsed("./a"))
	assert.False(t, viewModel.IsCollapsed("./b"))
}
