package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func treeNavigationBindings[T any](
	collapseDescription string,
	expandDescription string,
	tree filetree.ITree[T],
	getSelected func() *filetree.Node[T],
	setSelection func(int),
	postRefreshUpdate func(),
) []*types.Binding {
	navigate := func(expand bool) func() error {
		return func() error {
			index, changed := filetree.NavigateTreeNode(tree, getSelected(), expand)
			if changed {
				setSelection(index)
				postRefreshUpdate()
			}
			return nil
		}
	}

	return []*types.Binding{
		{CommandID: "navigation.tree.collapse", Verb: "collapse-tree-node", Handler: navigate(false), Description: collapseDescription},
		{CommandID: "navigation.tree.expand", Verb: "expand-tree-node", Handler: navigate(true), Description: expandDescription},
	}
}
