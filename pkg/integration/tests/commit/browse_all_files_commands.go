package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesCommands = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In the full listing, the commands that need a changed file are disabled on an unchanged file and act only on the changed files under a folder",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateDir("a/b")
		shell.CreateFileAndAdd("a/x.txt", "x\n")
		shell.CreateFileAndAdd("a/b/y.txt", "y\n")
		shell.CreateFileAndAdd("top.txt", "top\n")
		shell.Commit("first")
		shell.UpdateFileAndAdd("a/x.txt", "x changed\n")
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
				Equals("    top.txt"),
			)

		// top.txt is unchanged: removing it from the commit and toggling it into the
		// custom patch are crossed out with the reason.
		t.Views().CommitFiles().
			NavigateToLine(Equals("    top.txt")).
			Press(keys.Universal.Remove)
		t.ExpectToast(Equals("Disabled: This file is unchanged in this commit."))

		t.Views().CommitFiles().
			Press(keys.Universal.Select)
		t.ExpectToast(Equals("Disabled: This file is unchanged in this commit."))

		// Expand a: it holds the changed x.txt and the unchanged folder b.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  ▶ a")).
			Press(keys.Universal.GoInto).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ a").IsSelected(),
				Equals("    ▶ b"),
				Equals("    M x.txt"),
				Equals("    top.txt"),
			)

		// Toggling a adds only its changed file to the custom patch. A folder whose
		// changed files are all in the patch is whole, even with unchanged files
		// beside them, so toggling it again removes x.txt rather than adding it again.
		t.Views().CommitFiles().
			Press(keys.Universal.Select)
		t.Views().Secondary().Content(Contains("+x changed"))
		t.Views().CommitFiles().
			Press(keys.Universal.Select)
		t.Views().Secondary().Content(DoesNotContain("x changed"))

		// Folder b holds only unchanged files, so toggling it is crossed out.
		t.Views().CommitFiles().
			NavigateToLine(Equals("    ▶ b")).
			Press(keys.Universal.Select)
		t.ExpectToast(Equals("Disabled: Nothing under this folder changed in this commit."))

		// A folder shows its entries in the main view, not its diff.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  ▼ a"))
		t.Views().Main().
			Title(Equals("Folder: a/ (2 entries)")).
			TopLines(
				Equals("  b/"),
				MatchesRegexp(`^M x\.txt +10 B$`),
			)

		// Restoring the unchanged top.txt from the commit works without an error.
		t.Views().CommitFiles().
			NavigateToLine(Equals("    top.txt")).
			Press(keys.CommitFiles.CheckoutCommitFile).
			IsFocused()
		t.FileSystem().FileContent("top.txt", Equals("top\n"))
	},
})
