package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFlatCheckout = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default flat checkout keys under s: s c switches to a local branch, creates a tracking branch for a remote-only branch, names a commit or tag through the prompt, and s d detaches HEAD",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "one\n").Commit("one")
		// at-one names a branch at commit "one", and v1 a tag there.
		shell.RunCommand([]string{"git", "branch", "at-one"})
		shell.CreateLightweightTag("v1", "HEAD")
		// shared is first made at "one", so origin/shared is left at "one" when the
		// remote is cloned; the local branch is moved to "two" below.
		shell.RunCommand([]string{"git", "branch", "shared"})

		shell.NewBranch("feature").
			EmptyCommit("feature commit").
			Checkout("master")
		shell.EmptyCommit("two")

		// remote-only is removed locally after the clone, so only origin has it.
		shell.NewBranch("remote-only").
			Checkout("master")
		shell.CloneIntoRemote("origin")
		shell.RunCommand([]string{"git", "branch", "-D", "remote-only"})
		shell.RunCommand([]string{"git", "branch", "-f", "shared", "master"})
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		oneOID := t.Git().GetCommitHash("HEAD~1")
		twoOID := t.Git().GetCommitHash("HEAD")

		assertHead := func(name, oid string) {
			t.Git().CurrentBranchName(name)
			if got := t.Git().GetCommitHash("HEAD"); got != oid {
				t.Fail("Expected HEAD " + oid + ", got " + got)
			}
		}
		// Opens the real default s prefix menu and picks its continuation key.
		chordS := func(view *ViewDriver, key string) {
			view.Press(config.Keybinding{"s"})
			t.ExpectPopup().Menu().Title(Equals("s"))
			t.Views().Menu().Press(config.Keybinding{key})
		}
		// Opens the s c checkout prompt for the commit, tag or branch of the view.
		checkoutPrompt := func(view *ViewDriver, lineMatcher *TextMatcher, oid string) *PromptDriver {
			view.NavigateToLine(lineMatcher)
			chordS(view, "c")
			return t.ExpectPopup().Prompt().
				Title(Equals("Local branch name for checkout at " + oid[:8]))
		}

		// 1. s c on a local branch switches to it with no menu.
		branches := t.Views().Branches()
		branches.Focus().NavigateToLine(Contains("feature"))
		chordS(branches, "c")
		t.Views().Menu().IsInvisible()
		branches.IsFocused()
		assertHead("feature", t.Git().GetCommitHash("refs/heads/feature"))

		// 2. s c on a remote-only branch creates a local branch tracking it and
		// switches to it, with no menu.
		t.Views().Remotes().Focus().
			Lines(
				Contains("origin").IsSelected(),
			).
			Press(config.Keybinding{"v"})
		remoteBranches := t.Views().RemoteBranches()
		remoteBranches.IsFocused().NavigateToLine(Contains("remote-only"))
		chordS(remoteBranches, "c")
		t.Views().Menu().IsInvisible()
		t.Views().Branches().IsFocused().
			SelectedLine(Contains("remote-only").Contains("✓"))
		assertHead("remote-only", t.Git().GetCommitHash("refs/remotes/origin/remote-only"))

		// 3. s c on a remote branch whose local branch exists switches to that local
		// branch. The local shared is at "two" and origin/shared at "one", so HEAD
		// must end at the local branch's commit.
		t.Views().Remotes().Focus().
			Lines(
				Contains("origin").IsSelected(),
			).
			Press(config.Keybinding{"v"})
		remoteBranches.IsFocused().NavigateToLine(Contains("shared"))
		chordS(remoteBranches, "c")
		t.Views().Menu().IsInvisible()
		t.Views().Branches().IsFocused()
		assertHead("shared", twoOID)
		if got := t.Git().GetCommitHash("refs/remotes/origin/shared"); got != oneOID {
			t.Fail("Expected origin/shared to stay at " + oneOID + ", got " + got)
		}

		// 4. s c on a commit that a branch already names: the prompt lists that branch
		// first, and confirming its name checks it out without a second dialog.
		commits := t.Views().Commits()
		checkoutPrompt(commits.Focus(), Contains("one"), oneOID).
			SuggestionTopLines(Contains("at-one")).
			Type("at-one").
			Confirm()
		branches.IsFocused()
		assertHead("at-one", oneOID)

		// 5. s c on a commit with a new name creates the branch there and checks it
		// out. The branches at the commit are listed first.
		checkoutPrompt(commits.Focus(), Contains("two"), twoOID).
			SuggestionTopLines(MatchesRegexp(`master|shared`)).
			Type("from-two").
			Confirm()
		branches.IsFocused()
		assertHead("from-two", twoOID)

		// 6. s c on a commit naming a branch at another commit asks before moving it.
		// The feature branch is moved to "two" and checked out after confirmation.
		checkoutPrompt(commits.Focus(), Contains("two"), twoOID).
			Type("feature").
			Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Move branch and check out?")).
			Content(Contains("feature")).
			Confirm()
		branches.IsFocused()
		assertHead("feature", twoOID)

		// 7. s c on a tag opens the same prompt for the tag's commit.
		tags := t.Views().Tags()
		checkoutPrompt(tags.Focus(), Contains("v1"), oneOID).
			Type("from-tag").
			Confirm()
		branches.IsFocused()
		assertHead("from-tag", oneOID)

		// 8. s d on a commit detaches HEAD at that commit, with no menu.
		commits.Focus().NavigateToLine(Contains("two"))
		chordS(commits, "d")
		t.Views().Menu().IsInvisible()
		commits.IsFocused()
		assertHead("HEAD", twoOID)
		t.Shell().RunCommandExpectError([]string{"git", "symbolic-ref", "-q", "HEAD"})

		// 9. s d on a branch detaches HEAD at the branch's commit.
		branches.Focus().NavigateToLine(Contains("at-one"))
		chordS(branches, "d")
		t.Views().Menu().IsInvisible()
		branches.IsFocused()
		assertHead("HEAD", oneOID)
		t.Shell().RunCommandExpectError([]string{"git", "symbolic-ref", "-q", "HEAD"})
	},
})
