package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterAuthorKey = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w u in Commits opens the author prompt prefilled with the selected commit's author, and shows only that author's commits",
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
		// Fixed dates keep the default log order, newest first, the same in every run.
		shell.SetAuthor("Alice Author", "alice@example.com").
			EmptyCommitWithDate("alice 1", "2001-01-01 12:00:00 +0000").
			SetAuthor("Bob Builder", "bob@example.com").
			EmptyCommitWithDate("bob 1", "2001-01-02 12:00:00 +0000").
			SetAuthor("Alice Author", "alice@example.com").
			EmptyCommitWithDate("alice 2", "2001-01-03 12:00:00 +0000").
			SetAuthor("Bob Builder", "bob@example.com").
			EmptyCommitWithDate("bob 2", "2001-01-04 12:00:00 +0000")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("alice 2"))

		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"u"})

		t.ExpectPopup().Prompt().
			Title(Equals("Enter author:")).
			InitialText(Equals("Alice Author <alice@example.com>")).
			Confirm()

		// Only Alice's commits are left.
		t.Views().Commits().IsFocused().
			Lines(Contains("alice 2"), Contains("alice 1")).
			SelectedLines(Contains("alice 2"))
	},
})
