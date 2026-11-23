package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultBrowseAllFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default keys: v opens the Commits menu with d changed files and f all files, Space in Reflog opens the same menu",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, as fork_default_keymap_search does.
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
		shell.Commit("second")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Lines(
				Contains("second").IsSelected(),
				Contains("first"),
			).
			Press(config.Keybinding{"v"})

		t.ExpectPopup().Menu().
			Title(Equals("v")).
			ContainsLines(
				Contains("Changed files"),
				Contains("All files"),
			)
		t.Views().Menu().Press(config.Keybinding{"f"})

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

		t.Views().Commits().Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"d"})

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

		// Space in Reflog opens the same menu as v.
		t.Views().ReflogCommits().Focus().
			Press(config.Keybinding{"<space>"})
		t.ExpectPopup().Menu().Title(Equals("<space>"))
		t.Views().Menu().Press(config.Keybinding{"f"})
		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("All files"))
	},
})
