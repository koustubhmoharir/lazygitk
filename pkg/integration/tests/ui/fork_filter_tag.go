package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterTag = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w b shows an annotated tag with its history, alone, from Commits and from Tags, and Escape restores the full list",
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
			// v1.0 is an annotated tag at m1, which no branch points at.
			CreateAnnotatedTag("v1.0", "release", "HEAD~1")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Case 1: from Commits. The tag applies directly, and only its history is shown:
		// m1 and base, not m2.
		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("m1"))
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		commits.IsFocused().
			Lines(Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("Showing tag v1.0 and its history"))

		// Escape clears the scope: the full list is back, and the cursor is on m1.
		commits.Press(config.Keybinding{"<esc>"})
		commits.Lines(Contains("m2"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))

		// Case 2: from Tags. The tag is selected, so w b shows it with its history, and
		// Commits opens on the commit the annotated tag peels to.
		t.Views().Tags().Focus().NavigateToLine(Contains("v1.0"))
		t.Views().Tags().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		commits.IsFocused().
			Lines(Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("Showing tag v1.0 and its history"))
		commits.Press(config.Keybinding{"<esc>"})
	},
})
