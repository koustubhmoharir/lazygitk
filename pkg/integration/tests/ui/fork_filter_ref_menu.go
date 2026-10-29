package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterRefMenu = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w b on a commit with a local branch, a remote branch and a tag opens a menu in that order, and picking the tag shows it with its history",
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
		shell.EmptyCommitWithDate("base", "2001-01-01 12:00:00 +0000").
			EmptyCommitWithDate("m1", "2001-01-02 12:00:00 +0000").
			EmptyCommitWithDate("m2", "2001-01-03 12:00:00 +0000").
			// feat is a local branch at m1. The clone gives origin/feat at m1 too, and
			// feat tracks it, so the menu shows feat with its upstream.
			NewBranchFrom("feat", "HEAD~1").
			CloneIntoRemote("origin").
			SetBranchUpstream("feat", "origin/feat").
			// NewBranchFrom checks out feat, so the tag is made on feat, at m1.
			CreateAnnotatedTag("v1.0", "release", "feat")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("m1"))

		// Three refs point at m1, so w b opens a menu: local branches, then remote
		// branches, then tags.
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		t.ExpectPopup().Menu().
			Title(Equals("Show branch / tag")).
			ContainsLines(
				Contains("feat and origin/feat"),
				Contains("origin/feat"),
				Contains("tag v1.0"),
			).
			Select(Contains("tag v1.0")).
			Confirm()

		commits.IsFocused().
			Lines(Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.Views().Information().Content(Contains("Showing tag v1.0 and its history"))
		commits.Press(config.Keybinding{"<esc>"})
	},
})
