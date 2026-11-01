package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShowInCommitsUnreachableRefused = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w v on a reflog entry that no branch, tag or HEAD reaches is refused with a message, and nothing changes",
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
		loaded.Git.Log.ShowWholeGraph = true
	},
	SetupRepo: func(shell *Shell) {
		// The reset moves master back to m1. m2 is then named by the reflog alone.
		shell.CreateFileAndAdd("base.txt", "base").
			Commit("base").
			CreateFileAndAdd("m1.txt", "m1").
			Commit("m1").
			CreateFileAndAdd("m2.txt", "m2").
			Commit("m2").
			HardReset("HEAD~1")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().ReflogCommits().Focus().NavigateToLine(Contains("m2"))
		t.Views().ReflogCommits().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"v"})

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("is not in the Commits log: no branch, tag or HEAD reaches it.")).
			Confirm()

		t.Views().ReflogCommits().IsFocused()
		t.ExpectScreenMode(types.SCREEN_NORMAL, types.SCREEN_AREA_SIDE)
	},
})
