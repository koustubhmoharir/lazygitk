package keymap

import (
	"sort"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// CommandTree represents command routes as a trie. Bindings on a node are
// complete routes ending at that node; Children may still contain continuations.
type CommandTree struct {
	Bindings []*types.Binding
	Children []CommandEdge
}

// CommandEdge connects a key to the subtree for routes beginning with it.
type CommandEdge struct {
	Key  gocui.Key
	Tree *CommandTree
}

// BuildCommandTree builds a normalized command trie for view. View-specific
// routes shadow global routes only when their complete paths are identical.
func BuildCommandTree(bindings []*types.Binding, view string) *CommandTree {
	type route struct {
		binding *types.Binding
		path    []gocui.Key
	}

	var routes []route
	for _, binding := range bindings {
		if binding == nil || (binding.ViewName != "" && binding.ViewName != view) {
			continue
		}
		for _, key := range binding.Keys {
			if !gocui.IsMouseKey(key) {
				routes = append(routes, route{binding: binding, path: []gocui.Key{gocui.NormalizeCommandKey(key)}})
			}
		}
		for _, path := range binding.CommandPaths {
			if len(path) == 0 {
				continue
			}
			normalized := make([]gocui.Key, 0, len(path))
			for _, key := range path {
				if gocui.IsMouseKey(key) {
					normalized = nil
					break
				}
				normalized = append(normalized, gocui.NormalizeCommandKey(key))
			}
			if len(normalized) > 0 {
				routes = append(routes, route{binding: binding, path: normalized})
			}
		}
	}

	// Record locally-owned complete routes before insertion so a local route
	// shadows its global counterpart regardless of source order.
	var localPaths [][]gocui.Key
	for _, item := range routes {
		if item.binding.ViewName == view && view != "" {
			localPaths = append(localPaths, item.path)
		}
	}

	root := &CommandTree{}
	seen := make(map[*types.Binding]map[*CommandTree]struct{})
	for _, item := range routes {
		if item.binding.ViewName == "" && view != "" {
			shadowed := false
			for _, localPath := range localPaths {
				if equalPath(item.path, localPath) {
					shadowed = true
					break
				}
			}
			if shadowed {
				continue
			}
		}

		node := root
		for _, key := range item.path {
			node = nodeForKey(node, key)
		}
		if seen[item.binding] == nil {
			seen[item.binding] = make(map[*CommandTree]struct{})
		}
		if _, duplicate := seen[item.binding][node]; duplicate {
			continue
		}
		seen[item.binding][node] = struct{}{}
		node.Bindings = append(node.Bindings, item.binding)
	}

	sortCommandTree(root)
	return root
}

func nodeForKey(node *CommandTree, key gocui.Key) *CommandTree {
	for i := range node.Children {
		if node.Children[i].Key.Equals(key) {
			return node.Children[i].Tree
		}
	}
	child := &CommandTree{}
	node.Children = append(node.Children, CommandEdge{Key: key, Tree: child})
	return child
}

func sortCommandTree(tree *CommandTree) {
	sort.SliceStable(tree.Children, func(i, j int) bool {
		return config.LabelForKey(tree.Children[i].Key) < config.LabelForKey(tree.Children[j].Key)
	})
	for i := range tree.Children {
		sortCommandTree(tree.Children[i].Tree)
	}
}

func equalPath(a, b []gocui.Key) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}
