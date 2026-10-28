package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w b shows a branch with its upstream and their ancestors, from Branches and from Commits, and Escape restores the full list",
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
		// Fixed dates keep the default log order, newest first, the same in every run.
		// master has base and m1.
		shell.EmptyCommitWithDate("base", "2001-01-01 12:00:00 +0000").
			EmptyCommitWithDate("m1", "2001-01-02 12:00:00 +0000").
			// feat starts at base with f1, and tracks origin/feat at f1.
			NewBranchFrom("feat", "HEAD~1").
			EmptyCommitWithDate("f1", "2001-01-03 12:00:00 +0000").
			// The clone is a copy of this repo, so origin/feat starts at f1.
			CloneIntoRemote("origin").
			SetBranchUpstream("feat", "origin/feat").
			// f2 is local work that origin does not have.
			EmptyCommitWithDate("f2", "2001-01-05 12:00:00 +0000").
			// origin/feat moves on to r1, a child of f1, through the remote-tracking
			// ref alone: no local branch points at r1.
			RunShellCommand(`git update-ref refs/remotes/origin/feat $(GIT_AUTHOR_DATE='2001-01-04 12:00:00 +0000' GIT_COMMITTER_DATE='2001-01-04 12:00:00 +0000' git commit-tree 'feat~1^{tree}' -p feat~1 -m r1)`).
			Checkout("master").
			// other starts at base with o1 and has no upstream.
			NewBranchFrom("other", "HEAD~1").
			EmptyCommitWithDate("o1", "2001-01-06 12:00:00 +0000").
			Checkout("master").
			// side is a second local branch at m1, so that w b on m1 opens a menu. It
			// is not at o1: case 2 needs o1 to have only one branch. It is made here
			// because a branch created during the run did not reach the Branches pane
			// in this harness, even after a refresh.
			RunCommand([]string{"git", "branch", "side", "master"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Case 1: from Branches. feat with its upstream origin/feat shows f2, r1, f1
		// and base. m1 and o1 are hidden. Commits gets focus, with the cursor on feat's tip.
		t.Views().Branches().Focus().NavigateToLine(Contains("feat"))
		t.Views().Branches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		commits := t.Views().Commits()
		commits.IsFocused().
			Lines(Contains("f2"), Contains("r1"), Contains("f1"), Contains("base")).
			SelectedLines(Contains("f2"))
		t.Views().Information().Content(Contains("Showing feat and origin/feat"))

		// Escape clears the scope: the full list is back, and the cursor is on f2 again.
		commits.Press(config.Keybinding{"<esc>"})
		commits.Lines(Contains("o1"), Contains("f2"), Contains("r1"), Contains("f1"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("f2"))

		// Case 2: from Commits. o1 is the tip of other, which has no upstream. Only
		// one local branch points at it, so w b applies it directly.
		commits.NavigateToLine(Contains("o1"))
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		commits.IsFocused().
			Lines(Contains("o1"), Contains("base")).
			SelectedLines(Contains("o1"))
		t.Views().Information().Content(Contains("Showing other (no tracking branch)"))
		commits.Press(config.Keybinding{"<esc>"})

		// Case 3: m1 has two local branches, master and side, so w b opens a menu.
		commits.NavigateToLine(Contains("m1"))
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		// The menu lists the branches at m1 and applies the one chosen. Neither has an
		// upstream, so the result shows m1 and base.
		t.ExpectPopup().Menu().
			Title(Equals("Show branch and its upstream")).
			Content(Contains("master")).
			Content(Contains("side")).
			Select(Contains("side")).
			Confirm()

		commits.IsFocused().
			Lines(Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("Showing side (no tracking branch)"))
		commits.Press(config.Keybinding{"<esc>"})

		// Case 4: base has no local branch at its tip, so w b is not offered in the
		// w menu. The menu still lists the other filters.
		commits.NavigateToLine(Contains("base"))
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().
			Title(Equals("w")).
			ContainsLines(Contains("Show selected commits and their ancestors")).
			Content(DoesNotContain("Show branch and its upstream")).
			Cancel()
	},
})
