package theme

import (
	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
)

// UpdateBasicColors replaces the basic blue that upstream's presentation code
// hard-codes (commit dates, reflog and rebasing hashes, remote branch counts,
// echoed commands) with the Catppuccin Blue of the fork's palette. The
// terminal's ANSI blue is too dark to read on a black background in many
// terminal palettes.
//
// style.ColorMap keeps its own copy of the basic blue, so "blue" in the user's
// theme config still means the terminal's blue.
func UpdateBasicColors(lightBackground bool) {
	style.FgBlue = style.New().SetFg(style.NewRGBColor(color.HEX(config.ForkPaletteBlue(lightBackground))))
}
