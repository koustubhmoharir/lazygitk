package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var forkMochaFields = map[string][]string{
	"ActiveBorderColor":         {"#b4befe", "bold"},
	"InactiveBorderColor":       {"#a6adc8"},
	"OptionsTextColor":          {"#89b4fa"},
	"SelectedLineBgColor":       {"#313244"},
	"CherryPickedCommitBgColor": {"#45475a"},
	"CherryPickedCommitFgColor": {"#a6e3a1"},
	"MarkedBaseCommitBgColor":   {"#45475a"},
	"MarkedBaseCommitFgColor":   {"#fab387"},
	"UnstagedChangesColor":      {"#f38ba8"},
	"DefaultFgColor":            {"#cdd6f4"},
}

var forkLatteFields = map[string][]string{
	"ActiveBorderColor":         {"#7287fd", "bold"},
	"InactiveBorderColor":       {"#6c6f85"},
	"OptionsTextColor":          {"#1e66f5"},
	"SelectedLineBgColor":       {"#ccd0da"},
	"CherryPickedCommitBgColor": {"#bcc0cc"},
	"CherryPickedCommitFgColor": {"#40a02b"},
	"MarkedBaseCommitBgColor":   {"#bcc0cc"},
	"MarkedBaseCommitFgColor":   {"#fe640b"},
	"UnstagedChangesColor":      {"#d20f39"},
	"DefaultFgColor":            {"#4c4f69"},
}

// assertThemeFields checks the named fields of theme.
func assertThemeFields(t *testing.T, theme ThemeConfig, expected map[string][]string) {
	t.Helper()
	for name, want := range expected {
		assert.Equal(t, want, reflect.ValueOf(theme).FieldByName(name).Interface(), name)
	}
}

func TestForkThemePaletteMatchesDocumentedTable(t *testing.T) {
	theme := GetDefaultConfig().Gui.ThemeForBackground(false, "")
	assertThemeFields(t, theme, forkMochaFields)

	theme = GetDefaultConfig().Gui.ThemeForBackground(true, "")
	assertThemeFields(t, theme, forkLatteFields)
}

func TestForkThemePaletteDoesNotDependOnTheBackgroundColor(t *testing.T) {
	gui := GetDefaultConfig().Gui
	assertThemeFields(t, gui.ThemeForBackground(false, "#1e1e1e"), forkMochaFields)
	assertThemeFields(t, gui.ThemeForBackground(true, "#fdf6e3"), forkLatteFields)
}

func TestForkThemePaletteLeavesUserThemeFieldsAlone(t *testing.T) {
	gui := GetDefaultConfig().Gui
	gui.Theme.ActiveBorderColor = []string{"red"}
	assert.Equal(t, []string{"red"}, gui.ThemeForBackground(false, "").ActiveBorderColor)
	assert.Equal(t, []string{"red"}, gui.ThemeForBackground(true, "").ActiveBorderColor)

	// The other fields keep the palette.
	others := map[string][]string{}
	for name, value := range forkMochaFields {
		if name != "ActiveBorderColor" {
			others[name] = value
		}
	}
	assertThemeFields(t, gui.ThemeForBackground(false, ""), others)
}

func TestForkThemePaletteIsOverriddenPerBackground(t *testing.T) {
	gui := GetDefaultConfig().Gui
	gui.DarkTheme.DefaultFgColor = []string{"white"}
	assert.Equal(t, []string{"white"}, gui.ThemeForBackground(false, "").DefaultFgColor)
	assert.Equal(t, []string{"#4c4f69"}, gui.ThemeForBackground(true, "").DefaultFgColor)

	gui = GetDefaultConfig().Gui
	gui.LightTheme.OptionsTextColor = []string{"yellow"}
	assert.Equal(t, []string{"#89b4fa"}, gui.ThemeForBackground(false, "").OptionsTextColor)
	assert.Equal(t, []string{"yellow"}, gui.ThemeForBackground(true, "").OptionsTextColor)
}

func TestForkThemePaletteKeepsUpstreamDefaultsForOtherFields(t *testing.T) {
	gui := GetDefaultConfig().Gui
	dark := gui.ThemeForBackground(false, "#1e1e1e")
	assert.Equal(t, []string{"cyan", "bold"}, dark.SearchingActiveBorderColor)
	assert.Equal(t, []string{"bold"}, dark.SelectedLineFgColor)
	assert.Equal(t, []string{"#626262"}, dark.InactiveViewSelectedLineBgColor)

	light := gui.ThemeForBackground(true, "#fdf6e3")
	assert.Equal(t, []string{"cyan", "bold"}, light.SearchingActiveBorderColor)
	assert.Equal(t, []string{"#d7d1c1"}, light.InactiveViewSelectedLineBgColor)
}

func TestForkThemePaletteFieldsHaveNoStaticDefaults(t *testing.T) {
	for _, theme := range []ThemeConfig{forkThemePalette(false), forkThemePalette(true)} {
		for _, field := range setThemeFields(theme) {
			assert.Empty(t, reflect.ValueOf(GetDefaultConfig().Gui.Theme).FieldByName(field).Interface(), field)
		}
	}
}
