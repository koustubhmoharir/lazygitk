package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/stretchr/testify/assert"
)

func TestFocusFrameStyle(t *testing.T) {
	heavy := frameStyle{runes: []rune{'━', '┃', '┏', '┓', '┗', '┛'}, teeLeft: '┣', teeRight: '┫'}
	hidden := frameStyle{runes: []rune{' ', ' ', ' ', ' ', ' ', ' '}, teeLeft: ' ', teeRight: ' '}
	testCases := []struct {
		border   string
		focused  bool
		expected frameStyle
	}{
		{"", true, heavy},
		{"bold", true, heavy},
		{"rounded", true, heavy},
		{"double", true, heavy},
		{"hidden", true, hidden},
		{"", false, frameStyle{runes: []rune("─│┌┐└┘"), teeLeft: '├', teeRight: '┤'}},
		{"rounded", false, frameStyle{runes: []rune("─│╭╮╰╯"), teeLeft: '├', teeRight: '┤'}},
		{"double", false, frameStyle{runes: []rune("═║╔╗╚╝"), teeLeft: '╠', teeRight: '╣'}},
		{"bold", false, heavy},
		{"hidden", false, hidden},
	}
	for _, tc := range testCases {
		t.Run(tc.border, func(t *testing.T) {
			assert.Equal(t, tc.expected, focusFrameStyle(tc.border, tc.focused))
		})
	}
}

func TestSharesFocus(t *testing.T) {
	menu := &gocui.View{}
	menuFilterFrame := &gocui.View{ParentView: menu}
	menuFilter := &gocui.View{ParentView: menu}
	main := &gocui.View{}

	assert.True(t, sharesFocus(menu, menu))
	assert.True(t, sharesFocus(menuFilterFrame, menu), "the filter row is focused with its menu")
	assert.True(t, sharesFocus(menuFilter, menuFilterFrame))
	assert.False(t, sharesFocus(main, menu))
	assert.False(t, sharesFocus(menu, nil), "no view has the focus")
}
