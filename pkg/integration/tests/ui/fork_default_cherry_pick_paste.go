package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultCherryPickPaste = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default c p on a Commits selection of another branch: a cancelled review keeps the selection, a pick clears it, and Files c p is then a no-op",
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
		shell.CreateFileAndAdd("base.txt", "base\n").Commit("base").
			NewBranch("src").
			CreateFileAndAdd("p1.txt", "p1\n").Commit("p1").
			CreateFileAndAdd("p2.txt", "p2\n").Commit("p2").
			CreateFileAndAdd("p3.txt", "p3\n").Commit("p3").
			Checkout("master")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		masterHead := t.Git().GetCommitHash("master")
		space := config.Keybinding{"<space>"}

		// Opens the real default c prefix menu, then picks the continuation.
		cherry := func(view *ViewDriver, second string) {
			view.Press(config.Keybinding{"c"})
			t.ExpectPopup().Menu().Title(Equals("c"))
			t.Views().Menu().Press(config.Keybinding{second})
		}

		// src's commits are in Commits with the whole graph. Space adds p1 and p3
		// to the selection; p2 between them is not selected.
		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("p1")).Press(space)
		commits.NavigateToLine(Contains("p3")).Press(space)
		commits.ExplicitSelection(Contains("p3"), Contains("p1"))

		// Review must list p1 before p3 and name master.
		reviewContent := MatchesRegexp(`(?s)Cherry-pick these commits in this order.*1\. \S+ p1.*2\. \S+ p3`)

		// Cancelling the review moves nothing and keeps the selection.
		cherry(commits, "p")
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(reviewContent.Contains("master")).
			Cancel()
		if got := t.Git().GetCommitHash("HEAD"); got != masterHead {
			t.Fail("Cancelling cherry-pick review moved master to " + got)
		}
		commits.ExplicitSelection(Contains("p3"), Contains("p1"))

		// Confirming picks both, and the successful pick clears the selection.
		cherry(commits, "p")
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(reviewContent.Contains("master")).
			Confirm()
		commits.TopLines(Contains("p3"))
		t.Shell().RunCommand([]string{"git", "log", "--format=%s", "--output=actual-log"})
		t.FileSystem().FileContent("actual-log", Equals("p3\np1\nbase\n"))
		commits.ExplicitSelection()

		// With no selection, c in Files opens no menu and changes nothing.
		pastedHead := t.Git().GetCommitHash("HEAD")
		t.Views().Files().Focus().Press(config.Keybinding{"c"})
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()
		t.Views().Files().IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != pastedHead {
			t.Fail("Files c p after the paste moved master to " + got)
		}
	},
})
