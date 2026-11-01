package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultHistoryOperationsRebaseMerge = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default r rebase and m merge operations with explicit confirmations",
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
	},
})
