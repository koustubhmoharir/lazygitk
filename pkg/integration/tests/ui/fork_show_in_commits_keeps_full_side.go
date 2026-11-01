package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShowInCommitsKeepsFullSide = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w v leaves a full side area full, and selects the commit",
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
			NewBranch("old").
			Checkout("master").
			CreateFileAndAdd("m1.txt", "m1").
			Commit("m1")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		commits.Press(keys.Universal.NextScreenMode)
		commits.Press(keys.Universal.NextScreenMode)
		t.ExpectScreenMode(types.SCREEN_FULL, types.SCREEN_AREA_SIDE)

		t.Views().Branches().Focus().NavigateToLine(Contains("old"))
		t.Views().Branches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"v"})

		commits.IsFocused().
			Lines(Contains("m1"), Contains("base")).
			SelectedLines(Contains("base"))
		t.ExpectScreenMode(types.SCREEN_FULL, types.SCREEN_AREA_SIDE)
	},
})
