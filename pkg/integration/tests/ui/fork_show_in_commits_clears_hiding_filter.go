package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShowInCommitsClearsHidingFilter = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w v on a branch whose tip a path filter hides clears the filter without asking, selects the tip alone in Commits, and makes the side area half",
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
		// old is a branch at m1, which master reaches. m2 is the only commit that
		// touches m2.txt, so a path filter on it hides m1 and base.
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
		t.ExpectPopup().Prompt().Title(Equals("Enter path:")).Type("m2.txt").Confirm()
		commits.IsFocused().Lines(Contains("m2"))
		t.ExpectScreenMode(types.SCREEN_NORMAL, types.SCREEN_AREA_SIDE)

		t.Views().Branches().Focus().NavigateToLine(Contains("old"))
		t.Views().Branches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"v"})

		// The path filter is gone, so the whole log is back, and m1 is the only
		// selected commit.
		commits.IsFocused().
			Lines(Contains("m2"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	},
})
