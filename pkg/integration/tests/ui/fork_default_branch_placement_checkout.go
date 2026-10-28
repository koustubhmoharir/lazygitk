package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultBranchPlacementCheckout = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default n b branch placement, s c selected checkout for branches and commits, and s d detached checkout of a tag",
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
		shell.EmptyCommit("one").
			NewBranch("feature").
			Checkout("master").
			EmptyCommit("two").
			CreateLightweightTag("v1", "HEAD").
			CloneIntoRemote("origin").
			EmptyCommit("three")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		oneOID := t.Git().GetCommitHash("HEAD~2")
		twoOID := t.Git().GetCommitHash("HEAD~1")
		threeOID := t.Git().GetCommitHash("HEAD")

		assertRef := func(ref, oid string) {
			if got := t.Git().GetCommitHash(ref); got != oid {
				t.Fail("Expected " + ref + " at " + oid + ", got " + got)
			}
		}
		assertCurrent := func(name, oid string) {
			t.Git().CurrentBranchName(name)
			if got := t.Git().GetCommitHash("HEAD"); got != oid {
				t.Fail("Expected HEAD " + oid + ", got " + got)
			}
		}
		// Opens the real default n prefix menu, then picks its b continuation.
		placeBranch := func(view *ViewDriver) {
			view.Press(config.Keybinding{"n"})
			t.ExpectPopup().Menu().Title(Equals("n"))
			t.Views().Menu().Press(config.Keybinding{"b"})
		}
		promptName := func(name string) {
			t.ExpectPopup().Prompt().
				Title(Contains("Local branch name at").Contains("(without checkout)")).
				Type(name).
				Confirm()
		}

		// 1. Create a branch at a non-HEAD commit from Commits without checkout.
		commits := t.Views().Commits()
		commits.Focus().NavigateToLine(Contains("two"))
		placeBranch(commits)
		promptName("from-commit")
		commits.IsFocused()
		assertRef("refs/heads/from-commit", twoOID)
		assertCurrent("master", threeOID)

		// 2. Moving the existing non-current branch requires explicit confirmation.
		commits.NavigateToLine(Contains("three"))
		placeBranch(commits)
		promptName("feature")
		t.ExpectPopup().Confirmation().
			Title(Equals("Move existing branch?")).
			Content(Contains("feature")).
			Confirm()
		commits.IsFocused()
		assertRef("refs/heads/feature", threeOID)
		assertCurrent("master", threeOID)

		// 3. Place a branch at the tip of a local branch from the Branches pane.
		branches := t.Views().Branches()
		branches.Focus().NavigateToLine(Contains("from-commit"))
		placeBranch(branches)
		promptName("from-branch")
		branches.IsFocused()
		assertRef("refs/heads/from-branch", twoOID)
		assertCurrent("master", threeOID)

		// 4. Place a branch at a remote branch from the RemoteBranches pane.
		// The fork's default inspect key (v) opens remote branches; Enter is a
		// no-op outside Files.
		t.Views().Remotes().Focus().
			Lines(
				Contains("origin").IsSelected(),
			).
			Press(config.Keybinding{"v"})
		remoteBranches := t.Views().RemoteBranches()
		remoteBranches.IsFocused().NavigateToLine(Contains("master"))
		placeBranch(remoteBranches)
		promptName("from-remote")
		remoteBranches.IsFocused()
		assertRef("refs/heads/from-remote", t.Git().GetCommitHash("refs/remotes/origin/master"))
		assertCurrent("master", threeOID)

		// 5. Escape cancels a visible prefix menu without changing focus or HEAD.
		t.Views().Commits().Focus().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s")).Cancel()
		t.Views().Menu().IsInvisible()
		commits.IsFocused()
		assertCurrent("master", threeOID)

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
