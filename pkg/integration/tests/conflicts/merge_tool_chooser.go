package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var MergeToolChooser = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With no merge tool configured, d m asks for one; Just this time resolves the file and saves nothing",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shared.CreateMergeConflictFiles(shell)
		// merge.tool is unset, so the only tool is the user-defined one.
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.cmd", `printf 'resolved by tool\n' > "$MERGED"`})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.trustExitCode", "true"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Common().PretendMergeOrRebaseStartedInLazygit()

		t.Views().Files().
			IsFocused().
			NavigateToLine(Contains("file1")).
			Press(config.Keybinding{"d"})

		t.GlobalPress(config.Keybinding{"m"})

		t.ExpectPopup().Menu().
			Title(Equals("No merge tool is configured")).
			Select(Contains("fake")).
			Confirm()
		t.ExpectPopup().Menu().
			Title(Equals("Use fake")).
			Select(Contains("Just this time")).
			Confirm()

		t.Views().Files().
			IsFocused().
			Lines(
				Equals("▼ /"),
				Equals("  M  file1").IsSelected(),
				Equals("  UU file2"),
			)
		t.FileSystem().FileContent("file1", Equals("resolved by tool\n"))

		// Nothing was saved: the global config still has no merge.tool.
		t.Shell().RunShellCommand("git config --global --get merge.tool > .git/global-merge-tool || true")
		t.FileSystem().FileContent(".git/global-merge-tool", Equals(""))
	},
})
