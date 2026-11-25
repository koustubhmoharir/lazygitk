package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkSubCommitsReflogMultiSelect = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Sub-commits and Reflog selection: Space builds a set in Sub-commits, c p picks it, a pane switch keeps it, Escape clears it and then leaves; Reflog counts a commit on two rows once, and a one-commit command is disabled while the set exists",
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
		// The reset adds a reflog entry for foxtrot, so foxtrot is on two reflog rows.
		shell.CreateFileAndAdd("base.txt", "base\n").Commit("base").
			CreateFileAndAdd("a.txt", "a\n").Commit("alpha").
			CreateFileAndAdd("b.txt", "b\n").Commit("bravo").
			CreateFileAndAdd("c.txt", "c\n").Commit("charlie").
			CreateFileAndAdd("d.txt", "d\n").Commit("delta").
			CreateFileAndAdd("e.txt", "e\n").Commit("echo").
			CreateFileAndAdd("f.txt", "f\n").Commit("foxtrot").
			HardReset("HEAD").
			// Branches with history of their own, to open Sub-commits from Branches.
			NewBranchFrom("side", "HEAD~2").
			NewBranchFrom("other", "HEAD~4").
			Checkout("master")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		space := config.Keybinding{"<space>"}
		down := config.Keybinding{"j"}
		esc := config.Keybinding{"<esc>"}
		disabled := Equals("Disabled: Clear the selection first (Escape)")

		// Sub-commits lists side's history, newest first: delta, charlie, bravo, alpha, base.
		t.Views().Branches().Focus().NavigateToLine(Contains("side")).
			PressEnter()
		subCommits := t.Views().SubCommits().IsFocused()

		// Space on delta, the top row, selects it and leaves the cursor there.
		subCommits.Press(space)
		subCommits.ExplicitSelection(Contains("delta"))
		subCommits.Subtitle(Equals("1 selected"))

		// Space on bravo, two rows down, adds it: the set is not a run of adjacent commits.
		subCommits.Press(down).Press(down).Press(space)
		subCommits.ExplicitSelection(Contains("delta"), Contains("bravo"))
		subCommits.Subtitle(Equals("2 selected"))
		subCommits.MarkedLines(Contains("bravo"))

		// c p picks the set, not the cursor commit: the review lists delta and bravo only.
		t.Common().CherryPickSelectionWithDefaultKeys(subCommits)
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(Contains("delta").Contains("bravo").DoesNotContain("charlie")).
			Cancel()
		subCommits.ExplicitSelection(Contains("delta"), Contains("bravo"))

		// g c combines the set, as in Commits: its review lists delta and bravo, and Escape
		// cancels it, keeping the set.
		subCommits.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"c"})
		t.ExpectPopup().Confirmation().
			Title(Equals("Review commit combination")).
			Content(Contains("delta").Contains("bravo").DoesNotContain("charlie")).
			Cancel()
		subCommits.ExplicitSelection(Contains("delta"), Contains("bravo"))

		// A pane switch keeps the set: Sub-commits is still the view of the branches window.
		t.Views().Status().Focus().Press(config.Keybinding{"3"})
		subCommits.IsFocused()
		subCommits.ExplicitSelection(Contains("delta"), Contains("bravo"))
		subCommits.Subtitle(Equals("2 selected"))

		// Opening Sub-commits for another ref, from Branches in the same window, starts with
		// no selection: the set belonged to side.
		t.Views().Branches().Focus().NavigateToLine(Contains("other")).PressEnter()
		t.Views().SubCommits().IsFocused().ExplicitSelection()
		t.Views().SubCommits().Subtitle(Equals(""))

		// On other, Space selects the top commit; the first Escape clears the set and stays in
		// Sub-commits, and the second leaves for Branches.
		t.Views().SubCommits().Press(space)
		t.Views().SubCommits().Subtitle(Equals("1 selected"))
		t.Views().SubCommits().Press(esc)
		t.Views().SubCommits().ExplicitSelection()
		t.Views().SubCommits().IsFocused()
		t.Views().SubCommits().Press(esc)
		t.Views().Branches().IsFocused()

		// Reopening side shows no selection: the set went with the view.
		t.Views().Branches().NavigateToLine(Contains("side")).PressEnter()
		t.Views().SubCommits().IsFocused().ExplicitSelection()
		t.Views().SubCommits().Press(esc)
		t.Views().Branches().IsFocused()

		// Foxtrot is on three reflog rows: the checkout back to master, the reset to HEAD
		// and the commit. Space on one row selects the commit, so all three rows are
		// highlighted, and the subtitle counts it once.
		reflog := t.Views().ReflogCommits().Focus()
		reflog.NavigateToLine(Contains("commit: foxtrot")).Press(space)
		reflog.ExplicitSelection(Contains("checkout: moving from other to master"), Contains("reset: moving to HEAD"), Contains("commit: foxtrot"))
		reflog.Subtitle(Equals("1 selected"))

		// A command on one commit is disabled while the set exists, and the set stays.
		reflog.Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"d"})
		t.ExpectToast(disabled)
		t.ExpectPopup().Menu().Title(Equals("s")).Cancel()
		reflog.ExplicitSelection(Contains("checkout: moving from other to master"), Contains("reset: moving to HEAD"), Contains("commit: foxtrot"))

		// Escape clears the set, and the reflog stays focused.
		reflog.Press(esc)
		reflog.ExplicitSelection()
		reflog.IsFocused()
	},
})
