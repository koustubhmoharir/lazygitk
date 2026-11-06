package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPrefixDisabledRows = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A prefix row whose leaf has a failed precondition is crossed out with its reason: s p with no previous checkout reports why and runs nothing",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, so the real default prefix menus are tested.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		// No checkout has happened yet, so the reflog names no previous branch.
		shell.CreateFileAndAdd("file", "one\n").Commit("first")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		head := t.Git().GetCommitHash("HEAD")

		t.Views().Files().Focus().Press(config.Keybinding{"s"})
		menu := t.ExpectPopup().Menu().Title(Equals("s"))

		// The p row is selectable and explains the failed precondition.
		menu.Select(Contains("Checkout previous branch")).
			Tooltip(Contains("Disabled: No branch was checked out before the current one.")).
			Confirm().
			Tap(func() {
				t.ExpectToast(Equals("Disabled: No branch was checked out before the current one."))
			})

		// Pressing p reports the same reason, and the menu stays open.
		t.Views().Menu().Press(config.Keybinding{"p"})
		t.ExpectToast(Equals("Disabled: No branch was checked out before the current one."))
		t.ExpectPopup().Menu().Title(Equals("s"))

		menu.Cancel()
		t.Views().Files().IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("A disabled s p moved HEAD to " + got)
		}
		t.Git().CurrentBranchName("master")
	},
})
