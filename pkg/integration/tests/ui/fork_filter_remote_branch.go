package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterRemoteBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w b shows a remote branch with its history, alone, from Commits and from Remote branches, and Escape restores the full list",
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
		// master has base, m1 and m2.
		shell.EmptyCommitWithDate("base", "2001-01-01 12:00:00 +0000").
			EmptyCommitWithDate("m1", "2001-01-02 12:00:00 +0000").
			EmptyCommitWithDate("m2", "2001-01-03 12:00:00 +0000").
			CloneIntoRemote("origin").
			// origin/feat is a remote-tracking ref at m1 alone: no local branch points
			// at m1, so only the remote branch decorates it.
			RunCommand([]string{"git", "update-ref", "refs/remotes/origin/feat", "HEAD~1"})
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// Case 1: from Commits. m1 is decorated by origin/feat only, so w b applies it
		// directly. Only its history is shown: m1 and base, not m2.
		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("m1"))
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		commits.IsFocused().
			Lines(Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("Showing origin/feat and its history"))

		// Escape clears the scope: the full list is back, and the cursor is on m1.
		commits.Press(config.Keybinding{"<esc>"})
		commits.Lines(Contains("m2"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))

		// Case 2: from Remote branches. The remote branch origin/feat is selected, so
		// w b shows it with its history, and Commits opens on its tip.
		// v lists the remote's branches. In the real defaults Enter does not open them.
		t.Views().Remotes().Focus().
			Lines(Contains("origin").IsSelected()).
			Press(config.Keybinding{"v"})

		t.Views().RemoteBranches().
			IsFocused().
			NavigateToLine(Contains("feat"))
		t.Views().RemoteBranches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		commits.IsFocused().
			Lines(Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("Showing origin/feat and its history"))
		commits.Press(config.Keybinding{"<esc>"})
	},
})
