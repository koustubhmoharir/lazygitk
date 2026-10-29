package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterAncestorsOrder = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w a on commits of two diverged branches lists them in the order of the unfiltered log",
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
		// The unfiltered list walks every branch, so the other branches are in it.
		loaded.Git.Log.ShowWholeGraph = true
	},
	SetupRepo: func(shell *Shell) {
		// Fixed dates keep the log order the same in every run. The dates interleave
		// the branches: topo-order lists the feature commit f1 after the master commit
		// m1 in the unfiltered log, and a walk from the selected tips alone would list
		// f1 next to its own line instead.
		shell.EmptyCommitWithDate("base", "2001-01-01 12:00:00 +0000").
			NewBranchFrom("feature", "master").
			EmptyCommitWithDate("f1", "2001-01-02 12:00:00 +0000").
			EmptyCommitWithDate("f2", "2001-01-03 12:00:00 +0000").
			NewBranchFrom("side", "HEAD~1").
			EmptyCommitWithDate("s1", "2001-01-05 12:00:00 +0000").
			Checkout("master").
			EmptyCommitWithDate("m1", "2001-01-04 12:00:00 +0000")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		space := config.Keybinding{"<space>"}

		// The unfiltered log lists the branches interleaved: m1 sits between s1 and the
		// feature commits, and f2, which is outside the scope, sits above f1.
		commits.Lines(Contains("s1"), Contains("m1"), Contains("f2"), Contains("f1"), Contains("base"))

		// Space on s1 and on m1 selects one commit on each branch.
		commits.NavigateToLine(Contains("s1")).
			Press(space)
		commits.NavigateToLine(Contains("m1")).
			Press(space)

		// w opens the filter menu; a picks the ancestors of the selected commits.
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"a"})

		// The ancestors are listed in the unfiltered order: s1 and m1 are not grouped by
		// branch, and f2 is hidden because it is not an ancestor of either.
		commits.Lines(Contains("s1"), Contains("m1"), Contains("f1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("ancestors of 2 selected commits"))
	},
})
