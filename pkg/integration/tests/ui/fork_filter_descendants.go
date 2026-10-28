package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterDescendants = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w d shows the selected commits with their descendants, for one commit and for a Shift range",
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
		// Fixed dates keep the default log order the same in every run. Newest
		// first, the unfiltered list is: merge, o1, f2, f1, m2, m1, base.
		shell.EmptyCommitWithDate("base", "2001-01-01 12:00:00 +0000").
			EmptyCommitWithDate("m1", "2001-01-02 12:00:00 +0000").
			EmptyCommitWithDate("m2", "2001-01-03 12:00:00 +0000").
			NewBranchFrom("feature", "HEAD~1").
			EmptyCommitWithDate("f1", "2001-01-04 12:00:00 +0000").
			EmptyCommitWithDate("f2", "2001-01-05 12:00:00 +0000").
			Checkout("master").
			NewBranchFrom("other", "HEAD~2").
			EmptyCommitWithDate("o1", "2001-01-06 12:00:00 +0000").
			Checkout("master").
			Merge("feature")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()

		// m1's descendants are the merge, m2, f2 and f1, and m1 itself. base and o1 are hidden.
		commits.NavigateToLine(Contains("m1"))
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"d"})

		commits.LineCount(EqualsInt(5))
		commits.ContainsLines(Contains("Merge branch"))
		commits.ContainsLines(Contains("m2"))
		commits.ContainsLines(Contains("f2"))
		commits.ContainsLines(Contains("f1"))
		commits.ContainsLines(Contains("m1"))
		commits.SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("descendants of 1 selected commit"))

		// Escape clears the filter. Then f1 and m2, which are on neighbouring lines,
		// are selected as a Shift range; w d shows the union of their descendants.
		commits.Press(config.Keybinding{"<esc>"})
		commits.LineCount(EqualsInt(7))
		commits.NavigateToLine(Contains("f1")).
			Press(config.Keybinding{"<shift+down>"}).
			SelectedLines(Contains("f1"), Contains("m2"))
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"d"})

		// f1 and m2 with their descendants: the merge, f2, f1 and m2. m1, base and o1 are hidden.
		// The cursor stays on m2, the commit it was on; the Shift range collapses to it.
		commits.Lines(Contains("Merge branch"), Contains("f2"), Contains("f1"), Contains("m2")).
			SelectedLines(Contains("m2"))
		t.Views().Information().Content(Contains("descendants of 2 selected commits"))
	},
})
