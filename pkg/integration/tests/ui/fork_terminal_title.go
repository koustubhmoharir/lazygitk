package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkTerminalTitle = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "The terminal title names the repository being viewed: at startup, after entering and leaving a submodule, after an s w worktree switch, and after s n opens another repository",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the v, s and n routes come from the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("first")
		shell.CloneIntoSubmodule("my_submodule_name", "my_submodule_path")
		shell.GitAddAll()
		shell.Commit("add submodule")

		// A linked worktree on a branch that the main worktree does not use.
		shell.NewBranch("linked")
		shell.Checkout("master")
		shell.AddWorktreeCheckout("linked", "../linked-worktree")

		// A second repository to open by path. Its name is what the title should show.
		shell.CloneNonBare("other")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// The harness paths are absolute and may sit under any directory, so the
		// title is matched on its final path segment.
		assertTitleEndsWith := func(name string) {
			t.ExpectTerminalTitle(MatchesRegexp(`(^|/)` + name + `$`))
		}

		// 1. At startup the title is the main repository.
		assertTitleEndsWith("repo")

		// 2. Entering the submodule with v changes the title; Escape restores it.
		t.Views().Submodules().Focus().
			Lines(
				Contains("my_submodule_name").IsSelected(),
			).
			Press(config.Keybinding{"v"})
		t.Views().Status().Content(Contains("my_submodule_path"))
		assertTitleEndsWith("my_submodule_path")
		t.Views().Files().IsFocused().PressEscape()
		t.Views().Status().Content(Contains("repo")).Content(DoesNotContain("my_submodule_path"))
		assertTitleEndsWith("repo")

		// 3. s w in Worktrees switches to the linked worktree.
		t.Views().Worktrees().Focus().
			Lines(
				Contains("(main worktree)").IsSelected(),
				Contains("linked-worktree"),
			).
			NavigateToLine(Contains("linked-worktree")).
			Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"w"})
		t.Views().Status().Content(Contains("linked-worktree"))
		assertTitleEndsWith("linked-worktree")

		// 4. s n opens the other repository by path. The prompt is prefilled with
		// the parent of the launch directory, which holds the sibling repositories.
		t.Views().Commits().Focus().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"n"})
		t.ExpectPopup().Prompt().Title(Equals("Repository path")).Type("other")
		t.ExpectPopup().Prompt().Title(Equals("Repository path")).Confirm()
		t.Views().Status().Content(Contains("other → master"))
		assertTitleEndsWith("other")
	},
})
