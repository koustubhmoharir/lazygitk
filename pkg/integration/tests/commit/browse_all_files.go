package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browses every file of a commit's tree with v f, then lists only the changed files with v d",
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

		// v opens the menu of the two listings.
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
			// Folders start collapsed, so only the top-level entries are visible. The
			// unchanged top.txt has a blank status, which keeps its name aligned.
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ a"),
				Equals("  D gone.txt"),
				Equals("  A new.txt"),
				Equals("    top.txt"),
			)

		// Expanding a shows its files; x.txt was changed, so it keeps its M status.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  ▶ a")).
			Press(keys.Universal.GoInto).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ a"),
				Equals("    ▶ b"),
				Equals("    M x.txt"),
				Equals("  D gone.txt"),
				Equals("  A new.txt"),
				Equals("    top.txt"),
			)

		// Escape goes back to Commits; v d lists only the files the commit changed.
		t.Views().CommitFiles().PressEscape()
		t.Views().Commits().
			Press(config.Keybinding{"v"})
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
	},
})
