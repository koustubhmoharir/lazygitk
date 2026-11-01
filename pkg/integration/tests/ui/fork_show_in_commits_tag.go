package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShowInCommitsTag = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w v on a tag from Tags shows the tag's commit in Commits, selected, and makes the side area half",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.Gui.ShowRootItemInFileTree = true
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("base.txt", "base").
			Commit("base").
			CreateLightweightTag("v1", "HEAD").
			CreateFileAndAdd("m1.txt", "m1").
			Commit("m1").
			CreateFileAndAdd("m2.txt", "m2").
			Commit("m2")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Tags().Focus().NavigateToLine(Contains("v1"))
		t.Views().Tags().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"v"})

		t.Views().Commits().IsFocused().
			Lines(Contains("m2"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("base"))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	},
})
