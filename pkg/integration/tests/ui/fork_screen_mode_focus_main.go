package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkScreenModeFocusMain = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Enlarging the side area with + stays enlarged when the patch is focused with 0; + in the patch then enlarges the main area",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", "line 1\nline 2\n").
			Commit("first").
			CreateFileAndAdd("file.txt", "line 1\nline 2\nline 3\n").
			Commit("second")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		t.ExpectScreenMode(types.SCREEN_NORMAL, types.SCREEN_AREA_SIDE)

		commits.Press(keys.Universal.NextScreenMode)
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)

		// 0 focuses the patch. The side area stays enlarged, so the side panels
		// are still shown beside the patch.
		t.GlobalPress(keys.Universal.FocusMainView)
		t.Views().Main().IsFocused()
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
		t.Views().Commits().HasScreenArea()
		t.Views().Main().HasScreenArea()

		// + now enlarges the area of the focused patch one step, starting from normal:
		// the side panels go away.
		t.Views().Main().Press(keys.Universal.NextScreenMode)
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_MAIN)
		t.Views().Commits().HasNoScreenArea()
		t.Views().Main().HasScreenArea()
	},
})
