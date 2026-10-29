package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the + and - keys record the area of the focused window with the new
// mode (types/fork_screen_mode_area.go). When the focused window belongs to
// neither area (for example the extras window), the area stays as it was.
func (self *ScreenModeActions) setScreenModeInFocusedArea(mode types.ScreenMode) {
	repoState := self.c.State().GetRepoState()
	area := repoState.GetScreenModeArea()
	if focusedArea, ok := helpers.ScreenModeAreaForWindow(
		self.c.UserConfig(), self.c.Context().CurrentStatic().GetWindowName()); ok {
		area = focusedArea
	}
	repoState.SetScreenModeInArea(mode, area)
}

// focusedAreaScreenMode is the mode that + cycles from: the current mode when the
// focused window is in the enlarged area, and normal when it is in the other one,
// so that + in the main views while the side panels are enlarged enlarges the main
// views one step instead of jumping to full screen. - needs no such rule: the only
// way to be focused outside the enlarged area is a half side area (a hidden area
// shrinks the mode when it gets focus), and one step down from it is normal anyway.
func (self *ScreenModeActions) focusedAreaScreenMode() types.ScreenMode {
	repoState := self.c.State().GetRepoState()
	focusedArea, ok := helpers.ScreenModeAreaForWindow(
		self.c.UserConfig(), self.c.Context().CurrentStatic().GetWindowName())
	if ok && focusedArea != repoState.GetScreenModeArea() {
		return types.SCREEN_NORMAL
	}
	return repoState.GetScreenMode()
}
