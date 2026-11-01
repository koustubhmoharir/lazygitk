package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultCherryPickMarks = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default c y and c x marks: Files c is a no-op without marks, marks survive subview exit, and Escape in Commits clears them",
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

		// 0. No marks and no selection: in Files nothing under c applies, so c
		// opens no menu and leaves no popup and no change.
		t.Views().Files().Focus().Press(config.Keybinding{"c"})
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
	},
})
