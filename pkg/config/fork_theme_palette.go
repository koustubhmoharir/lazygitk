package config

// forkThemePalette returns the Catppuccin colours that the fork uses for the
// theme fields that gui.theme doesn't set: Mocha on a dark background and Latte
// on a light one. ThemeForBackground layers it over gui.theme and under
// gui.darkTheme / gui.lightTheme, and over upstream's background-derived
// defaults for the same fields.
//
// The fields are the ones in the roadmap's item 15 table. GetDefaultConfig
// leaves them empty, see applyForkDefaults in fork_defaults.go.
func forkThemePalette(lightBackground bool) ThemeConfig {
	if lightBackground {
		return ThemeConfig{
			ActiveBorderColor:         []string{"#7287fd", "bold"},     // Lavender
			InactiveBorderColor:       []string{"#6c6f85"},             // Subtext0
			OptionsTextColor:          []string{ForkPaletteBlue(true)}, // Blue
			SelectedLineBgColor:       []string{"#ccd0da"},             // Surface0
			CherryPickedCommitBgColor: []string{"#bcc0cc"},             // Surface1
			CherryPickedCommitFgColor: []string{"#40a02b"},             // Green
			MarkedBaseCommitBgColor:   []string{"#bcc0cc"},             // Surface1
			MarkedBaseCommitFgColor:   []string{"#fe640b"},             // Peach
			UnstagedChangesColor:      []string{"#d20f39"},             // Red
			DefaultFgColor:            []string{"#4c4f69"},             // Text
		}
	}

	return ThemeConfig{
		ActiveBorderColor:         []string{"#b4befe", "bold"},      // Lavender
		InactiveBorderColor:       []string{"#a6adc8"},              // Subtext0
		OptionsTextColor:          []string{ForkPaletteBlue(false)}, // Blue
		SelectedLineBgColor:       []string{"#313244"},              // Surface0
		CherryPickedCommitBgColor: []string{"#45475a"},              // Surface1
		CherryPickedCommitFgColor: []string{"#a6e3a1"},              // Green
		MarkedBaseCommitBgColor:   []string{"#45475a"},              // Surface1
		MarkedBaseCommitFgColor:   []string{"#fab387"},              // Peach
		UnstagedChangesColor:      []string{"#f38ba8"},              // Red
		DefaultFgColor:            []string{"#cdd6f4"},              // Text
	}
}

// ForkPaletteBlue returns the palette's Blue: Mocha's on a dark background and
// Latte's on a light one. Besides the options text, the fork uses it for the
// text that upstream and delta colour with the terminal's basic blue (see
// pkg/theme/fork_basic_colors.go and forkDeltaCommand).
func ForkPaletteBlue(lightBackground bool) string {
	if lightBackground {
		return "#1e66f5"
	}
	return "#89b4fa"
}
