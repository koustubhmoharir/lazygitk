package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkScreenModeFullSideFocusMain = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A full side area gives way to half when the patch is focused, so the patch shows beside the side panels",
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

		commits.Press(keys.Universal.NextScreenMode)
		commits.Press(keys.Universal.NextScreenMode)
		t.ExpectScreenMode(types.SCREEN_FULL, types.SCREEN_AREA_SIDE)
		t.Views().Main().HasNoScreenArea()

		// The full side area hides the patch, so focusing it shrinks the mode to
		// half: the patch is shown beside the side panels, and it has the focus.
		t.GlobalPress(keys.Universal.FocusMainView)
		t.Views().Main().IsFocused()
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
		t.Views().Main().HasScreenArea()
		t.Views().Commits().HasScreenArea()
	},
})
