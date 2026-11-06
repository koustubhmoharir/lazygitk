package gui

import (
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// forkCommandLabelsForView maps each command ID to the label of its shortest
// effective route in view (ties go to the first route in walk order). Routes
// are those of the display projection (forkEffectiveRoutes), so a route hidden
// by an explicit leaf is not advertised, and a route whose first key is claimed
// by a custom binding for the view or globally is skipped. Modal views do not
// use command paths, so their labels come from the first native key of the core
// binding.
func forkCommandLabelsForView(core, custom []*types.Binding, view string) map[string]string {
	labels := make(map[string]string)
	if !forkCommandView(view) {
		for _, binding := range core {
			if binding.ViewName != view {
				continue
			}
			if _, ok := labels[binding.CommandID]; ok {
				continue
			}
			if keyLabels := binding.KeyLabels(); len(keyLabels) > 0 {
				labels[binding.CommandID] = keyLabels[0]
			}
		}
		return labels
	}

	var customs []*types.Binding
	for _, binding := range custom {
		if binding.ViewName == "" || binding.ViewName == view {
			customs = append(customs, binding)
		}
	}
	claimed := forkCustomClaims(customs)
	shortest := make(map[string]int)
	// Labels are cached per keybinding reset and read from worker goroutines, so
	// they cannot evaluate live applicability; forkStaticallyApplicable is used.
	for _, route := range forkEffectiveRoutes(keymap.BuildCommandTree(core, view), forkStaticallyApplicable) {
		if len(route.path) > 0 && claimed[route.path[0]] {
			continue
		}
		id := route.binding.CommandID
		length, seen := shortest[id]
		if seen && length <= len(route.path) {
			continue
		}
		shortest[id] = len(route.path)
		labels[id] = types.CommandPathLabel(route.path)
	}
	return labels
}

// forkStaticallyApplicable counts every binding as applicable. Labels describe
// the routes that the configuration makes, not the ones available at this moment.
func forkStaticallyApplicable(*types.Binding) bool {
	return true
}

// commandKeyLabel returns the effective label of commandID in view, falling back
// to the global view when view does not route it, or "" when it has no key.
// Upstream legacy key fields are not consulted: they do not describe the
// command paths that normal panes dispatch.
func (gui *Gui) commandKeyLabel(commandID, view string) string {
	gui.forkCommandLabelsMu.Lock()
	defer gui.forkCommandLabelsMu.Unlock()

	if label := gui.forkCommandLabelsLocked(view)[commandID]; label != "" {
		return label
	}
	if view != "" {
		return gui.forkCommandLabelsLocked("")[commandID]
	}
	return ""
}

// forkCommandLabelsLocked fills the per-view cache lazily. The caller must hold
// forkCommandLabelsMu.
func (gui *Gui) forkCommandLabelsLocked(view string) map[string]string {
	if labels, ok := gui.forkCommandLabelCache[view]; ok {
		return labels
	}
	labels := forkCommandLabelsForView(gui.forkCoreBindings, gui.forkCustomBindings, view)
	if gui.forkCommandLabelCache == nil {
		gui.forkCommandLabelCache = make(map[string]map[string]string)
	}
	gui.forkCommandLabelCache[view] = labels
	return labels
}

// setForkCommandBindings records the bindings used for command key labels and
// discards labels computed from earlier bindings.
func (gui *Gui) setForkCommandBindings(core, custom []*types.Binding) {
	gui.forkCommandLabelsMu.Lock()
	defer gui.forkCommandLabelsMu.Unlock()

	gui.forkCoreBindings = core
	gui.forkCustomBindings = custom
	gui.forkCommandLabelCache = make(map[string]map[string]string)
}
