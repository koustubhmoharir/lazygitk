package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ForkShortKeysConflicts = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default short keys in the merge conflicts view: ] and [ move between conflicts, and Space picks the selected hunk",
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
		shared.CreateMergeConflictFileMultiple(shell)
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Lines(
				Contains("UU file").IsSelected(),
			)
		t.Views().Files().Press(config.Keybinding{"v"})

		conflicts := t.Views().MergeConflicts()
		conflicts.
			IsFocused().
			SelectedLines(
				Contains("<<<<<<< HEAD"),
				Contains("First Change"),
				Contains("======="),
			)

		// ] moves to the second conflict; [ moves back to the first.
		conflicts.Press(config.Keybinding{"]"})
		conflicts.
			IsFocused().
			SelectedLines(
				Contains("<<<<<<< HEAD"),
				Contains("Other First Change"),
				Contains("======="),
			)
		conflicts.Press(config.Keybinding{"["})
		conflicts.
			IsFocused().
			SelectedLines(
				Contains("<<<<<<< HEAD"),
				Contains("First Change"),
				Contains("======="),
			)

		// Space picks the first hunk of the selected (first) conflict, the same
		// hunk that g h p picks in ForkDefaultConflicts.
		conflicts.Press(config.Keybinding{"<space>"})
		conflicts.
			IsFocused().
			Content(DoesNotContain("<<<<<<< HEAD\nFirst Change")).
			Content(Contains("<<<<<<< HEAD\nOther First Change"))
	},
})
