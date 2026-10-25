package gui

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// forkCommandLabelsForView maps each command ID to the label of its shortest
// effective route in view (ties go to the first route in walk order). Routes
// whose first key is claimed by a custom binding for the view or globally are
// skipped, as they cannot be reached. Modal views do not use command paths, so
// their labels come from the first native key of the core binding.
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

	claimed := make(map[gocui.Key]bool)
	for _, binding := range custom {
		if binding.ViewName == "" || binding.ViewName == view {
			for _, key := range binding.Keys {
				claimed[key] = true
			}
		}
	}
	shortest := make(map[string]int)
	var walk func(*keymap.CommandTree, []gocui.Key)
	walk = func(node *keymap.CommandTree, path []gocui.Key) {
		if len(path) > 0 && claimed[path[0]] {
			return
		}
		for _, binding := range node.Bindings {
			length, seen := shortest[binding.CommandID]
			if seen && length <= len(path) {
				continue
			}
			shortest[binding.CommandID] = len(path)
			labels[binding.CommandID] = types.CommandPathLabel(path)
		}
		for _, edge := range node.Children {
			next := append(append([]gocui.Key(nil), path...), edge.Key)
			walk(edge.Tree, next)
		}
	}
	walk(keymap.BuildCommandTree(core, view), nil)
	return labels
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
