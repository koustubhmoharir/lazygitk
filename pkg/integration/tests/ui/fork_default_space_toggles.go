package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultSpaceToggles = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Space: in Commit files it toggles the file in the custom patch and Enter shows its diff; in a regular menu Space runs nothing and Enter runs the item",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so that Space comes from the fork catalogue.
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
		shell.CreateFileAndAdd("file1", "file1 content\n")
		shell.CreateFileAndAdd("file2", "file2 content\n")
		shell.Commit("first commit")
		// An unstaged change for the stash menu to take.
		shell.UpdateFile("file1", "changed\n")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// a. In Commit files, Space adds the selected file to the custom patch, and
		// Space again removes it.
		t.Views().Commits().
			Focus().
			Lines(
				Contains("first commit").IsSelected(),
			).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			NavigateToLine(Contains("file1")).
			Press(config.Keybinding{"<space>"})

		t.Views().CommitFiles().
			ContainsLines(
				Contains("● file1").IsSelected(),
				Contains("A file2"),
			)
		t.Views().Information().Content(Contains("Building patch"))

		t.Views().CommitFiles().Press(config.Keybinding{"<space>"})

		t.Views().CommitFiles().
			ContainsLines(
				Contains("A file1").IsSelected(),
				Contains("A file2"),
			)
		t.Views().Information().Content(DoesNotContain("Building patch"))

		// b. Enter shows the file's diff with the main view focused. Escape returns to
		// Commit files.
		t.Views().CommitFiles().
			Press(config.Keybinding{"<enter>"})
		t.Views().Main().
			IsFocused().
			Content(Contains("+file1 content"))
		t.Views().Main().PressEscape()
		t.Views().CommitFiles().IsFocused()
		t.Views().CommitFiles().PressEscape()
		t.Views().Commits().IsFocused()

		// c. In a regular menu, Space runs nothing and leaves the menu open. Enter runs
		// the selected item. g s o opens the stash options menu.
		t.Views().Files().
			Focus().
			Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("g s"))
		t.Views().Menu().Press(config.Keybinding{"o"})
		t.ExpectPopup().Menu().Title(Equals("Stash options"))

		t.Views().Menu().Press(config.Keybinding{"<space>"})
		t.ExpectPopup().Menu().Title(Equals("Stash options"))
		t.Views().Stash().IsEmpty()
		t.Views().Files().
			Lines(
				Contains("file1"),
			)

		t.ExpectPopup().Menu().
			Title(Equals("Stash options")).
			Select(MatchesRegexp("Stash all changes$")).
			Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Stash changes")).Type("space stash").Confirm()

		t.Views().Stash().
			Lines(
				Contains("space stash"),
			)
		t.Views().Files().
			IsEmpty()
	},
})
