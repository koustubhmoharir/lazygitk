package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterGraph = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w a draws the graph of the filtered commits, lane by lane, and Escape draws the full graph again",
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
		loaded.Git.Log.ShowGraph = "always"
		// The unfiltered list walks every branch, so the other branches are in it.
		loaded.Git.Log.ShowWholeGraph = true
	},
	SetupRepo: func(shell *Shell) {
		// Same branch layout as fork_filter_ancestors_order, so the unfiltered graph
		// interleaves the branches and the filtered one has two lanes.
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

		commits.Lines(
			Contains("○ s1"),
			Contains("│ ○ m1"),
			Contains("│ │ ○ f2"),
			Contains("○ │─╯ f1"),
			Contains("○─╯ base"),
		)

		commits.NavigateToLine(Contains("s1")).
			Press(space)
		commits.NavigateToLine(Contains("m1")).
			Press(space)

		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"a"})

		// The filtered list keeps the unfiltered order, and f2 and its lane are gone.
		// m1 keeps its lane down to base, and f1 now sits in the first lane.
		commits.Lines(
			Contains("○ s1"),
			Contains("│ ○ m1"),
			Contains("○ │ f1"),
			Contains("○─╯ base"),
		).
			SelectedLines(Contains("m1"))

		// Escape clears the filter: the full graph is drawn again with the cursor kept.
		// The selected m1 draws its lane over f1's, so f1's line reads differently
		// from the unselected graph above.
		commits.Press(config.Keybinding{"<esc>"})
		commits.Lines(
			Contains("○ s1"),
			Contains("│ ○ m1"),
			Contains("│ │ ○ f2"),
			Contains("○─│ ╯ f1"),
			Contains("○─╯ base"),
		).
			SelectedLines(Contains("m1"))
	},
})
