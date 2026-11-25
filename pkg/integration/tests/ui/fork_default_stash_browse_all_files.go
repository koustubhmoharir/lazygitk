package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultStashBrowseAllFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default keys: Enter in Stash lists the changed files, v opens the Stash menu with d changed files and f all files, and f lists the stashed tree",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, as fork_default_browse_all_files does.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.Gui.ShowRootItemInFileTree = true
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateDir("a/b")
		shell.CreateFileAndAdd("a/x.txt", "x\n")
		shell.CreateFileAndAdd("a/b/y.txt", "y\n")
		shell.CreateFileAndAdd("top.txt", "top\n")
		shell.CreateFileAndAdd("gone.txt", "gone\n")
		shell.Commit("first")
		shell.UpdateFileAndAdd("a/x.txt", "x changed\n")
		shell.CreateFileAndAdd("new.txt", "new\n")
		shell.DeleteFileAndAdd("gone.txt")
		shell.Stash("wip")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Enter in Stash lists the changed files, as before.
		t.Views().Stash().
			Focus().
			Lines(
				Contains("wip").IsSelected(),
			).
			Press(config.Keybinding{"<enter>"})

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("Diff files")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▼ a"),
				Equals("    M x.txt"),
				Equals("  D gone.txt"),
				Equals("  A new.txt"),
			)
		t.Views().CommitFiles().PressEscape()

		// v in Stash opens the same menu as in Commits.
		t.Views().Stash().Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().
			Title(Equals("v")).
			ContainsLines(
				Contains("Changed files"),
				Contains("All files"),
			)
		t.Views().Menu().Press(config.Keybinding{"f"})

		// v f lists every file of the stashed tree: the changed files keep their
		// status, and the unchanged top.txt has a blank status.
		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("All files")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ a"),
				Equals("  D gone.txt"),
				Equals("  A new.txt"),
				Equals("    top.txt"),
			)
		t.Views().CommitFiles().PressEscape()
		t.Views().Stash().IsFocused()

		// v d lists the changed files again.
		t.Views().Stash().Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"d"})
		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("Diff files"))
	},
})
