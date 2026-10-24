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
	core := make([]*types.Binding, 0, len(bindings))
	customCopies := make(map[*types.Binding]*types.Binding)
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		if strings.HasPrefix(binding.GetCommandID(), "custom.") {
			customCopies[binding] = cloneForkBinding(binding)
			continue
		}
		core = append(core, binding)
	}
	tree := keymap.BuildCommandTree(core, view)
	effective := make(map[*types.Binding][][]gocui.Key)
	var walk func(*keymap.CommandTree, []gocui.Key)
	walk = func(node *keymap.CommandTree, path []gocui.Key) {
		active := forkApplicableBindings(node.Bindings)
		leaf := len(active) > 0 && !active[0].DefaultCommand
		if leaf && !forkExplicitDescendants(node) {
			for _, binding := range node.Bindings {
				effective[binding] = append(effective[binding], append([]gocui.Key(nil), path...))
			}
			return
		}
		for _, binding := range node.Bindings {
			effective[binding] = append(effective[binding], append([]gocui.Key(nil), path...))
		}
		for _, edge := range node.Children {
			next := append(append([]gocui.Key(nil), path...), edge.Key)
			walk(edge.Tree, next)
		}
	}
	walk(tree, nil)

	result := make([]*types.Binding, 0, len(bindings))
	var eligibleCustoms, eligibleCore []*types.Binding
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		copy := *binding
		if customCopy := customCopies[binding]; customCopy != nil {
			copy = *customCopy
			if binding.ViewName == "" || binding.ViewName == view {
				eligibleCustoms = append(eligibleCustoms, &copy)
			}
		} else if forkCommandView(binding.ViewName) && (binding.ViewName == "" || binding.ViewName == view) {
			copy.SetCommandPaths(effective[binding])
			eligibleCore = append(eligibleCore, &copy)
		}
		result = append(result, &copy)
	}
	clearForkPathsShadowedByCustoms(eligibleCore, eligibleCustoms)
	return result
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

// Custom command roots use native exact event matching. Preserve case
// distinctions here, matching the custom dispatcher rather than core routing.
func clearForkPathsShadowedByCustoms(bindings, customs []*types.Binding) {
	claimed := make(map[gocui.Key]bool)
	for _, binding := range customs {
		if binding == nil || !strings.HasPrefix(binding.GetCommandID(), "custom.") {
			continue
		}
		for _, key := range binding.Keys {
			if !gocui.IsMouseKey(key) {
				claimed[key] = true
			}
		}
	}
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
