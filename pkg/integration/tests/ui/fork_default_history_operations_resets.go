package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultHistoryOperationsResets = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default s h/s m/s s reset operations with explicit confirmations, on topic already rebased onto master and merged with side",
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
		// Reproduces the state left by the rebase and merge steps of
		// fork_default_history_operations_rebase_merge: topic rebased onto master,
		// then merged with side, so HEAD is the merge commit.
		shell.CreateFileAndAdd("base.txt", "base\n").Commit("base").
			CreateFileAndAdd("m1.txt", "m1\n").Commit("m1").
			CreateFileAndAdd("m2.txt", "m2\n").Commit("m2").
			NewBranchFrom("topic", "master~2").
			CreateFileAndAdd("t1.txt", "t1\n").Commit("t1").
			NewBranchFrom("side", "topic~1").
			CreateFileAndAdd("s1.txt", "s1\n").Commit("s1").
			Checkout("topic").
			RunCommand([]string{"git", "rebase", "master"}).
			RunCommand([]string{"git", "merge", "--no-ff", "--no-edit", "side"})
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// Opens the real default s prefix menu, then picks its continuation.
		resetPrefix := func(commits *ViewDriver, key string) {
			commits.Press(config.Keybinding{"s"})
			t.ExpectPopup().Menu().Title(Equals("s"))
			t.Views().Menu().Press(config.Keybinding{key})
		}

		commits := t.Views().Commits()

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
