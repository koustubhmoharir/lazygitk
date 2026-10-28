package gocui

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
)

// Fork: MockTerminalTitle lets the integration harness read the window title
// that lazygitk set with Screen.SetTitle (roadmap item 23). It is for the test
// harness only. It returns "" unless the screen is the simulated terminal that
// tcellInitSimulation creates, so the real terminal path never reads it.
func MockTerminalTitle() string {
	withTty, ok := Screen.(interface{ Tty() (tcell.Tty, bool) })
	if !ok {
		return ""
	}
	tty, _ := withTty.Tty()
	mockTerm, ok := tty.(vt.MockTerm)
	if !ok {
		return ""
	}
	return mockTerm.GetTitle()
}
