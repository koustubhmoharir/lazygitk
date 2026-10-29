package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterKeepsScreenMode = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Filtering the commits by ancestors and clearing the filter keeps the side area enlarged",
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
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		commits.Press(keys.Universal.NextScreenMode)
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)

		commits.NavigateToLine(Contains("f1"))

		// w opens the filter menu; a picks the ancestors of the selected commit.
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"a"})
		commits.Lines(Contains("f1"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("f1"))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)

		// Escape clears the filter, and the mode stays as it was.
		commits.Press(config.Keybinding{"<esc>"})
		commits.LineCount(EqualsInt(6))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	},
})
