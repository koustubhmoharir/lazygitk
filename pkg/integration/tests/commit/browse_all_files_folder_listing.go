package commit

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesFolderListing = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In the full listing of a commit's files, a folder shows its entries, with the status of the files the commit changed, and a folder's diff stays in the changed-files listing",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("README.md", "readme\n")
		shell.CreateDir("src/gui")
		shell.CreateDir("src/config")
		shell.CreateFileAndAdd("src/gui/gui.go", "package gui\n")
		shell.CreateFileAndAdd("src/config/config.go", "package config\n")
		shell.CreateFileAndAdd("src/main.go", "package main\n")
		shell.CreateFileAndAdd("src/old.go", "old\n")
		shell.CreateFileAndAdd("src/util.go", strings.Repeat("u", 1499)+"\n")
		shell.Commit("first")
		shell.UpdateFileAndAdd("src/main.go", "package main\n// changed\n")
		shell.CreateFileAndAdd("src/new.go", "new\n")
		shell.DeleteFileAndAdd("src/old.go")
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

		// The root item lists the commit's top-level entries.
		t.Views().CommitFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("    README.md"),
				Equals("  ▶ src"),
			)
		t.Views().Main().
			Title(Equals("Folder: / (2 entries)")).
			TopLines(
				Equals("  src/"),
				MatchesRegexp(`^  README\.md +7 B$`),
			)

		// Expanding src lists its entries, with the commit's changes; the listing starts
		// collapsed, so src is expanded first.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  ▶ src")).
			Press(keys.Universal.GoInto).
			Lines(
				Equals("▼ /"),
				Equals("    README.md"),
				Equals("  ▼ src"),
				Equals("    ▶ config"),
				Equals("    ▶ gui"),
				Equals("    M main.go"),
				Equals("    A new.go"),
				Equals("    D old.go"),
				Equals("      util.go"),
			)

		// A folder lists its subfolders first, then its files with their sizes; the
		// deleted file keeps its size from before the commit.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  ▼ src"))
		t.Views().Main().
			Title(Equals("Folder: src/ (6 entries)")).
			TopLines(
				Equals("  config/"),
				Equals("  gui/"),
				MatchesRegexp(`^M main\.go +24 B$`),
				MatchesRegexp(`^A new\.go +4 B$`),
				MatchesRegexp(`^D old\.go +4 B$`),
				MatchesRegexp(`^  util\.go +1\.5 KB$`),
			)

		// A subfolder lists its own entries.
		t.Views().CommitFiles().
			NavigateToLine(Equals("    ▶ gui"))
		t.Views().Main().
			Title(Equals("Folder: src/gui/ (1 entry)")).
			TopLines(MatchesRegexp(`^  gui\.go +12 B$`))

		// A folder shows no diff or file for the renderer keys to cycle, so they are disabled.
		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderers)
		t.ExpectToast(Equals("Disabled: Folders and submodules show no diff or file to render"))

		// Escape goes back to Commits; v d lists only the changed files, and the folder
		// there still shows its diff.
		t.Views().CommitFiles().PressEscape()
		t.Views().Commits().
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"d"})

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("Diff files")).
			NavigateToLine(Contains("src"))
		t.Views().Main().
			Title(Equals("Patch")).
			Content(Contains("+// changed"))
	},
})
