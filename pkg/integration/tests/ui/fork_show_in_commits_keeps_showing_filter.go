package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShowInCommitsKeepsShowingFilter = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w v on a branch whose tip a path filter already shows keeps the filter, and selects the tip",
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
		// old is a branch at m1, and m1 is the only commit that touches m1.txt.
		shell.CreateFileAndAdd("base.txt", "base").
			Commit("base").
			CreateFileAndAdd("m1.txt", "m1").
			Commit("m1").
			NewBranch("old").
			Checkout("master").
			CreateFileAndAdd("m2.txt", "m2").
			Commit("m2")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"p"})
		t.ExpectPopup().Prompt().Title(Equals("Enter path:")).Type("m1.txt").Confirm()
		commits.IsFocused().Lines(Contains("m1"))

		t.Views().Branches().Focus().NavigateToLine(Contains("old"))
		t.Views().Branches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"v"})

		// The filter still shows m1, so it stays, and m1 is selected.
		commits.IsFocused().
			Lines(Contains("m1")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("Filtering by 'm1.txt'"))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	},
})
