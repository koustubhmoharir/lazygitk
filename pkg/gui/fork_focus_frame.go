package gui

import "github.com/jesseduffield/lazygit/pkg/gocui"

// Fork: the focused view draws a heavy frame, so that the focus shows in the
// shape of the frame as well as in its colour. The other views keep the frame
// that gui.border configures. Upstream gave every view the frame of gui.border.
// The border switch that used to be in configureViewProperties is now
// focusFrameStyle, and configureViewProperties and ContextMgr.Activate call
// updateFocusFrames.

// frameStyle holds the runes of a view frame. runes are in the order that gocui
// expects: horizontal and vertical edge, then the top left, top right, bottom
// left and bottom right corner. teeLeft and teeRight are the top corners of a
// frame that hangs off the bottom of another view.
type frameStyle struct {
	runes             []rune
	teeLeft, teeRight rune
}

// focusFrameStyle returns the frame of a view for the given gui.border, and
// whether the view is focused. A hidden frame stays hidden when focused.
func focusFrameStyle(border string, focused bool) frameStyle {
	switch {
	case border == "hidden":
		return frameStyle{runes: []rune{' ', ' ', ' ', ' ', ' ', ' '}, teeLeft: ' ', teeRight: ' '}
	case focused:
		return frameStyle{runes: []rune{'━', '┃', '┏', '┓', '┗', '┛'}, teeLeft: '┣', teeRight: '┫'}
	}

	switch border {
	case "double":
		return frameStyle{runes: []rune{'═', '║', '╔', '╗', '╚', '╝'}, teeLeft: '╠', teeRight: '╣'}
	case "rounded":
		return frameStyle{runes: []rune{'─', '│', '╭', '╮', '╰', '╯'}, teeLeft: '├', teeRight: '┤'}
	case "bold":
		return frameStyle{runes: []rune{'━', '┃', '┏', '┓', '┗', '┛'}, teeLeft: '┣', teeRight: '┫'}
	default:
		return frameStyle{runes: []rune{'─', '│', '┌', '┐', '└', '┘'}, teeLeft: '├', teeRight: '┤'}
	}
}

// sharesFocus reports whether v is drawn as focused while current is the current
// view. This is the same rule that gocui uses for the active frame colour: views
// that are embedded in one another (see View.ParentView) are focused together.
func sharesFocus(v *gocui.View, current *gocui.View) bool {
	return current != nil && outermostView(v) == outermostView(current)
}

func outermostView(v *gocui.View) *gocui.View {
	for v.ParentView != nil {
		v = v.ParentView
	}
	return v
}

// updateFocusFrames sets the frame runes of every view from gui.border and from
// the current view. It must run whenever the current view or gui.border changes.
func (gui *Gui) updateFocusFrames() {
	border := gui.c.UserConfig().Gui.Border
	current := gui.g.CurrentView()
	for _, mapping := range gui.orderedViewNameMappings() {
		view := *mapping.viewPtr
		view.FrameRunes = focusFrameStyle(border, sharesFocus(view, current)).runes
	}

	// The filter row's frame hangs off the bottom of the menu, so its top corners
	// are tees.
	style := focusFrameStyle(border, sharesFocus(gui.Views.MenuFilterFrame, current))
	gui.Views.MenuFilterFrame.FrameRunes = frameRunesWithTopCorners(style.runes, style.teeLeft, style.teeRight)
}
