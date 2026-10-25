package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultCherryPick = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default c y, c x, c p cherry-pick marks survive subview exit, paste from marked source, and Files no-op",
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

		// 0. No marks and no selection: Files c p must leave no popup and no change.
		// The c menu is opened from Files; with nothing to paste, p must be
		// inapplicable and close without a review.
		t.Views().Files().Focus().Press(config.Keybinding{"c"})
		t.ExpectPopup().Menu().Title(Equals("c")).ContainsLines(Contains("Cherry-pick"), Contains("Cancel"))
		t.Views().Menu().Press(config.Keybinding{"p"})
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()
		t.Views().Files().IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != masterHead {
			t.Fail("No-marks Files c p moved master to " + got)
		}

		// Opens the real default c prefix menu, then picks the continuation.
		cherry := func(view *ViewDriver, second string) {
			view.Press(config.Keybinding{"c"})
			t.ExpectPopup().Menu().Title(Equals("c"))
			t.Views().Menu().Press(config.Keybinding{second})
		}
		// The mark state is reported by the Information view as the total number
		// of marked commits; the marked hash colour is not visible in buffer text.
		assertMarkCount := func(count string) {
			t.Views().Information().Content(MatchesRegexp(`(^|[^0-9])` + count + ` commits? copied`))
		}
		// Opens src's commits from the Branches pane with the default v key.
		openSrcSubCommits := func() *ViewDriver {
			t.Views().Branches().Focus().NavigateToLine(Contains("src")).Press(config.Keybinding{"v"})
			return t.Views().SubCommits().IsFocused()
		}

		// 1. Mark p1, then range-select p3 and p2 downwards from p3 and mark both.
		subCommits := openSrcSubCommits()
		subCommits.NavigateToLine(Contains("p1"))
		cherry(subCommits, "y")
		assertMarkCount("1")

		subCommits.NavigateToLine(Contains("p3")).Press(config.Keybinding{"<shift+down>"})
		subCommits.TopLines(Contains("p3").IsSelected(), Contains("p2").IsSelected(), Contains("p1"))
		cherry(subCommits, "y")
		assertMarkCount("3")

		// A range stays active until dismissed; with a range active, Escape only
		// dismisses it and SubCommits keeps focus.
		subCommits.Press(config.Keybinding{"<esc>"}).IsFocused()

		// Marking p2 alone again keeps it marked; unmarking p2 leaves the rest.
		subCommits.NavigateToLine(Contains("p2"))
		cherry(subCommits, "y")
		assertMarkCount("3")
		cherry(subCommits, "x")
		assertMarkCount("2")

		// 2. With no range active, Escape exits SubCommits and the marks survive.
		subCommits.IsFocused().Press(config.Keybinding{"<esc>"})
		t.Views().Branches().IsFocused()
		assertMarkCount("2")

		// 3. Escape in Commits clears the marks.
		commits := t.Views().Commits()
		commits.Focus().Press(config.Keybinding{"<esc>"}).IsFocused()
		t.Views().Information().Content(DoesNotContain("copied"))

		// 4. Reopen src's SubCommits and re-mark p1 and p3.
		subCommits = openSrcSubCommits()
		subCommits.NavigateToLine(Contains("p1"))
		cherry(subCommits, "y")
		assertMarkCount("1")
		subCommits.NavigateToLine(Contains("p3"))
		cherry(subCommits, "y")
		assertMarkCount("2")
		subCommits.IsFocused().Press(config.Keybinding{"<esc>"})

		// 5. c p in Commits with marks offers the marked and selected sources.
		commits.Focus()
		cherry(commits, "p")
		t.ExpectPopup().Menu().
			Title(Equals("Choose commits to cherry-pick")).
			ContainsLines(
				Contains("Previously marked commits (2)"),
				Contains("Selected commits (1)"),
			).
			Select(Contains("Previously marked commits (2)")).
			Confirm()
		// Review must list p1 before p3 and name master.
		reviewContent := MatchesRegexp(`(?s)Cherry-pick these commits in this order.*1\. \S+ p1.*2\. \S+ p3`)
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(reviewContent.Contains("master")).
			Cancel()
		if got := t.Git().GetCommitHash("HEAD"); got != masterHead {
			t.Fail("Cancelling cherry-pick review moved master to " + got)
		}
		assertMarkCount("2")

		cherry(commits, "p")
		t.ExpectPopup().Menu().
			Title(Equals("Choose commits to cherry-pick")).
			Select(Contains("Previously marked commits (2)")).
			Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(reviewContent.Contains("master")).
			Confirm()
		commits.TopLines(Contains("p3"))
		t.Shell().RunCommand([]string{"git", "log", "--format=%s", "--output=actual-log"})
		t.FileSystem().FileContent("actual-log", Equals("p3\np1\nbase\n"))
		// Observed after paste: the copied-commit indicator is gone (as cherry_pick.go expects).
		t.Views().Information().Content(DoesNotContain("commits copied"))
		pastedHead := t.Git().GetCommitHash("HEAD")

		// Retained pasted commits are intended: Files c p offers the re-paste
		// review again, which must list p1 before p3 and then be cancelled.
		t.Views().Files().Focus()
		cherry(t.Views().Files(), "p")
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(reviewContent.Contains("master")).
			Cancel()
		if got := t.Git().GetCommitHash("HEAD"); got != pastedHead {
			t.Fail("Cancelling the re-paste review moved master to " + got)
		}
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()
	},
})
