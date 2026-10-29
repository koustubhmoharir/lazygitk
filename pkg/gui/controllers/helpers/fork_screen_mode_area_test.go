package helpers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestScreenModeAreaForWindow(t *testing.T) {
	userConfig := config.GetDefaultConfig()

	tests := []struct {
		window   string
		expected types.ScreenModeArea
		ok       bool
	}{
		{"main", types.SCREEN_AREA_MAIN, true},
		{"secondary", types.SCREEN_AREA_MAIN, true},
		{"files", types.SCREEN_AREA_SIDE, true},
		{"commits", types.SCREEN_AREA_SIDE, true},
		{"status", types.SCREEN_AREA_SIDE, true},
		{"extras", types.SCREEN_AREA_SIDE, false},
		{"menu", types.SCREEN_AREA_SIDE, false},
	}

	for _, test := range tests {
		area, ok := ScreenModeAreaForWindow(userConfig, test.window)
		if area != test.expected || ok != test.ok {
			t.Errorf("window %s: expected (%s, %t), got (%s, %t)", test.window, test.expected, test.ok, area, ok)
		}
	}
}

func TestScreenModeAfterFocus(t *testing.T) {
	userConfig := config.GetDefaultConfig()
	const side = types.SCREEN_AREA_SIDE
	const main = types.SCREEN_AREA_MAIN

	tests := []struct {
		name     string
		mode     types.ScreenMode
		area     types.ScreenModeArea
		window   string
		expected types.ScreenMode
		changed  bool
	}{
		{"side half, main focused keeps the mode", types.SCREEN_HALF, side, "main", types.SCREEN_HALF, false},
		{"side full, main focused shrinks to half", types.SCREEN_FULL, side, "main", types.SCREEN_HALF, true},
		{"side full, secondary focused shrinks to half", types.SCREEN_FULL, side, "secondary", types.SCREEN_HALF, true},
		{"side full, side focused keeps the mode", types.SCREEN_FULL, side, "commits", types.SCREEN_FULL, false},
		{"side half, side focused keeps the mode", types.SCREEN_HALF, side, "files", types.SCREEN_HALF, false},
		{"main half, side focused goes to normal", types.SCREEN_HALF, main, "commits", types.SCREEN_NORMAL, true},
		{"main full, side focused goes to normal", types.SCREEN_FULL, main, "files", types.SCREEN_NORMAL, true},
		{"main half, main focused keeps the mode", types.SCREEN_HALF, main, "main", types.SCREEN_HALF, false},
		{"main full, secondary focused keeps the mode", types.SCREEN_FULL, main, "secondary", types.SCREEN_FULL, false},
		{"normal, main focused keeps the mode", types.SCREEN_NORMAL, side, "main", types.SCREEN_NORMAL, false},
		{"normal, side focused keeps the mode", types.SCREEN_NORMAL, main, "files", types.SCREEN_NORMAL, false},
		{"side full, extras focused keeps the mode", types.SCREEN_FULL, side, "extras", types.SCREEN_FULL, false},
		{"main full, menu focused keeps the mode", types.SCREEN_FULL, main, "menu", types.SCREEN_FULL, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mode, changed := ScreenModeAfterFocus(test.mode, test.area, test.window, userConfig)
			if mode != test.expected || changed != test.changed {
				t.Errorf("expected (%d, %t), got (%d, %t)", test.expected, test.changed, mode, changed)
			}
		})
	}
}
