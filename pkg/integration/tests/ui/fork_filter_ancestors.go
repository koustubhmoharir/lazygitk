package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterAncestors = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w a shows the selected commit with its ancestors only, and Escape restores the full list with the cursor kept",
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
		shell.EmptyCommitWithDate("base", "2001-01-01 12:00:00 +0000").
			EmptyCommitWithDate("m1", "2001-01-02 12:00:00 +0000").
			EmptyCommitWithDate("m2", "2001-01-03 12:00:00 +0000").
			NewBranchFrom("feature", "HEAD~1").
			EmptyCommitWithDate("f1", "2001-01-04 12:00:00 +0000").
			EmptyCommitWithDate("f2", "2001-01-05 12:00:00 +0000").
			Checkout("master").
			NewBranchFrom("other", "HEAD~2").
			EmptyCommitWithDate("o1", "2001-01-06 12:00:00 +0000").
			Checkout("master")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("f1"))

		// w opens the filter menu; a picks the ancestors of the selected commit.
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"a"})

		// f1 has the ancestors m1 and base. f2 (its descendant), m2 and o1 are hidden.
		commits.Lines(Contains("f1"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("f1"))
		t.Views().Information().Content(Contains("ancestors of 1 selected commit"))

		// Escape clears the filter: every commit is back, and the cursor stays on f1.
		commits.Press(config.Keybinding{"<esc>"})
		commits.LineCount(EqualsInt(6))
		commits.ContainsLines(Contains("f2"))
		commits.ContainsLines(Contains("m2"))
		commits.ContainsLines(Contains("o1"))
		commits.SelectedLines(Contains("f1"))
	},
})
