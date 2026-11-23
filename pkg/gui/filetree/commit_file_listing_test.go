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

// changedPathsUnder lists the changed files under the given internal paths of the view
// model's tree, which must exist.
func changedPathsUnder(t *testing.T, viewModel *CommitFileTreeViewModel, internalPaths ...string) []string {
	t.Helper()
	nodes := make([]*Node[models.CommitFile], 0, len(internalPaths))
	for _, internalPath := range internalPaths {
		var found *Node[models.CommitFile]
		for _, node := range viewModel.GetAllItems() {
			if node.GetInternalPath() == internalPath {
				found = node.Raw()
			}
		}
		if found == nil {
			t.Fatalf("no node at %q", internalPath)
		}
		nodes = append(nodes, found)
	}
	files := viewModel.ChangedFilesUnder(nodes...)
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func TestCommitFileTreeChangedFilesUnderFullListing(t *testing.T) {
	files := []*models.CommitFile{
		{Path: "a/x.txt", ChangeStatus: "M"},
		{Path: "a/b/y.txt"},
		{Path: "c/z.txt"},
		{Path: "c/w.txt", ChangeStatus: "D"},
		{Path: "top.txt"},
	}
	viewModel := newListingTestViewModel(files)
	viewModel.SetFullListing(true)

	for _, tc := range []struct {
		name          string
		internalPaths []string
		want          []string
	}{
		{name: "changed file", internalPaths: []string{"./a/x.txt"}, want: []string{"a/x.txt"}},
		{name: "unchanged file", internalPaths: []string{"./top.txt"}, want: []string{}},
		{name: "folder with changed and unchanged files", internalPaths: []string{"./c"}, want: []string{"c/w.txt"}},
		{name: "folder with only unchanged files", internalPaths: []string{"./a/b"}, want: []string{}},
		{name: "root", internalPaths: []string{"."}, want: []string{"a/x.txt", "c/w.txt"}},
		{name: "overlapping nodes count each file once", internalPaths: []string{"./a", "./a/x.txt"}, want: []string{"a/x.txt"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, changedPathsUnder(t, viewModel, tc.internalPaths...))
		})
	}
}

func TestCommitFileTreeChangedFilesUnderChangedFilesListingUnaffected(t *testing.T) {
	// Every file of the changed-files listing is changed, so the helper returns every
	// file under a node, as walking the node does.
	viewModel := newListingTestViewModel([]*models.CommitFile{
		{Path: "a/x.txt", ChangeStatus: "M"},
		{Path: "a/b/y.txt", ChangeStatus: "A"},
		{Path: "top.txt", ChangeStatus: "D"},
	})

	assert.True(t, viewModel.IsChangedFile(&models.CommitFile{Path: "a/x.txt", ChangeStatus: "M"}))
	assert.True(t, viewModel.IsChangedFile(&models.CommitFile{Path: "unlisted.txt"}))
	assert.Equal(t, []string{"a/b/y.txt", "a/x.txt"}, changedPathsUnder(t, viewModel, "./a"))
	assert.Equal(t, []string{"a/b/y.txt", "a/x.txt", "top.txt"}, changedPathsUnder(t, viewModel, "."))
}

func TestCommitFileTreeChangedFilesUnderFollowsTheTextFilter(t *testing.T) {
	viewModel := newListingTestViewModel([]*models.CommitFile{
		{Path: "a/x.txt", ChangeStatus: "M"},
		{Path: "a/other.txt"},
	})
	viewModel.SetFullListing(true)
	viewModel.SetFilter("x.txt", false)

	assert.Equal(t, []string{"a/x.txt"}, changedPathsUnder(t, viewModel, "./a"))
}
