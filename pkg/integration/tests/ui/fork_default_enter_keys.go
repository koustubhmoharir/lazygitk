package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultEnterKeys = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "v and Space enter a submodule from Submodules and v enters a submodule entry in Files, with Escape returning to the parent; s w switches to the selected worktree; t has no r route",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the v, Space and s routes come from the fork catalogue.
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
		// Move the submodule's checked-out commit past the recorded one so the
		// parent's Files pane lists the submodule as modified.
		shell.Chdir("my_submodule_path")
		shell.EmptyCommit("submodule change")
		shell.Chdir("..")

		// A linked worktree on a branch that the main worktree does not use.
		shell.NewBranch("linked")
		shell.Checkout("master")
		shell.AddWorktreeCheckout("linked", "../linked-worktree")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		assertInParentRepo := func() {
			t.Views().Status().Content(Contains("repo")).Content(DoesNotContain("my_submodule_path"))
		}
		assertInSubmodule := func() {
			t.Views().Status().Content(Contains("my_submodule_path"))
		}

		assertInParentRepo()

		// a. v in Submodules enters the submodule; Escape returns to the parent.
		t.Views().Submodules().Focus().
			Lines(
				Contains("my_submodule_name").IsSelected(),
			).
			Press(config.Keybinding{"v"})
		assertInSubmodule()
		t.Views().Files().IsFocused().PressEscape()
		assertInParentRepo()

		// b. Space in Submodules enters the submodule as well; Escape returns.
		t.Views().Submodules().Focus().
			Lines(
				Contains("my_submodule_name").IsSelected(),
			).
			Press(config.Keybinding{"<space>"})
		assertInSubmodule()
		t.Views().Files().IsFocused().PressEscape()
		assertInParentRepo()

		// c. v on the modified submodule entry in Files enters the submodule.
		t.Views().Files().Focus().
			Lines(
				MatchesRegexp(` M.*my_submodule_path \(submodule\)`).IsSelected(),
			).
			Press(config.Keybinding{"v"})
		assertInSubmodule()
		t.Views().Files().IsFocused().PressEscape()
		assertInParentRepo()

		// d. s opens the s menu, and w switches to the selected worktree.
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

		// e. The t menu has no r route. Pressing r inside it does nothing, so
		// the t menu stays open; the menu is then cancelled.
		t.Views().Commits().Focus().Press(config.Keybinding{"t"})
		t.ExpectPopup().Menu().Title(Equals("t"))
		t.Views().Menu().Press(config.Keybinding{"r"})
		t.ExpectPopup().Menu().Title(Equals("t")).Cancel()
	},
})
