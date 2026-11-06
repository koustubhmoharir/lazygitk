package gui

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// effectiveForkDisplayBindings returns private copies with only routes that
// remain visible after local/global ownership and prefix dispatch are applied.
// Definitions stay in the palette even when their displayed route is shadowed.
func effectiveForkDisplayBindings(bindings []*types.Binding, view string) []*types.Binding {
	return buildForkCommandTree(bindings, view).displayBindings(bindings, view)
}

// forkCommandTree is the structure step of effectiveForkDisplayBindings: the
// command tree of a view's core bindings, and the custom bindings among the
// bindings it was built from. It depends only on keys, command paths, view
// names and default flags, not on applicability, so it can be built once per
// keybinding reset (see fork_binding_cache.go).
type forkCommandTree struct {
	tree   *keymap.CommandTree
	custom map[*types.Binding]bool
}

func buildForkCommandTree(bindings []*types.Binding, view string) forkCommandTree {
	core := make([]*types.Binding, 0, len(bindings))
	custom := make(map[*types.Binding]bool)
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		if strings.HasPrefix(binding.GetCommandID(), "custom.") {
			custom[binding] = true
			continue
		}
		core = append(core, binding)
	}
	return forkCommandTree{tree: keymap.BuildCommandTree(core, view), custom: custom}
}

// displayBindings is the per-call step of effectiveForkDisplayBindings. It
// evaluates applicability live, so a route follows the current selection and
// mode. It returns fresh copies and never mutates bindings or the tree.
func (structure forkCommandTree) displayBindings(bindings []*types.Binding, view string) []*types.Binding {
	effective := make(map[*types.Binding][][]gocui.Key)
	for _, route := range forkEffectiveRoutes(structure.tree, forkApplicableNow) {
		effective[route.binding] = append(effective[route.binding], route.path)
	}

	result := make([]*types.Binding, 0, len(bindings))
	var eligibleCustoms, eligibleCore []*types.Binding
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		var copy *types.Binding
		if structure.custom[binding] {
			copy = cloneForkBinding(binding)
			if binding.ViewName == "" || binding.ViewName == view {
				eligibleCustoms = append(eligibleCustoms, copy)
			}
		} else {
			value := *binding
			copy = &value
			if forkCommandView(binding.ViewName) && (binding.ViewName == "" || binding.ViewName == view) {
				copy.SetCommandPaths(effective[binding])
				eligibleCore = append(eligibleCore, copy)
			}
		}
		result = append(result, copy)
	}
	clearForkPathsShadowedByCustoms(eligibleCore, eligibleCustoms)
	return result
}

// forkRoute is one path by which dispatch reaches a binding.
type forkRoute struct {
	binding *types.Binding
	path    []gocui.Key
}

// forkEffectiveRoutes is the one traversal of prefix ownership, in walk order.
// Every binding of a node is routed at that node's path. An owner at a node
// (applicable under the given rule) hides the node's default descendants unless
// the subtree has explicit ones too, as forkPrefixFlow.run does. The rule
// decides who owns: live applicability for the display projection, and a
// static one for command labels (see forkCommandLabelsForView).
func forkEffectiveRoutes(tree *keymap.CommandTree, applicable func(*types.Binding) bool) []forkRoute {
	var routes []forkRoute
	var walk func(*keymap.CommandTree, []gocui.Key)
	walk = func(node *keymap.CommandTree, path []gocui.Key) {
		for _, binding := range node.Bindings {
			routes = append(routes, forkRoute{binding: binding, path: append([]gocui.Key(nil), path...)})
		}
		owners := forkOwnersWhere(node.Bindings, applicable)
		if len(owners) > 0 && !owners[0].DefaultCommand && !forkExplicitDescendants(node) {
			return
		}
		for _, edge := range node.Children {
			walk(edge.Tree, append(append([]gocui.Key(nil), path...), edge.Key))
		}
	}
	walk(tree, nil)
	return routes
}

func cloneForkBinding(binding *types.Binding) *types.Binding {
	copy := *binding
	copy.Keys = append([]gocui.Key(nil), binding.Keys...)
	copy.CommandPaths = make([][]gocui.Key, len(binding.CommandPaths))
	for i, path := range binding.CommandPaths {
		copy.CommandPaths[i] = append([]gocui.Key(nil), path...)
	}
	return &copy
}

func deduplicateForkDisplayAliases(bindings []*types.Binding) []*types.Binding {
	result := make([]*types.Binding, 0, len(bindings))
	indices := make(map[string]int)
	for _, binding := range bindings {
		if binding == nil || binding.CommandActive != nil && !binding.CommandActive() {
			continue
		}
		id := binding.GetCommandID()
		if id == "" {
			result = append(result, binding)
			continue
		}
		if index, exists := indices[id]; exists {
			previous := result[index]
			paths := appendForkDisplayPaths(previous, binding)
			copy := cloneForkBinding(previous)
			copy.SetCommandPaths(paths)
			result[index] = copy
			continue
		}
		indices[id] = len(result)
		result = append(result, binding)
	}
	return result
}

func appendForkDisplayPaths(first, second *types.Binding) [][]gocui.Key {
	var paths [][]gocui.Key
	appendPath := func(path []gocui.Key) {
		for _, old := range paths {
			if equalForkDisplayPath(old, path) {
				return
			}
		}
		paths = append(paths, append([]gocui.Key(nil), path...))
	}
	for _, binding := range []*types.Binding{first, second} {
		for _, key := range binding.Keys {
			appendPath([]gocui.Key{key})
		}
		for _, path := range binding.CommandPaths {
			appendPath(path)
		}
	}
	return paths
}

func equalForkDisplayPath(first, second []gocui.Key) bool {
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if !first[i].Equals(second[i]) {
			return false
		}
	}
	return true
}

// forkCustomClaims returns the keys that custom command roots claim. Custom
// roots use native exact event matching, so their case is preserved here,
// matching the custom dispatcher rather than core routing. Mouse keys are not
// claimed.
func forkCustomClaims(customs []*types.Binding) map[gocui.Key]bool {
	claimed := make(map[gocui.Key]bool)
	for _, binding := range customs {
		if binding == nil {
			continue
		}
		for _, key := range binding.Keys {
			if !gocui.IsMouseKey(key) {
				claimed[key] = true
			}
		}
	}
	return claimed
}

func clearForkPathsShadowedByCustoms(bindings, customs []*types.Binding) {
	claimed := forkCustomClaims(customs)
	for _, binding := range bindings {
		if binding == nil || strings.HasPrefix(binding.GetCommandID(), "custom.") {
			continue
		}
		paths := make([][]gocui.Key, 0, len(binding.Keys)+len(binding.CommandPaths))
		for _, key := range binding.Keys {
			if !claimed[key] {
				paths = append(paths, []gocui.Key{key})
			}
		}
		for _, path := range binding.CommandPaths {
			if len(path) > 0 && !claimed[path[0]] {
				paths = append(paths, path)
			}
		}
		binding.SetCommandPaths(paths)
	}
}
