package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultHistoryOperations = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default r rebase, m merge and s h/s m/s s reset operations with explicit confirmations",
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
			CreateFileAndAdd("m1.txt", "m1\n").Commit("m1").
			CreateFileAndAdd("m2.txt", "m2\n").Commit("m2").
			NewBranchFrom("topic", "master~2").
			CreateFileAndAdd("t1.txt", "t1\n").Commit("t1").
			NewBranchFrom("side", "topic~1").
			CreateFileAndAdd("s1.txt", "s1\n").Commit("s1").
			Checkout("topic")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// Opens the real default s prefix menu, then picks its continuation.
		resetPrefix := func(commits *ViewDriver, key string) {
			commits.Press(config.Keybinding{"s"})
			t.ExpectPopup().Menu().Title(Equals("s"))
			t.Views().Menu().Press(config.Keybinding{key})
		}
		assertAncestor := func(ancestor, descendant string) {
			t.Shell().RunCommand([]string{"git", "merge-base", "--is-ancestor", ancestor, descendant})
		}

		// 1a. Rebase the current branch (topic) onto the selected branch master
		// from the Branches pane using the ordinary/simple rebase option.
		branches := t.Views().Branches()
		branches.Focus().NavigateToLine(Contains("master")).Press(config.Keybinding{"r"})
		t.ExpectPopup().Menu().
			Title(Equals("Rebase 'topic'")).
			Select(Contains("Simple rebase onto 'master'")).
			Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Rebase 'topic'")).
			Content(Contains("topic").Contains("onto selected source").Contains("master").Contains("Mode:")).
			Confirm()

		commits := t.Views().Commits()
		commits.TopLines(Contains("t1"))
		t.Git().CurrentBranchName("topic")
		assertAncestor("master", "topic")

		// 1b. Merge the side branch into topic using the plain configured merge.
		branches.Focus().NavigateToLine(Contains("side")).Press(config.Keybinding{"m"})
		t.ExpectPopup().Menu().
			Title(Equals("Merge")).
			Select(MatchesRegexp(`^[[:space:]]*m Merge using configured options`)).
			Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Merge")).
			Content(Contains(`Current branch: "topic"`).Contains(`Selected source: "side"`).Contains("Mode: Merge using configured options")).
			Confirm()

		commits.TopLines(Contains(`Merge branch "side" into "topic"`))
		t.Git().CurrentBranchName("topic")
		assertAncestor("side", "topic")

		// 1c. Soft reset: HEAD is the merge commit, so HEAD~2 is m2 (two below HEAD).
		m2 := t.Git().GetCommitHash("HEAD~2")
		m1 := t.Git().GetCommitHash("HEAD~3")
		mergeHead := t.Git().GetCommitHash("HEAD")

		commits.Focus().NavigateToLine(Contains("m2"))
		resetPrefix(commits, "s")
		t.ExpectPopup().Confirmation().
			Title(Equals("Soft reset")).
			Content(Contains(m2[:7]).Contains("staged changes")).
			Cancel()
		if got := t.Git().GetCommitHash("HEAD"); got != mergeHead {
			t.Fail("Cancelling soft reset moved HEAD to " + got)
		}

		resetPrefix(commits, "s")
		t.ExpectPopup().Confirmation().
			Title(Equals("Soft reset")).
			Content(Contains(m2[:7]).Contains("staged changes")).
			Confirm()
		if got := t.Git().GetCommitHash("HEAD"); got != m2 {
			t.Fail("Expected HEAD " + m2 + " after soft reset, got " + got)
		}
		t.Views().Files().ContainsLines(Contains("t1.txt"))

		// 1d. Hard reset to m1, one commit below the soft-reset HEAD, discards the
		// staged changes and leaves the Files pane empty.
		commits.Focus().NavigateToLine(Contains("m1"))
		resetPrefix(commits, "h")
		t.ExpectPopup().Confirmation().
			Title(Equals("Hard reset")).
			Content(Contains(m1[:7]).Contains("untracked or ignored")).
			Confirm()
		t.Views().Files().IsEmpty()
		if got := t.Git().GetCommitHash("HEAD"); got != m1 {
			t.Fail("Expected HEAD " + m1 + " after hard reset, got " + got)
		}

		// 1e. Mixed reset is only opened and cancelled; nothing may change.
		commits.Focus().NavigateToLine(Contains("base"))
		resetPrefix(commits, "m")
		t.ExpectPopup().Confirmation().
			Title(Equals("Mixed reset")).
			Content(Contains("unstaged")).
			Cancel()
		if got := t.Git().GetCommitHash("HEAD"); got != m1 {
			t.Fail("Cancelling mixed reset moved HEAD to " + got)
		}
		t.Views().Files().IsEmpty()
		t.Git().CurrentBranchName("topic")
	},
})
