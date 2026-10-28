package gocui

// Fork: an explicit set of selected lines, for the Commits pane's multi-select mode.
// That selection is not contiguous, and the cursor moves without changing it, so the
// cursor's line is not necessarily selected. It is drawn separately: with the inactive
// selection background and without the selection's text color, so it stays visible
// without looking selected. While the set is set, the range-select anchor is ignored,
// since the set is the selection.

// SetSelectedLines sets the explicit selection of the view: lines is indexed by line of
// the content (as SetInclusionGutter's marks are), and says which lines are selected.
// Every view line of a selected content line is highlighted, and the cursor's line is
// drawn separately when it is not selected. nil removes the override, and the view
// highlights the cursor line or the range select as it does without one. The view keeps
// lines, so the caller must not change it afterwards.
func (v *View) SetSelectedLines(lines []bool) {
	v.writeMutex.Lock()
	defer v.writeMutex.Unlock()

	v.selectedLines = lines
}

// SelectedLineSet returns the explicit selection set by SetSelectedLines, or nil when
// there is none.
func (v *View) SelectedLineSet() []bool {
	v.writeMutex.Lock()
	defer v.writeMutex.Unlock()

	return v.selectedLines
}

// highlightSelectedLine returns the colors of a cell under an explicit selection
// (SetSelectedLines). y is the cell's row on screen, and contentLine the line of the
// content it belongs to. The colors are only changed where the selection colors would
// be, as the range highlight changes them: a selected line gets the selection colors,
// and the cursor's line, when it is not selected, gets the inactive selection
// background alone.
func (v *View) highlightSelectedLine(x, y, contentLine, width int, fgColor, bgColor Attribute, isWindowFocused bool) (Attribute, Attribute) {
	selected := contentLine >= 0 && contentLine < len(v.selectedLines) && v.selectedLines[contentLine]
	isCursor := y == v.cy
	if !selected && !isCursor {
		return fgColor, bgColor
	}

	colorWidth := v.SelectedLineColorWidth
	if colorWidth != 0 && x >= colorWidth && x+width <= v.InnerWidth()-colorWidth {
		return fgColor, bgColor
	}

	if selected {
		fgColor = applySelTextColor(fgColor, v.SelTextColor)
		if v.HighlightInactive || !isWindowFocused {
			return fgColor, (bgColor & AttrStyleBits) | v.InactiveViewSelBgColor
		}
		return fgColor, (bgColor & AttrStyleBits) | v.SelBgColor
	}
	return fgColor, (bgColor & AttrStyleBits) | v.InactiveViewSelBgColor
}
