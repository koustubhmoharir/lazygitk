package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Panel borders describe the routes owned by the current pane, rather than the
// upstream raw key fields (whose defaults also serve legacy configuration).
func forkJumpLabelsForView(core, custom []*types.Binding, view string) map[string]string {
	labels := make(map[string]string)
	if !forkCommandView(view) {
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
	owned := make(map[*types.Binding]map[string]bool)
	var walk func(*keymap.CommandTree, []gocui.Key)
	walk = func(node *keymap.CommandTree, path []gocui.Key) {
		if len(path) > 0 && claimed[path[0]] {
			return
		}
		for _, binding := range node.Bindings {
			if binding.CommandID != "navigation.focus-diff" && !strings.HasPrefix(binding.CommandID, "navigation.focus-panel.") {
				continue
			}
			if owned[binding] == nil {
				owned[binding] = make(map[string]bool)
			}
			owned[binding][types.CommandPathLabel(path)] = true
		}
		for _, edge := range node.Children {
			next := append(append([]gocui.Key(nil), path...), edge.Key)
			walk(edge.Tree, next)
		}
	}
	walk(keymap.BuildCommandTree(core, view), nil)
	for _, binding := range core {
		if labels[binding.CommandID] != "" {
			continue
		}
		for _, label := range binding.KeyLabels() {
			if owned[binding][label] {
				labels[binding.CommandID] = label
				break
			}
		}
	}
	return labels
}

func (gui *Gui) updateForkPanelJumpLabels(core, custom []*types.Binding) {
	if gui.State == nil || gui.State.ContextMgr == nil || gui.Views.Main == nil || gui.Views.Secondary == nil {
		return
	}
	labels := forkJumpLabelsForView(core, custom, gui.State.ContextMgr.Current().GetViewName())
	prefix := func(id string) string {
		if !gui.c.UserConfig().Gui.ShowPanelJumps || labels[id] == "" {
			return ""
		}
		return fmt.Sprintf("[%s]", labels[id])
	}
	for _, panel := range gui.c.UserConfig().Gui.SidePanels {
		label := prefix("navigation.focus-panel." + panel[0])
		for _, name := range panel {
			if view, err := gui.g.View(sidePanelViewNames[name]); err == nil {
				view.TitlePrefix = label
			}
		}
	}
	gui.focusMainViewJumpLabel = prefix("navigation.focus-diff")
	if gui.State.MainPanes == types.SecondaryPaneOnly {
		gui.showFocusMainViewJumpLabelOn(gui.Views.Secondary)
	} else {
		gui.showFocusMainViewJumpLabelOn(gui.Views.Main)
	}
}

func (gui *Gui) refreshForkPanelJumpLabels() {
	if gui.State == nil || gui.State.Contexts == nil || gui.State.ContextMgr == nil || gui.CustomCommandsClient == nil || gui.git == nil {
		return
	}
	core, custom, _ := gui.initialBindingsWithCustomCommands()
	gui.updateForkPanelJumpLabels(core, custom)
}
