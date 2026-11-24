package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultFilesEnterAndCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Files keys: Enter focuses a file's diff and a directory's diff without collapsing it, with Escape returning to Files; c with nothing in progress opens the commit message box and commits",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the Enter and c routes come from the fork catalogue.
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
		shell.
			CreateFileAndAdd("file.txt", "base\n").
			CreateFileAndAdd("dir/inner.txt", "inner\n").
			Commit("base")
		shell.UpdateFile("file.txt", "changed\n")
		shell.UpdateFile("dir/inner.txt", "changed inner\n")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// a. Enter on a file focuses its diff; Escape returns to Files.
		t.Views().Files().
			Focus().
			NavigateToLine(Contains("file.txt")).
			PressEnter()
		t.Views().Main().IsFocused()
		t.Views().Main().PressEscape()
		t.Views().Files().IsFocused()

		// b. Enter on a directory focuses the main view and leaves the directory open.
		t.Views().Files().
			NavigateToLine(Contains("dir")).
			PressEnter()
		t.Views().Main().IsFocused()
		t.Views().Main().PressEscape()
		t.Views().Files().
			IsFocused().
			Content(Contains("inner.txt"))

		// c. c with nothing in progress commits: stage the file, then c opens the
		// commit message box and the commit is made.
		t.Views().Files().
			NavigateToLine(Contains("file.txt")).
			Press(config.Keybinding{"<space>"})
		t.Views().Files().Press(config.Keybinding{"c"})
		t.ExpectPopup().CommitMessagePanel().Type("enter and c commit").Confirm()
		t.Views().Commits().TopLines(Contains("enter and c commit"))
		t.Views().Files().Content(DoesNotContain("file.txt"))
	},
})
