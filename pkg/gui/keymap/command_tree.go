package keymap

import (
	"fmt"
	"slices"
	"sort"
	"strings"

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
	// Shadowed, set on the root only, lists the default global bindings with a route
	// that a view-local route hides (see shadowGlobalRoutes). They have no route in
	// the view, so labels must not fall back to their global route.
	Shadowed []*types.Binding
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
	var routes []commandRoute
	for _, binding := range bindings {
		if binding == nil || (binding.ViewName != "" && binding.ViewName != view) {
			continue
		}
		for _, key := range binding.Keys {
			if !gocui.IsMouseKey(key) {
				routes = append(routes, commandRoute{binding: binding, path: []gocui.Key{gocui.NormalizeCommandKey(key)}})
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
				routes = append(routes, commandRoute{binding: binding, path: normalized})
			}
		}
	}

	routes, shadowed := shadowGlobalRoutes(routes, view)

	root := &CommandTree{Shadowed: shadowed}
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
	sortCommandTree(root, nil)
	return root
}

// commandRoute is one normalized path by which a binding is reached.
type commandRoute struct {
	binding *types.Binding
	path    []gocui.Key
}

// shadowGlobalRoutes implements the local-shadows-global rule of BuildCommandTree.
// In view, a complete local path P hides every default global route that strictly
// extends P: a local single key such as Files' c hides the global c c and c p, so
// those keys mean nothing in that pane. Global routes that do not extend a local
// path are kept, and so are explicit global routes, which keep the precedence of
// pruneCommandTreeOwners. The rule works on the tree, so labels and prefix menus
// that read the tree agree with it.
func shadowGlobalRoutes(routes []commandRoute, view string) ([]commandRoute, []*types.Binding) {
	if view == "" {
		return routes, nil
	}
	local := make(map[string]struct{})
	for _, item := range routes {
		if item.binding.ViewName == view {
			local[commandPathID(item.path)] = struct{}{}
		}
	}
	var shadowed []*types.Binding
	kept := routes[:0]
	for _, item := range routes {
		if item.binding.ViewName == "" && item.binding.DefaultCommand && extendsLocalPath(item.path, local) {
			if !slices.Contains(shadowed, item.binding) {
				shadowed = append(shadowed, item.binding)
			}
			continue
		}
		kept = append(kept, item)
	}
	return kept, shadowed
}

// extendsLocalPath reports whether a proper prefix of path is a local complete path.
func extendsLocalPath(path []gocui.Key, local map[string]struct{}) bool {
	for i := 1; i < len(path); i++ {
		if _, ok := local[commandPathID(path[:i])]; ok {
			return true
		}
	}
	return false
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

// sortCommandTree orders each node's children in default declaration order.
// Keys that no default declares follow, alphabetically by label.
func sortCommandTree(tree *CommandTree, prefix []gocui.Key) {
	rank := func(key gocui.Key) (int, bool) {
		declared, ok := defaultPathRanks[commandPathID(extendCommandPath(prefix, key))]
		return declared, ok
	}
	sort.SliceStable(tree.Children, func(i, j int) bool {
		a, b := tree.Children[i], tree.Children[j]
		rankA, declaredA := rank(a.Key)
		rankB, declaredB := rank(b.Key)
		if declaredA != declaredB {
			return declaredA
		}
		if declaredA && rankA != rankB {
			return rankA < rankB
		}
		return config.LabelForKey(a.Key) < config.LabelForKey(b.Key)
	})
	for i := range tree.Children {
		sortCommandTree(tree.Children[i].Tree, extendCommandPath(prefix, tree.Children[i].Key))
	}
}

// extendCommandPath returns a new path, never sharing prefix's backing array.
func extendCommandPath(prefix []gocui.Key, key gocui.Key) []gocui.Key {
	return append(prefix[:len(prefix):len(prefix)], key)
}

// commandPathID identifies a normalized key path for defaultPathRanks.
func commandPathID(path []gocui.Key) string {
	var b strings.Builder
	for _, key := range path {
		fmt.Fprintf(&b, "%d/%d/%q;", key.KeyName(), key.Mod(), key.Str())
	}
	return b.String()
}
