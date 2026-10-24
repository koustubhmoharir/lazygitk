package gui

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// registerForkCommands compiles normal-pane routes. Popup, editor, mouse and
// custom registrations retain their native dispatch. Call only with projected
// core bindings: each complete path denotes one semantic action in that pane.
func (gui *Gui) registerForkCommands(core, custom []*types.Binding) {
	views := map[string]bool{}
	var all []*types.Binding
	for _, binding := range core {
		if !forkCommandView(binding.ViewName) {
			gui.SetKeybinding(binding)
			continue
		}
		for _, key := range binding.Keys {
			if forkMouseKey(key) {
				copy := *binding
				copy.Keys = []gocui.Key{key}
				gui.SetKeybinding(&copy)
			}
		}
		if binding.ViewName != "" {
			views[binding.ViewName] = true
		} else {
			gui.setNativeGlobalKeys(binding)
		}
		copy := *binding
		copy.ViewName = ""
		all = append(all, &copy)
	}
	for _, binding := range custom {
		gui.SetKeybinding(binding)
	}
	roots := keymap.BuildCommandTree(all, "")
	for view := range views {
		// Native local keys normally precede global keys. Copy global custom routes
		// into the normal-pane scope before compiled defaults so the new local
		// no-op reservations cannot mask an explicitly configured global command.
		for _, binding := range custom {
			if binding.ViewName == "" {
				copy := *binding
				copy.ViewName = view
				gui.SetKeybinding(&copy)
			}
		}
		tree := keymap.BuildCommandTree(core, view)
		for _, root := range roots.Children {
			node := &keymap.CommandTree{}
			for _, edge := range tree.Children {
				if edge.Key.Equals(root.Key) {
					node = edge.Tree
					break
				}
			}
			gui.g.SetCommandKeybinding(view, root.Key, func(*gocui.Gui, *gocui.View) error {
				return gui.startForkCommand(node, root.Key)
			})
		}
	}
}

// setNativeGlobalKeys makes a global core binding reachable from views that
// have no compiled command tree (menus, prompts, confirmations, ...). Upstream
// registered these single keys natively, so gocui's global fallback reaches
// them there. Only direct keys are registered: command paths are compiled into
// fork-view trees. The guard keeps fork views unchanged: their compiled roots
// own every global key, and a fork view without its own bindings must not
// gain a new global meaning.
func (gui *Gui) setNativeGlobalKeys(binding *types.Binding) {
	for _, key := range binding.Keys {
		if forkMouseKey(key) {
			continue // Mouse routes are registered by registerForkCommands.
		}
		gui.g.SetKeybinding("", key, func(_ *gocui.Gui, v *gocui.View) error {
			if v != nil && forkCommandView(v.Name()) {
				return nil
			}
			return gui.callKeybindingHandler(binding)
		})
	}
}

func (gui *Gui) startForkCommand(node *keymap.CommandTree, key gocui.Key) error {
	origin := gui.captureForkCommandOrigin()
	flow := &forkPrefixFlow{
		valid:       func() bool { return origin.valid(gui) },
		currentMenu: func() bool { return gui.State.ContextMgr.Current() == gui.State.Contexts.Menu },
		restored:    func() bool { return gui.State.ContextMgr.Current() == origin.context },
		openMenu:    gui.createMenu,
		invoke:      gui.callKeybindingHandler,
	}
	gui.invalidateForkPrefix()
	gui.forkPrefix = flow
	return flow.run(freezeForkCommandTree(node), []gocui.Key{key}, nil)
}

func (gui *Gui) invalidateForkPrefix() {
	if gui.forkPrefix != nil {
		gui.forkPrefix.invalidate()
	}
}
