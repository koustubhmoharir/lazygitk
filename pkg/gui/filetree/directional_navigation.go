package filetree

import (
	"path"
	"strings"
)

// NavigateTreeNode expands or collapses a directory, or moves to the first
// visible child/parent. The returned index is -1 when no navigation occurred.
func NavigateTreeNode[T any](tree ITree[T], selected *Node[T], expand bool) (newIndex int, changed bool) {
	if tree == nil || selected == nil || !tree.InTreeMode() || tree.Len() == 0 {
		return -1, false
	}

	selectedIndex, found := tree.GetIndexForPath(selected.GetInternalPath())
	if !found || selectedIndex < 0 || selectedIndex >= tree.Len() {
		return -1, false
	}

	if expand {
		if len(selected.Children) > 0 && tree.IsCollapsed(selected.GetInternalPath()) {
			tree.ToggleCollapsed(selected.GetInternalPath())
			return selectedIndex, true
		}
		if len(selected.Children) == 0 {
			return -1, false
		}
		childIndex, found := tree.GetIndexForPath(selected.Children[0].GetInternalPath())
		if !found || childIndex < 0 || childIndex >= tree.Len() {
			return -1, false
		}
		return childIndex, true
	}

	if len(selected.Children) > 0 && !tree.IsCollapsed(selected.GetInternalPath()) {
		tree.ToggleCollapsed(selected.GetInternalPath())
		return selectedIndex, true
	}

	selectedPath := selected.GetInternalPath()
	parentPath := path.Dir(selectedPath)
	for parentPath != "" && parentPath != selectedPath {
		lookupPath := parentPath
		if strings.HasPrefix(selectedPath, "./") && lookupPath != "." {
			lookupPath = "./" + strings.TrimPrefix(lookupPath, "./")
		}
		parentIndex, found := tree.GetIndexForPath(lookupPath)
		if found {
			if parentIndex >= 0 && parentIndex < tree.Len() {
				return parentIndex, true
			}
			return -1, false
		}
		next := path.Dir(parentPath)
		if next == parentPath {
			break
		}
		parentPath = next
	}

	return -1, false
}
