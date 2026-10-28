package ui

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCopyFileKeys = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "The flat y copy menu copies file names, paths and diffs from the Files panel, and file names, diffs and contents from commit files",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the y routes come from the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
		loaded.OS.CopyToClipboardCmd = "printf '%s' {{text}} > clipboard"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateDir("dir")
		shell.CreateFileAndAdd("dir/file1", "1st line\n")
		shell.Commit("first")
		shell.UpdateFileAndAdd("dir/file1", "1st line\n2nd line\n")
		shell.Commit("second")
		// An unstaged change, so the working-tree diffs have content.
		shell.UpdateFile("dir/file1", "1st line\n2nd line\n3rd line\n")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		files := t.Views().Files().
			IsFocused().
			Lines(
				Contains("dir").IsSelected(),
				Contains("file1"),
			).
			NavigateToLine(Contains("file1"))

		// The y menu lists the flat entries and no nested copy menu.
		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Content(
			Contains("Copy file name").
				Contains("Copy all files diff").
				DoesNotContain("Copy to clipboard"),
		)
		t.Views().Menu().Press(config.Keybinding{"<esc>"})

		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"n"})
		t.ExpectToast(Equals("File name copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("file1"))

		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"p"})
		// The existing clipboard.copy-file-path route words its toast differently
		// from the nested "Relative path" item.
		t.ExpectToast(Equals("'dir/file1' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("dir/file1"))

		worktreeDir, _ := os.Getwd()
		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"f"})
		t.ExpectToast(Equals("File path copied to clipboard"))
		// On windows the following path would have backslashes, but we don't run integration tests on windows yet.
		t.FileSystem().FileContent("clipboard", Equals(worktreeDir+"/dir/file1"))

		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"d"})
		t.ExpectToast(Equals("File diff copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Contains("+3rd line"))

		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"w"})
		t.ExpectToast(Equals("All files diff copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Contains("+3rd line"))

		// Commit files: the same y keys copy the file as it is at that commit.
		t.Views().Commits().
			Focus().
			Lines(
				Contains("second").IsSelected(),
				Contains("first"),
			).
			Press(config.Keybinding{"v"})

		commitFiles := t.Views().CommitFiles().
			IsFocused().
			Lines(
				Contains("dir").IsSelected(),
				Contains("file1"),
			).
			NavigateToLine(Contains("file1"))

		commitFiles.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"n"})
		t.ExpectToast(Equals("File name copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("file1"))

		commitFiles.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"d"})
		t.ExpectToast(Equals("File diff copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Contains("+2nd line"))

		commitFiles.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"c"})
		t.ExpectToast(Equals("File content copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("1st line\n2nd line\n"))
	},
})
