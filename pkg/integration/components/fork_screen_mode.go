package components

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: asserts the screen mode and the area it applies to (types/fork_screen_mode_area.go).
// Use it together with view visibility to check the layout: a view that the mode
// hides is invisible, and an enlarged view is visible with the rest of its area.
func (self *TestDriver) ExpectScreenMode(mode types.ScreenMode, area types.ScreenModeArea) *TestDriver {
	self.assertWithRetries(func() (bool, string) {
		actualMode := self.gui.ScreenMode()
		actualArea := self.gui.ScreenModeArea()
		return actualMode == mode && actualArea == area,
			fmt.Sprintf("Expected screen mode %d in the %s area, but it was %d in the %s area", mode, area, actualMode, actualArea)
	})

	return self
}

// Fork: asserts whether the view takes up any space on the screen. The layout
// gives a panel that the screen mode hides a zero-sized box (its far edge lies
// before its near edge), and the view stays Visible, so IsInvisible can't tell a
// hidden panel from a shown one.
func (self *ViewDriver) HasNoScreenArea() *ViewDriver {
	self.t.assertWithRetries(func() (bool, string) {
		x0, y0, x1, y1 := self.screenDimensions()
		return x1 < x0 || y1 < y0,
			fmt.Sprintf("%s: Expected view to take up no space on the screen, but it spans (%d,%d)-(%d,%d)", self.context, x0, y0, x1, y1)
	})

	return self
}

// Fork: asserts that the view takes up some space on the screen (see HasNoScreenArea).
func (self *ViewDriver) HasScreenArea() *ViewDriver {
	self.t.assertWithRetries(func() (bool, string) {
		x0, y0, x1, y1 := self.screenDimensions()
		return x1 >= x0 && y1 >= y0,
			fmt.Sprintf("%s: Expected view to take up space on the screen, but it spans (%d,%d)-(%d,%d)", self.context, x0, y0, x1, y1)
	})

	return self
}

func (self *ViewDriver) screenDimensions() (x0, y0, x1, y1 int) {
	self.t.gui.OnUIThreadAndWait(func() {
		x0, y0, x1, y1 = self.getView().Dimensions()
	})
	return x0, y0, x1, y1
}
