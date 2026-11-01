package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultBranchPlacementCheckoutSelectedCheckout = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default s c selected checkout for branches and commits, and s d detached checkout of a tag",
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
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		// Reproduces the state left by the branch placement steps of
		// fork_default_branch_placement_checkout_placement: from-branch exists at
		// two, and master is at three with tag v1 at two.
		shell.EmptyCommit("one").
			EmptyCommit("two").
			CreateLightweightTag("v1", "HEAD").
			NewBranch("from-branch").
			Checkout("master").
			EmptyCommit("three")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		oneOID := t.Git().GetCommitHash("HEAD~2")
		twoOID := t.Git().GetCommitHash("HEAD~1")

		assertCurrent := func(name, oid string) {
			t.Git().CurrentBranchName(name)
			if got := t.Git().GetCommitHash("HEAD"); got != oid {
				t.Fail("Expected HEAD " + oid + ", got " + got)
			}
		}

		commits := t.Views().Commits()
		branches := t.Views().Branches()

		// 6. Selected branch s c switches to the existing branch from-branch with
		// no menu.
		branches.Focus().NavigateToLine(Contains("from-branch")).Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"c"})
		t.Views().Menu().IsInvisible()
		branches.IsFocused()
		assertCurrent("from-branch", twoOID)

		// 7. Selected commit s c opens the named-checkout prompt for that commit and
		// creates the new branch there.
		commits.Focus().NavigateToLine(Contains("one")).Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"c"})
		t.ExpectPopup().Prompt().
			Title(Equals("Local branch name for checkout at " + oneOID[:8])).
			Type("co-new").
			Confirm()
		branches.IsFocused().SelectedLine(Contains("co-new"))
		assertCurrent("co-new", oneOID)

		// 8. Selected tag s d detaches HEAD at the tag's commit, with no menu.
		tags := t.Views().Tags()
		tags.Focus().NavigateToLine(Contains("v1")).Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"d"})
		t.Views().Menu().IsInvisible()
		tags.IsFocused()
		if got := t.Git().GetCommitHash("HEAD"); got != twoOID {
			t.Fail("Expected detached HEAD " + twoOID + ", got " + got)
		}
		t.Shell().RunCommandExpectError([]string{"git", "symbolic-ref", "-q", "HEAD"})
	},
})
