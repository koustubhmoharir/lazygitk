package filetree

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

type flatTreeSpy[T any] struct {
	ITree[T]
	pathLookups int
}

func (self *flatTreeSpy[T]) InTreeMode() bool {
	return false
}

func (self *flatTreeSpy[T]) GetIndexForPath(path string) (int, bool) {
	self.pathLookups++
	return self.ITree.GetIndexForPath(path)
}

func fileTreeIndex(vm *FileTreeViewModel, nodePath string) (int, bool) {
	for _, node := range vm.GetAllItems() {
		if node.GetPath() == nodePath {
			return vm.GetIndexForPath(node.GetInternalPath())
		}
	}
	return -1, false
}

func commitFileTreeIndex(vm *CommitFileTreeViewModel, nodePath string) (int, bool) {
	for _, node := range vm.GetAllItems() {
		if node.GetPath() == nodePath {
			return vm.GetIndexForPath(node.GetInternalPath())
		}
	}
	return -1, false
}

func visibleFileTreeRoot(vm *FileTreeViewModel) *FileNode {
	for _, node := range vm.GetAllItems() {
		if node.GetInternalPath() == "." || node.GetInternalPath() == "./" {
			return node
		}
	}
	return nil
}

func TestNavigateFileTree(t *testing.T) {
	files := []*models.File{{Path: "dir/first"}, {Path: "dir/sub/second"}, {Path: "other"}}
	vm := NewFileTreeViewModel(func() []*models.File { return files }, common.NewDummyCommon(), true)
	vm.SetTree()

	idx, found := fileTreeIndex(vm, "dir")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed := NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), true)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "dir/first", vm.Get(idx).GetPath())

	// Collapse an expanded directory and retain its own row.
	idx, found = fileTreeIndex(vm, "dir")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "dir", vm.Get(idx).GetPath())
	assert.True(t, vm.IsCollapsed(vm.GetSelected().GetInternalPath()))

	// Left from a collapsed nested directory selects its visible parent.
	vm.ExpandAll()
	idx, found = fileTreeIndex(vm, "dir/sub")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	assert.Equal(t, vm.GetSelectedLineIdx(), idx, "collapsing retains its selected directory")
	if !assert.True(t, changed) {
		return
	}
	assert.True(t, vm.IsCollapsed(vm.GetSelected().GetInternalPath()))
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "dir", vm.Get(idx).GetPath())

	// Left from a top-level file selects the shown root directory.
	idx, found = fileTreeIndex(vm, "other")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, ".", vm.Get(idx).GetPath())

	vm.ExpandAll()
	idx, found = fileTreeIndex(vm, "dir/first")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "dir", vm.Get(idx).GetPath())
}

func TestNavigateCompressedDirectoryAndFilteredFirstChild(t *testing.T) {
	files := []*models.File{{Path: "one/two/file"}, {Path: "one/two/other"}}
	vm := NewFileTreeViewModel(func() []*models.File { return files }, common.NewDummyCommon(), true)
	vm.SetTree()
	idx, found := fileTreeIndex(vm, "one/two")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed := NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "one/two", vm.Get(idx).GetPath())

	vm.ExpandAll()
	vm.SetTextFilter("other", false)
	idx, found = fileTreeIndex(vm, "one/two")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), true)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "one/two/other", vm.Get(idx).GetPath())
}

func TestNavigateVisibleRootAndCompressedParentWalk(t *testing.T) {
	files := []*models.File{{Path: "a/b/c/file"}, {Path: "a/b/c/other"}, {Path: "top"}}
	vm := NewFileTreeViewModel(func() []*models.File { return files }, common.NewDummyCommon(), true)
	vm.SetTree()

	root := visibleFileTreeRoot(vm)
	if !assert.NotNil(t, root) {
		return
	}
	rootPath := root.GetInternalPath()
	vm.CollapseAll()
	rootIndex, found := vm.GetIndexForPath(rootPath)
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(rootIndex)
	idx, changed := NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), true)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, rootIndex, idx)
	assert.False(t, vm.IsCollapsed(rootPath))

	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), true)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "a/b/c", vm.Get(idx).GetPath())

	// Collapse the compressed directory, then walk through its missing
	// uncompressed ancestors to the visible root.
	vm.ExpandAll()
	idx, found = fileTreeIndex(vm, "a/b/c")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	assert.Equal(t, vm.GetSelectedLineIdx(), idx, "collapsing retains its selected directory")
	if !assert.True(t, changed) {
		return
	}
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, rootPath, vm.Get(idx).GetInternalPath())

	// An expanded root collapses; an already collapsed root has no parent.
	vm.SetSelection(rootIndex)
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	assert.Equal(t, vm.GetSelectedLineIdx(), idx, "collapsing retains its selected directory")
	if !assert.True(t, changed) {
		return
	}
	assert.True(t, vm.IsCollapsed(rootPath))
	idx, changed = NavigateTreeNode[models.File](vm, vm.GetSelected().Raw(), false)
	assert.False(t, changed)
	assert.Equal(t, -1, idx)
}

