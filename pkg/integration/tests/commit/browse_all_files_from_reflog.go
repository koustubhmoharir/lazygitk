package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesFromReflog = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Opens the all-files and changed-files listings of a reflog entry with v f and v d",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("top.txt", "top\n")
		shell.CreateFileAndAdd("other.txt", "other\n")
		shell.Commit("first")
		shell.UpdateFileAndAdd("top.txt", "top changed\n")
		shell.CreateFileAndAdd("new.txt", "new\n")
		shell.Commit("second")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().ReflogCommits().
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
				Equals("  A new.txt"),
				Equals("    other.txt"),
				Equals("  M top.txt"),
			)

		t.Views().CommitFiles().PressEscape()
		t.Views().ReflogCommits().
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"d"})

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("Diff files")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  A new.txt"),
				Equals("  M top.txt"),
			)
	},
})
