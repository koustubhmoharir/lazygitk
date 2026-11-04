package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var MergeToolFromConflictView = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "d m in the conflicts view resolves the file being edited and returns focus to Files",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shared.CreateMergeConflictFiles(shell)
		shell.RunCommand([]string{"git", "config", "--local", "merge.tool", "fake"})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.cmd", `printf 'resolved by tool\n' > "$MERGED"`})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.trustExitCode", "true"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Common().PretendMergeOrRebaseStartedInLazygit()

		t.Views().Files().
			IsFocused().
			NavigateToLine(Contains("file1")).
			Press(config.Keybinding{"v"})

		t.Views().MergeConflicts().
			IsFocused().
			Press(config.Keybinding{"d"})

		t.GlobalPress(config.Keybinding{"m"})

		t.Views().Files().
			IsFocused().
			Lines(
				Equals("▼ /"),
				Equals("  M  file1").IsSelected(),
				Equals("  UU file2"),
			)
		t.FileSystem().FileContent("file1", Equals("resolved by tool\n"))
	},
})