func TestNavigateFileTreeNoops(t *testing.T) {
	empty := NewFileTreeViewModel(func() []*models.File { return nil }, common.NewDummyCommon(), true)
	empty.SetTree()
	idx, changed := NavigateTreeNode[models.File](empty, nil, true)
	assert.False(t, changed)
	assert.Equal(t, -1, idx)

	withRoot := NewFileTreeViewModel(func() []*models.File { return []*models.File{{Path: "dir/file"}} }, common.NewDummyCommon(), true)
	withRoot.SetTree()
	root := withRoot.GetRoot().Raw()
	idx, changed = NavigateTreeNode[models.File](withRoot, root, true)
	assert.False(t, changed)
	assert.Equal(t, -1, idx)

	flat := NewFileTreeViewModel(func() []*models.File { return []*models.File{{Path: "file"}} }, common.NewDummyCommon(), false)
	flat.SetTree()
	idx, found := fileTreeIndex(flat, "file")
	if !assert.True(t, found) {
		return
	}
	flat.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.File](flat, flat.GetSelected().Raw(), true)
	assert.False(t, changed)
	assert.Equal(t, -1, idx)
	idx, changed = NavigateTreeNode[models.File](flat, flat.GetSelected().Raw(), false)
	assert.False(t, changed)
	assert.Equal(t, -1, idx)
	spy := &flatTreeSpy[models.File]{ITree: flat}
	_, _ = NavigateTreeNode[models.File](spy, flat.GetSelected().Raw(), true)
	assert.Zero(t, spy.pathLookups)
}

func TestNavigateCommitFileTree(t *testing.T) {
	files := []*models.CommitFile{{Path: "dir/file"}, {Path: "dir/sub/deep"}, {Path: "other"}}
	vm := NewCommitFileTreeViewModel(func() []*models.CommitFile { return files }, common.NewDummyCommon(), true)
	vm.SetTree()
	idx, found := commitFileTreeIndex(vm, "dir")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed := NavigateTreeNode[models.CommitFile](vm, vm.GetSelected().Raw(), true)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "dir/file", vm.Get(idx).GetPath())

	idx, found = commitFileTreeIndex(vm, "dir/file")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.CommitFile](vm, vm.GetSelected().Raw(), false)
	if !assert.True(t, changed) {
		return
	}
	assert.Equal(t, "dir", vm.Get(idx).GetPath())

	idx, found = commitFileTreeIndex(vm, "dir/sub")
	if !assert.True(t, found) {
		return
	}
	vm.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.CommitFile](vm, vm.GetSelected().Raw(), false)
	assert.Equal(t, vm.GetSelectedLineIdx(), idx, "collapsing retains its selected directory")
	if !assert.True(t, changed) {
		return
	}
	idx, changed = NavigateTreeNode[models.CommitFile](vm, vm.GetSelected().Raw(), false)
	assert.True(t, changed)
	assert.Equal(t, "dir", vm.Get(idx).GetPath())

	flat := NewCommitFileTreeViewModel(func() []*models.CommitFile { return files }, common.NewDummyCommon(), false)
	flat.SetTree()
	idx, found = commitFileTreeIndex(flat, "other")
	if !assert.True(t, found) {
		return
	}
	flat.SetSelection(idx)
	idx, changed = NavigateTreeNode[models.CommitFile](flat, flat.GetSelected().Raw(), false)
	assert.False(t, changed)
	assert.Equal(t, -1, idx)
}
