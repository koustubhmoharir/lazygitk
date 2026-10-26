package gocui

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
)

// Fork: MockClipboardText lets the integration harness read what a copy wrote
// through OSC 52 (roadmap item 21). It is for the test harness only. It returns
// "" unless the screen is the simulated terminal that tcellInitSimulation
// creates, so the real terminal path never reads it.
func MockClipboardText() string {
	withTty, ok := Screen.(interface{ Tty() (tcell.Tty, bool) })
	if !ok {
		return ""
	}
	tty, _ := withTty.Tty()
	mockTerm, ok := tty.(vt.MockTerm)
	if !ok {
		return ""
	}
	return string(mockTerm.Backend().GetClipboard())
}
