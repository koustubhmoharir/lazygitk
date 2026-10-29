package gui

import (
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the screen mode belongs to an area (see types/fork_screen_mode_area.go).
// Focus never changes the layout by itself, with one exception: when focus moves
// to a window that the current arrangement hides, the mode shrinks just enough to
// show it. ContextMgr.Activate calls applyScreenModeForFocus for every focus
// change, so keys, clicks and programmatic focus all behave the same.

func (self *GuiRepoState) GetScreenModeArea() types.ScreenModeArea {
	return self.ScreenModeArea
}

func (self *GuiRepoState) SetScreenModeInArea(mode types.ScreenMode, area types.ScreenModeArea) {
	self.ScreenMode = mode
	self.ScreenModeArea = area
}

// applyScreenModeForFocus shrinks the screen mode when c is in an area that the
// current arrangement hides. Popups and the other windows that belong to neither
// area never change it.
func (gui *Gui) applyScreenModeForFocus(c types.Context) {
	mode, changed := helpers.ScreenModeAfterFocus(
		gui.State.ScreenMode, gui.State.ScreenModeArea, c.GetWindowName(), gui.c.UserConfig())
	if !changed {
		return
	}

	gui.State.SetScreenModeInArea(mode, gui.State.GetScreenModeArea())
	// The rendering of the contents that depend on the screen mode (for example
	// the authorship column of the commits) must follow the new mode, just as it
	// does when the user changes the mode (controllers/screen_mode_actions.go).
	for _, context := range gui.State.ContextMgr.AllList() {
		if context.NeedsRerenderOnWidthChange() == types.NEEDS_RERENDER_ON_WIDTH_CHANGE_WHEN_SCREEN_MODE_CHANGES {
			context.HandleRender()
		}
	}
}

// The integration test driver reads the mode the same way it reads the current
// context: on the test goroutine, while the gui is idle.
func (self *GuiDriver) ScreenMode() types.ScreenMode {
	return self.gui.State.GetScreenMode()
}

func (self *GuiDriver) ScreenModeArea() types.ScreenModeArea {
	return self.gui.State.GetScreenModeArea()
}
