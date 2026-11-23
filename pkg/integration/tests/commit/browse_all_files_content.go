package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesContent = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In the full listing of a commit's files, a file shows its content, a deleted file its content before the commit, a binary file a note, and a folder its diff",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateDir("a/b")
		shell.CreateFileAndAdd("a/x.txt", "x\n")
		shell.CreateFileAndAdd("a/b/y.txt", "y\n")
		shell.CreateFileAndAdd("top.txt", "top\n")
		shell.CreateFileAndAdd("gone.txt", "gone\n")
		shell.Commit("first")
		shell.UpdateFileAndAdd("a/x.txt", "x changed\n")
		shell.CreateFileAndAdd("new.txt", "new\n")
		shell.CreateFileAndAdd("bin.dat", "a\x00b")
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
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"f"})

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("All files")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ a"),
				Equals("  A bin.dat"),
				Equals("  D gone.txt"),
				Equals("  A new.txt"),
				Equals("    top.txt"),
			)

		// Expanding a shows x.txt, which the commit changed, and the unchanged b folder.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  ▶ a")).
			Press(keys.Universal.GoInto).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ a"),
				Equals("    ▶ b"),
				Equals("    M x.txt"),
				Equals("  A bin.dat"),
				Equals("  D gone.txt"),
				Equals("  A new.txt"),
				Equals("    top.txt"),
			)

		// An unchanged file shows its content at the commit.
		t.Views().CommitFiles().
			NavigateToLine(Equals("    top.txt"))
		t.Views().Main().
			Title(Equals("File content")).
			Content(Equals("1 top"))

		// A modified file shows its new content, not a diff.
		t.Views().CommitFiles().
			NavigateToLine(Equals("    M x.txt"))
		t.Views().Main().
			Title(Equals("File content")).
			Content(Equals("1 x changed")).
			Content(DoesNotContain("@@"))

		// A deleted file shows what it was before the commit.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  D gone.txt"))
		t.Views().Main().
			Title(Equals("File content before this commit (deleted)")).
			Content(Equals("1 gone"))

		// A binary file shows a note instead of its bytes.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  A bin.dat"))
		t.Views().Main().
			Title(Equals("File content")).
			Content(Equals("Binary file, not shown"))

		// A folder shows the commit's changes under it, as the changed-files listing does.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  ▼ a"))
		t.Views().Main().
			Title(Equals("Patch")).
			Content(Contains("+x changed"))

		// 0 focuses the main view on the content, which can be scrolled, and Escape returns.
		t.Views().CommitFiles().
			NavigateToLine(Equals("    M x.txt")).
			Press(config.Keybinding{"0"})
		t.Views().Main().
			IsFocused().
			Content(Equals("1 x changed"))
		t.Views().Main().PressEscape()
		t.Views().CommitFiles().IsFocused()

		// Escape goes back to Commits; v d lists only the files the commit changed, and a
		// modified file there still shows its diff.
		t.Views().CommitFiles().PressEscape()
		t.Views().Commits().
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"d"})

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("Diff files")).
			NavigateToLine(Equals("    M x.txt"))
		t.Views().Main().
			Title(Equals("Patch")).
			Content(Contains("+x changed"))
	},
})
