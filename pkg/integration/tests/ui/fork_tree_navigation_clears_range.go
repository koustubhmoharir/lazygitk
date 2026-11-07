package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkTreeNavigationClearsRange = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A tree navigation that moves the cursor clears a range selection in the Files tree, and a navigation that does nothing leaves the range as it is",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the Left and Right routes come from the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateDir("dir")
		shell.CreateFileAndAdd("dir/file1", "1st line\n")
		shell.CreateFileAndAdd("dir/file2", "a\n")
		shell.Commit("first")
		// Unstaged changes, so both files appear in the Files panel.
		shell.UpdateFile("dir/file1", "1st line\n2nd line\n")
		shell.UpdateFile("dir/file2", "a\nb\n")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		files := t.Views().Files().
			IsFocused().
			Lines(
				Contains("dir").IsSelected(),
				Contains("file1"),
				Contains("file2"),
			).
			NavigateToLine(Contains("file1"))

		// Shift+J extends the selection to file2, so the range is file1 and file2.
		files.Press(config.Keybinding{"J"}).
			SelectedLines(
				Contains("file1"),
				Contains("file2"),
			)

		// Left on a file goes to its parent directory. That moves the cursor, so the
		// range is cleared and only the directory row is selected.
		files.Press(config.Keybinding{"<left>"}).
			SelectedLines(Contains("dir")).
			SelectedLineIdx(0)

		// Make a new range from file1. Right on a file has no child to go to, so the
		// navigation does nothing and the range stays.
		files.Press(config.Keybinding{"j"}).
			SelectedLines(Contains("file1")).
			Press(config.Keybinding{"J"}).
			SelectedLines(
				Contains("file1"),
				Contains("file2"),
			).
			Press(config.Keybinding{"<right>"}).
			SelectedLines(
				Contains("file1"),
				Contains("file2"),
			)
	},
})
