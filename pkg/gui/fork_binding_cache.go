package gui

import (
	"slices"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// forkBindingCache holds the bindings of the last keybinding reset, so the
// options bar and the command palette do not rebuild and project every binding
// on each frame. Dispatch already registers those same binding objects, so the
// cache shows what dispatch uses.
//
// The cache is valid only for the view current at the reset. Applicability
// (disabled reasons and command activity) is still evaluated on every call; only
// the structure is reused.
type forkBindingCache struct {
	valid         bool
	view          string
	all           []*types.Binding
	mouseBindings []*gocui.ViewMouseBinding
	// tree is the applicability-free command tree of view, set only for command
	// views; nil otherwise.
	tree *forkCommandTree
}

// storeForkBindingCache records the bindings registered by resetKeybindings.
// The binding objects are the registered ones, not copies. Custom bindings come
// first in all, as the palette and options bar have always listed them.
func (gui *Gui) storeForkBindingCache(bindings, customBindings []*types.Binding, mouseBindings []*gocui.ViewMouseBinding) {
	view := gui.State.ContextMgr.CurrentStatic().GetViewName()
	all := make([]*types.Binding, 0, len(customBindings)+len(bindings))
	all = append(all, customBindings...)
	all = append(all, bindings...)
	cache := forkBindingCache{
		valid:         true,
		view:          view,
		all:           slices.Clip(all),
		mouseBindings: slices.Clip(mouseBindings),
	}
	if forkCommandView(view) {
		tree := buildForkCommandTree(cache.all, view)
		cache.tree = &tree
	}
	gui.forkBindingCache = cache
}

// clearForkBindingCache drops the cache. It must be called wherever the
// controllers or the repo state are replaced, since the cached bindings capture
// them.
func (gui *Gui) clearForkBindingCache() {
	gui.forkBindingCache = forkBindingCache{}
}

// cachedForkKeybindings returns the cached result of
// GetInitialKeybindingsWithCustomCommands when the cache was made for view.
// ok is false when the caller must rebuild.
func (gui *Gui) cachedForkKeybindings(view string) (bindings []*types.Binding, mouseBindings []*gocui.ViewMouseBinding, ok bool) {
	cache := &gui.forkBindingCache
	if !cache.valid || cache.view != view {
		return nil, nil, false
	}
	if cache.tree != nil {
		return cache.tree.displayBindings(cache.all, view), cache.mouseBindings, true
	}
	return cache.all, cache.mouseBindings, true
}
