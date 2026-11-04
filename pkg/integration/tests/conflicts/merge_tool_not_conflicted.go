package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var MergeToolNotConflicted = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "d m on a file that is not conflicted does nothing: no menu, no tool",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shared.CreateMergeConflictFiles(shell)
		shell.CreateFileAndAdd("plain", "not conflicted\n")
		// If the tool ran, it would leave this marker behind.
		shell.RunCommand([]string{"git", "config", "--local", "merge.tool", "fake"})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.cmd", `touch "$(git rev-parse --absolute-git-dir)/fake-tool-ran"`})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.trustExitCode", "true"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Common().PretendMergeOrRebaseStartedInLazygit()

		// While a merge is in progress Files shows only conflicting files, so
		// clear that filter to get the plain file into view.
		t.Views().Files().
			IsFocused().
			Press(config.Keybinding{"w"})
		t.GlobalPress(config.Keybinding{"s"})
		t.ExpectPopup().Menu().
			Title(Equals("Filtering")).
			Select(Contains("No filter")).
			Confirm()

		t.Views().Files().
			IsFocused().
			NavigateToLine(Contains("plain")).
			Press(config.Keybinding{"d"})

		// m has no entry for a plain file, so the d menu is left as it was.
		t.GlobalPress(config.Keybinding{"m"})
		t.FileSystem().PathNotPresent(".git/fake-tool-ran")

		t.GlobalPress(keys.Universal.Return)
		t.Views().Files().
			IsFocused().
			Lines(
				Equals("▼ /"),
				Equals("  UU file1"),
				Equals("  UU file2"),
				Equals("  A  file3"),
				Equals("  A  plain").IsSelected(),
			)
		t.FileSystem().PathNotPresent(".git/fake-tool-ran")
	},
})
