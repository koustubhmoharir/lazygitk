package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPrefixEdgeCases = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Prefix menus with explicit leaf and descendant on one key, palette paths, and pointer or focus changes while a prefix is open",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, so only the command-ID overrides below apply.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"

		// n is an explicit leaf for tag creation and an explicit descendant z for
		// revert, so the n prefix stays a visible menu offering both choices.
		loaded.CommandBindings = map[string]config.CommandBindingPaths{
			"tag.create-at-selection": {{"n"}},
			"commit.revert":           {{"n", "z"}},
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one").
			CreateFileAndAdd("two.txt", "two\n").Commit("two").
			CreateFileAndAdd("three.txt", "three\n").Commit("three")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		head := t.Git().GetCommitHash("HEAD")
		twoOID := t.Git().GetCommitHash("HEAD~1")
		commits := t.Views().Commits()

		// 1. The command palette shows the overridden revert path and the default
		// three-key rebase base path.
		commits.Focus()
		commits.Press(config.Keybinding{"?"})
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Revert")
		t.Views().Menu().SelectedLine(Contains("n z").Contains("Revert"))
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().MenuFilter().IsInvisible()

		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Mark as base commit for rebase")
		t.Views().Menu().SelectedLine(Contains("g r b").Contains("Mark as base commit for rebase"))
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().MenuFilter().IsInvisible()
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().Menu().IsInvisible()
		commits.IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Browsing the command palette moved HEAD to " + got)
		}

		// 2. n opens a visible menu with the tag entry and the z continuation.
		commits.Press(config.Keybinding{"n"})
		t.ExpectPopup().Menu().Title(Equals("n"))
		t.Views().Menu().Content(Contains("Tag commit").Contains("Revert"))
		t.Views().Menu().Press(config.Keybinding{"z"})
		t.ExpectPopup().Confirmation().
			Title(Equals("Revert commit")).
			Content(Contains("Are you sure you want to revert " + head[:8] + "?")).
			Cancel()
		commits.IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Cancelling revert moved HEAD to " + got)
		}

		commits.Press(config.Keybinding{"n"})
		t.ExpectPopup().Menu().Title(Equals("n"))
		t.Views().Menu().NavigateToLine(Contains("Tag commit")).
			SelectedLine(Contains("Tag commit")).PressEnter()
		t.ExpectPopup().CommitMessagePanel().Title(Equals("Tag name")).Cancel()
		commits.IsFocused()
		t.Git().TagNamesAt(head, []string{})
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Cancelling tag creation moved HEAD to " + got)
		}

		// 3. A click on another Commits row does not dismiss the open s menu, and
		// s then reaches the soft-reset confirmation for the captured commit two.
		commits.NavigateToLine(Contains("two")).Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		commits.Click(0, 0)
		t.Views().Menu().IsVisible().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectPopup().Confirmation().
			Title(Equals("Soft reset")).
			Content(Contains(twoOID)).
			Cancel()
		commits.IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Cancelling soft reset moved HEAD to " + got)
		}

		// 4. A click on the Files view while the g menu is open leaves the menu up
		// and runs nothing; Escape then dismisses it without changing HEAD.
		commits.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Files().Click(0, 0)
		t.Views().Menu().IsVisible().Title(Equals("g"))
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Clicking Files with the g menu open moved HEAD to " + got)
		}
		t.Views().Menu().PressEscape()
		t.Views().Menu().IsInvisible()
		commits.IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Dismissing the g menu moved HEAD to " + got)
		}
	},
})
