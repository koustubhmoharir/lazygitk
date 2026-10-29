package theme

import (
	"testing"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/stretchr/testify/assert"
)

func TestUpdateBasicColorsUsesThePaletteBlue(t *testing.T) {
	original := style.FgBlue
	t.Cleanup(func() { style.FgBlue = original })

	UpdateBasicColors(false)
	assert.Equal(t, style.New().SetFg(style.NewRGBColor(color.HEX("#89b4fa"))), style.FgBlue)

	UpdateBasicColors(true)
	assert.Equal(t, style.New().SetFg(style.NewRGBColor(color.HEX("#1e66f5"))), style.FgBlue)

	// "blue" in the user's theme config still means the terminal's blue.
	assert.Equal(t, style.New().SetFg(style.NewBasicColor(color.FgBlue)), GetTextStyle([]string{"blue"}, false))
}
