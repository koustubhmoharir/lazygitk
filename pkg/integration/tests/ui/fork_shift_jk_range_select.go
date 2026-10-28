package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShiftJkRangeSelect = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Shift+J and Shift+K extend the Commits selection like Shift+Down and Shift+Up, and plain j ends the range",
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
		shell.CreateFileAndAdd("base.txt", "base\n").Commit("base").
			CreateFileAndAdd("f1.txt", "1\n").Commit("commit 1").
			CreateFileAndAdd("f2.txt", "2\n").Commit("commit 2").
			CreateFileAndAdd("f3.txt", "3\n").Commit("commit 3").
			CreateFileAndAdd("f4.txt", "4\n").Commit("commit 4")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()

		// Shift+J twice extends the range down from the top commit to three commits.
		commits.Press(config.Keybinding{"J"}).
			Press(config.Keybinding{"J"}).
			SelectedLines(Contains("commit 4"), Contains("commit 3"), Contains("commit 2"))

		// Shift+K retracts the range by one commit.
		commits.Press(config.Keybinding{"K"}).
			SelectedLines(Contains("commit 4"), Contains("commit 3"))

		// A plain j moves the cursor and ends the range, leaving one selected line.
		commits.Press(config.Keybinding{"j"}).
			SelectedLines(Contains("commit 2"))
	},
})
