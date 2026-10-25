package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCommandBindingOverrides = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Command-ID overrides, explicit legacy fields and single-key displacement over the real default profile",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config; the explicit overrides below are then the only ones.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"

		// Command-ID paths take precedence over explicit legacy fields and fork
		// defaults. An empty non-nil path list removes the keyboard paths only.
		loaded.CommandBindings = map[string]config.CommandBindingPaths{
			"commit.combine-selected": {{"<f6>"}},
			"commit.checkout-menu":    {},
			"commit.revert":           {{"<f9>"}},
			"tag.create-at-selection": {{"n"}},
		}
		// Explicit legacy fields: the ID path for commit.revert must win over this.
		loaded.Keybinding.Commits.CreateMoveBranch = config.Keybinding{"<f7>"}
		loaded.Keybinding.Commits.RevertCommit = config.Keybinding{"<f8>"}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one").
			CreateFileAndAdd("two.txt", "two\n").Commit("two").
			CreateFileAndAdd("three.txt", "three\n").Commit("three")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		head := t.Git().GetCommitHash("HEAD")
		oneOID := t.Git().GetCommitHash("HEAD~2")
		twoOID := t.Git().GetCommitHash("HEAD~1")
		commits := t.Views().Commits()

		// 1. The ID path moves combine to f6, so the default g c route is gone.
		commits.Focus().NavigateToLine(Contains("two")).Press(config.Keybinding{"<shift+up>"})
		commits.Press(config.Keybinding{"<f6>"})
		t.ExpectPopup().Confirmation().
			Title(Equals("Review commit combination")).
			Content(Contains("Unselected commits remain separate")).
			Cancel()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Cancelling combine review moved HEAD to " + got)
		}
		commits.Press(config.Keybinding{"<esc>"}).IsFocused()

		commits.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Content(DoesNotContain("Combine selected commits"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		t.Views().Menu().IsInvisible()
		commits.IsFocused()

		// 2. The empty path list removes the s c checkout route but keeps the
		// action in the command palette; reset continuations are untouched.
		commits.Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().
			Title(Equals("s")).
			ContainsLines(Contains("Hard reset"), Contains("Mixed reset"), Contains("Soft reset"))
		t.Views().Menu().Content(DoesNotContain("Checkout"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		t.Views().Menu().IsInvisible()
		commits.IsFocused()

		commits.Press(config.Keybinding{"?"})
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Checkout")
		t.Views().Menu().Content(Contains("Checkout"))
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().MenuFilter().IsInvisible()
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().Menu().IsInvisible()
		commits.IsFocused()

		// 3. Explicit legacy CreateMoveBranch is f7 and places a ref without checkout.
		commits.NavigateToLine(Contains("one")).Press(config.Keybinding{"<f7>"})
		t.ExpectPopup().Prompt().
			Title(Contains("Local branch name at").Contains("(without checkout)")).
			Type("via-legacy").
			Confirm()
		commits.IsFocused()
		if got := t.Git().GetCommitHash("refs/heads/via-legacy"); got != oneOID {
			t.Fail("Expected refs/heads/via-legacy at " + oneOID + ", got " + got)
		}
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Placing a branch without checkout moved HEAD to " + got)
		}

		// 4. The command-ID path for commit.revert replaces the explicit legacy
		// f8 route, so f8 opens nothing at all.
		commits.Press(config.Keybinding{"<f8>"})
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()
		commits.IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Pressing the superseded revert legacy key moved HEAD to " + got)
		}

		// f9 is the command-ID route and opens the revert confirmation.
		commits.Press(config.Keybinding{"<f9>"})
		t.ExpectPopup().Confirmation().
			Title(Equals("Revert commit")).
			Content(Contains("Are you sure you want to revert " + oneOID[:8] + "?")).
			Cancel()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Cancelling revert moved HEAD to " + got)
		}
		commits.IsFocused()

		// 5. The explicit single-key n displaces the default n prefix menu, so
		// the tag-creation prompt opens directly.
		commits.NavigateToLine(Contains("two")).Press(config.Keybinding{"n"})
		t.Views().Menu().IsInvisible()
		t.ExpectPopup().CommitMessagePanel().Title(Equals("Tag name")).Cancel()
		commits.IsFocused()
		t.Git().TagNamesAt(twoOID, []string{})
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Cancelling tag creation moved HEAD to " + got)
		}
	},
})
