package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCommitsMultiSelect = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Commits multi-select: Space builds an explicit set, Shift runs join it, Space toggles runs, w a keeps the set, Escape clears the filter then the set then the marks, and Space in Sub-commits still inspects",
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
		// Newest first in the list: foxtrot, echo, delta, charlie, bravo, alpha, base.
		shell.CreateFileAndAdd("base.txt", "base\n").Commit("base").
			CreateFileAndAdd("a.txt", "a\n").Commit("alpha").
			CreateFileAndAdd("b.txt", "b\n").Commit("bravo").
			CreateFileAndAdd("c.txt", "c\n").Commit("charlie").
			CreateFileAndAdd("d.txt", "d\n").Commit("delta").
			CreateFileAndAdd("e.txt", "e\n").Commit("echo").
			CreateFileAndAdd("f.txt", "f\n").Commit("foxtrot").
			// A branch with history of its own, to open Sub-commits from Branches.
			NewBranchFrom("side", "HEAD~2").
			Checkout("master")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		space := config.Keybinding{"<space>"}
		down := config.Keybinding{"j"}
		up := config.Keybinding{"k"}
		shiftDown := config.Keybinding{"J"}

		// Space on foxtrot puts it in the set; the cursor stays on it.
		commits.NavigateToLine(Contains("foxtrot")).
			Press(space)
		commits.ExplicitSelection(Contains("foxtrot"))
		commits.MarkedLines(Contains("foxtrot"))
		// Space in Commits no longer inspects: the commit files stay unopened.
		commits.IsFocused()

		// Two plain moves move the cursor to delta, leaving the set as it was.
		commits.Press(down).Press(down)
		commits.ExplicitSelection(Contains("foxtrot"))
		commits.MarkedLines(Contains("delta"))

		// Space on delta adds it, and the set is listed in list order.
		commits.Press(space)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("delta"))

		// A plain move onto charlie does not select it.
		commits.Press(down)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("delta"))
		commits.MarkedLines(Contains("charlie"))

		// Shift+J twice from charlie runs down to alpha; a plain j ends the run and
		// keeps charlie, bravo and alpha in the set, with the cursor on base.
		commits.Press(shiftDown).Press(shiftDown)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("delta"), Contains("charlie"), Contains("bravo"), Contains("alpha"))
		commits.Press(down)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("delta"), Contains("charlie"), Contains("bravo"), Contains("alpha"))
		commits.MarkedLines(Contains("base"))

		// Space on a selected commit removes it.
		commits.Press(up).Press(space)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("delta"), Contains("charlie"), Contains("bravo"))

		// Space on a run of selected commits removes the whole run and ends it.
		commits.Press(up).Press(up).
			Press(shiftDown)
		commits.Press(space)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("delta"))

		// A refresh keeps the set.
		commits.Press(config.Keybinding{"<f5>"})
		commits.ExplicitSelection(Contains("foxtrot"), Contains("delta"))

		// w a shows the set and its ancestors, and the set stays selected. Removing
		// foxtrot and adding echo leaves the set echo and delta, so foxtrot is hidden.
		commits.NavigateToLine(Contains("foxtrot")).Press(space)
		commits.NavigateToLine(Contains("echo")).Press(space)
		commits.ExplicitSelection(Contains("echo"), Contains("delta"))

		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"a"})
		commits.Lines(Contains("echo"), Contains("delta"), Contains("charlie"), Contains("bravo"), Contains("alpha"), Contains("base"))
		commits.ExplicitSelection(Contains("echo"), Contains("delta"))
		t.Views().Information().Content(Contains("ancestors of 2 selected commits"))

		// Escape clears the filter first: every commit is back and the set is kept.
		// Cherry-pick marks made now are cleared by the third Escape.
		commits.Press(config.Keybinding{"c"})
		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"y"})

		commits.Press(config.Keybinding{"<esc>"})
		commits.LineCount(EqualsInt(7))
		commits.ExplicitSelection(Contains("echo"), Contains("delta"))
		// The filter label is gone, so the Information view reports the marks.
		t.Views().Information().Content(MatchesRegexp(`(^|[^0-9])2 commits? copied`))

		// The second Escape clears the set: the selection is the cursor commit alone.
		commits.Press(config.Keybinding{"<esc>"})
		commits.ExplicitSelection()
		commits.NoMarkedLines()
		commits.SelectedLines(Contains("echo"))
		t.Views().Information().Content(MatchesRegexp(`(^|[^0-9])2 commits? copied`))

		// The third Escape clears the cherry-pick marks.
		commits.Press(config.Keybinding{"<esc>"})
		t.Views().Information().Content(DoesNotContain("copied"))

		// v still inspects in Commits: the commit's files open, and Escape returns.
		commits.Press(config.Keybinding{"v"})
		t.Views().CommitFiles().IsFocused()
		t.Views().CommitFiles().Press(config.Keybinding{"<esc>"})
		commits.IsFocused()

		// Sub-commits keeps Space as inspect: Space opens the selected commit's files.
		t.Views().Branches().Focus().NavigateToLine(Contains("side")).
			Press(config.Keybinding{"v"})
		t.Views().SubCommits().IsFocused().
			Press(space)
		t.Views().CommitFiles().IsFocused()
		t.Views().CommitFiles().Press(config.Keybinding{"<esc>"})

		// Reflog has no inspect route in upstream either, so Space does nothing there.
		t.Views().ReflogCommits().Focus().
			Press(space)
		t.Views().ReflogCommits().IsFocused()
	},
})
