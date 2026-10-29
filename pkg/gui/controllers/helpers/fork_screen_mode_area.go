package helpers

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: the screen mode's area (types/fork_screen_mode_area.go). These are the
// rules that the layout and the focus hook share.

// ScreenModeAreaForWindow returns the area of a static window: the main and
// secondary views are the main area, and the side panels are the side area. ok
// is false for every other window (the extras window, popups, ...), which belong
// to neither area.
func ScreenModeAreaForWindow(userConfig *config.UserConfig, window string) (area types.ScreenModeArea, ok bool) {
	if window == "main" || window == "secondary" {
		return types.SCREEN_AREA_MAIN, true
	}
	if lo.Contains(sideWindowNames(userConfig), window) {
		return types.SCREEN_AREA_SIDE, true
	}
	return types.SCREEN_AREA_SIDE, false
}

// ScreenModeAfterFocus returns the screen mode that focusing window requires, and
// whether it differs from mode. Only a window in the hidden area can change it:
// a side-area FULL layout hides the main views, so focusing one shrinks it to
// HALF; a main-area HALF or FULL layout hides the side panels, so focusing one
// returns to NORMAL. Focusing a window in the enlarged area, or a window that
// belongs to neither area, leaves the mode alone.
func ScreenModeAfterFocus(mode types.ScreenMode, area types.ScreenModeArea, window string, userConfig *config.UserConfig) (types.ScreenMode, bool) {
	focusedArea, ok := ScreenModeAreaForWindow(userConfig, window)
	if !ok || focusedArea == area || mode == types.SCREEN_NORMAL {
		return mode, false
	}

	if area == types.SCREEN_AREA_SIDE && mode == types.SCREEN_FULL && focusedArea == types.SCREEN_AREA_MAIN {
		return types.SCREEN_HALF, true
	}
	if area == types.SCREEN_AREA_MAIN && focusedArea == types.SCREEN_AREA_SIDE {
		return types.SCREEN_NORMAL, true
	}
	return mode, false
}
