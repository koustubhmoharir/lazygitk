package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// The reason shown for the path copy when several rows are selected.
const forkCopyRangeRefusedReason = "Action does not support range selection, please select a single item"

var ForkCopyFileRangeRefused = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "For a range of files, the y copy menu leaves out the single-file copies in Files and Commit files and leaves the clipboard alone, while all files diff still copies the whole tree",
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
		shell.CreateFileAndAdd("dir/file2", "a\n")
		shell.Commit("first")
		// The second commit changes both files, so commit files has two rows to select.
		shell.UpdateFileAndAdd("dir/file1", "1st line\n2nd line\n")
		shell.UpdateFileAndAdd("dir/file2", "a\nb\n")
		shell.Commit("second")
		// Unstaged changes, so the working-tree diffs have content.
		shell.UpdateFile("dir/file1", "1st line\n2nd line\n3rd line\n")
		shell.UpdateFile("dir/file2", "a\nb\nfourth line\n")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		files := t.Views().Files().
			IsFocused().
			Lines(
				Contains("dir").IsSelected(),
				Contains("file1"),
				Contains("file2"),
			).
			NavigateToLine(Contains("file1"))

		// A single-file copy first, so the clipboard holds a known value to compare against.
		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"n"})
		t.ExpectToast(Equals("File name copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("file1"))

		// Shift+J extends the selection to file2, so the cursor sits on file2 and the range is file1 and file2.
		files.Press(config.Keybinding{"J"}).
			SelectedLines(
				Contains("file1"),
				Contains("file2"),
			)

		// A prefix menu leaves out commands that have nothing to act on here, so the
		// single-file copies (name, absolute path, diff) are not listed for a range.
		// The path copy, which has its own range guard, is listed and crossed out.
		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y")).
			Lines(
				Contains("Copy path to clipboard").IsSelected(),
				Contains("Copy all files diff"),
				Contains("Cancel"),
			).
			Tooltip(Contains("Disabled: " + forkCopyRangeRefusedReason)).
			Confirm().
			Tap(func() {
				t.ExpectToast(Equals("Disabled: " + forkCopyRangeRefusedReason))
			})

		// The refusals did not touch the clipboard, and the range is still selected.
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		t.FileSystem().FileContent("clipboard", Equals("file1"))
		files.SelectedLines(
			Contains("file1"),
			Contains("file2"),
		)

		// All files diff works on the whole tree, so the range does not block it.
		files.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"w"})
		t.ExpectToast(Equals("All files diff copied to clipboard"))
		t.FileSystem().
			FileContent("clipboard", Contains("+3rd line")).
			FileContent("clipboard", Contains("+fourth line"))

		// Commit files: the same refusal applies to a range of commit files. The flat y
		// menu has no all-files item here, so only the file-specific items are checked.
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
				Contains("file2"),
			).
			NavigateToLine(Contains("file1")).
			Press(config.Keybinding{"J"}).
			SelectedLines(
				Contains("file1"),
				Contains("file2"),
			)

		// Commit files' single-file copies are left out the same way; only the
		// path copy is listed, crossed out.
		commitFiles.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y")).
			Lines(
				Contains("Copy path to clipboard").IsSelected(),
				Contains("Cancel"),
			).
			Tooltip(Contains("Disabled: " + forkCopyRangeRefusedReason)).
			Confirm().
			Tap(func() {
				t.ExpectToast(Equals("Disabled: " + forkCopyRangeRefusedReason))
			})
		t.Views().Menu().Press(config.Keybinding{"<esc>"})

		// The clipboard still holds the all files diff copied above.
		t.FileSystem().FileContent("clipboard", Contains("+fourth line"))
		commitFiles.SelectedLines(
			Contains("file1"),
			Contains("file2"),
		)
	},
})
