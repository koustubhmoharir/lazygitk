package gui

import (
	"github.com/go-errors/errors"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// Fork: writeClipboardOSC52 is the terminal clipboard writer for copy actions
// (roadmap item 21). tcell's SetClipboard emits OSC 52 with base64 text under
// the screen lock, so the sequence never lands inside a frame being drawn.
// The screen is read at call time, so the order of setup does not matter.
func writeClipboardOSC52(text string) error {
	if gocui.Screen == nil {
		return errors.New("the terminal screen is not running, so the text cannot be copied")
	}
	gocui.Screen.SetClipboard([]byte(text))
	return nil
}
