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
	// Fallbacks are the lower-priority owners of this node's complete path,
	// highest priority first. They are never shown as choices; dispatch tries
	// them only after a leaf's handler declines with ErrKeybindingNotHandled.
	Fallbacks []*types.Binding
	Children  []CommandEdge
}

// CommandEdge connects a key to the subtree for routes beginning with it.
type CommandEdge struct {
	Key  gocui.Key
	Tree *CommandTree
}

// BuildCommandTree builds a normalized command trie for view. Explicit routes
// precede defaults; within either class, local routes precede global routes.
// Ownership applies to complete paths, never an entire shared prefix.
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

	root := &CommandTree{}
	seen := make(map[*types.Binding]map[*CommandTree]struct{})
	for _, item := range routes {
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

	pruneCommandTreeOwners(root, view)
	sortCommandTree(root)
	return root
}

func pruneCommandTreeOwners(tree *CommandTree, view string) {
	priority := func(binding *types.Binding) int {
		value := 0
		if !binding.DefaultCommand {
			value += 2
		}
		if view != "" && binding.ViewName == view {
			value++
		}
		return value
	}
	// The sort is stable, so owners of equal priority keep their input order.
	owners := append([]*types.Binding(nil), tree.Bindings...)
	sort.SliceStable(owners, func(i, j int) bool { return priority(owners[i]) > priority(owners[j]) })
	if len(owners) > 0 {
		best := priority(owners[0])
		kept := 1
		for kept < len(owners) && priority(owners[kept]) == best {
			kept++
		}
		tree.Bindings = owners[:kept:kept]
		tree.Fallbacks = owners[kept:]
	}
	for _, edge := range tree.Children {
		pruneCommandTreeOwners(edge.Tree, view)
	}
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
