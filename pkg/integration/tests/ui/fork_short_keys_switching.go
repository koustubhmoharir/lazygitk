package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShortKeysSwitching = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default switching keys: s p switches to the previous branch and back, v and Space in Worktrees show a linked worktree's commits without switching, and g s a applies a stash and keeps it",
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
		shell.CreateFileAndAdd("file", "original\n").Commit("first")

		// A linked worktree on a branch that has a commit master lacks.
		shell.NewBranch("linked").
			EmptyCommit("linked commit").
			Checkout("master")
		shell.AddWorktreeCheckout("linked", "../linked-worktree")

		// Checking out feature and then master leaves the latest reflog checkout
		// as "moving from feature to master", which s p names as the previous branch.
		shell.NewBranch("feature").
			Checkout("master")

		// A stash of a tracked file, made on master so it applies cleanly there.
		shell.UpdateFile("file", "stashed\n").
			Stash("stash one")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// Opens the real default prefix menu for the first key, then picks the
		// continuation key from the visible menu.
		chord := func(view *ViewDriver, title string, keys ...string) {
			view.Press(config.Keybinding{keys[0]})
			t.ExpectPopup().Menu().Title(Equals(title))
			for _, key := range keys[1:] {
				t.Views().Menu().Press(config.Keybinding{key})
			}
		}

		// 1. s opens the s menu, which names the previous branch; p switches to it.
		t.Views().Files().IsFocused()
		chord(t.Views().Files(), "s", "s")
		t.Views().Menu().Content(Contains("Switch to previous branch: feature"))
		t.Views().Menu().Press(config.Keybinding{"p"})
		t.Git().CurrentBranchName("feature")

		// Running s p again returns to master. The switch moved focus to Branches,
		// so Files is focused again before the second s.
		t.Views().Files().Focus()
		chord(t.Views().Files(), "s", "s")
		t.Views().Menu().Content(Contains("Switch to previous branch: master"))
		t.Views().Menu().Press(config.Keybinding{"p"})
		t.Git().CurrentBranchName("master")

		// 2. v in Worktrees shows the linked worktree's commits without switching.
		worktrees := t.Views().Worktrees()
		worktrees.
			Focus().
			Lines(
				Contains("(main worktree)").IsSelected(),
				Contains("linked-worktree"),
			).
			NavigateToLine(Contains("linked-worktree")).
			Press(config.Keybinding{"v"})
		t.Views().SubCommits().
			IsFocused().
			Content(Contains("linked commit"))
		t.Views().Status().Content(DoesNotContain("linked-worktree"))
		t.Git().CurrentBranchName("master")

		// Escape returns to Worktrees.
		t.Views().SubCommits().PressEscape()
		t.Views().Worktrees().IsFocused()

		// Space does the same.
		t.Views().Worktrees().
			Press(config.Keybinding{"<space>"})
		t.Views().SubCommits().
			IsFocused().
			Content(Contains("linked commit"))
		t.Views().Status().Content(DoesNotContain("linked-worktree"))
		t.Git().CurrentBranchName("master")
		t.Views().SubCommits().PressEscape()
		t.Views().Worktrees().IsFocused()

		// 3. g s a applies the stash: the change returns to Files and the stash
		// entry stays, because apply does not pop it.
		t.Views().Files().Content(DoesNotContain("file"))
		t.Views().Stash().
			Focus().
			Lines(
				Contains("stash one").IsSelected(),
			).
			Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("g s"))
		t.Views().Menu().Press(config.Keybinding{"a"})

		t.ExpectPopup().Confirmation().
			Title(Equals("Stash apply")).
			Content(Contains("Are you sure you want to apply this stash entry?")).
			Confirm()

		t.Views().Files().Content(Contains("file"))
		t.Views().Stash().Lines(
			Contains("stash one"),
		)
	},
})
