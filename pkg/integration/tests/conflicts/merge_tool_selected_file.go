package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var MergeToolSelectedFile = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "d m opens the merge tool for the selected conflicted file only, without a backup file",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shared.CreateMergeConflictFiles(shell)
		// The tool writes a resolution to $MERGED and exits 0, so Git stages
		// the file. The tool is local, so the user's global config is untouched.
		shell.RunCommand([]string{"git", "config", "--local", "merge.tool", "fake"})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.cmd", `printf 'resolved by tool\n' > "$MERGED"`})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.trustExitCode", "true"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Common().PretendMergeOrRebaseStartedInLazygit()

		t.Views().Files().
			IsFocused().
			Lines(
				Equals("▼ /"),
				Equals("  UU file1"),
				Equals("  UU file2"),
			).
			NavigateToLine(Contains("file1")).
			Press(config.Keybinding{"d"})

		t.GlobalPress(config.Keybinding{"m"})

		// No menu opened: Files keeps focus, and only file1 was resolved.
		t.Views().Files().
			IsFocused().
			Lines(
				Equals("▼ /"),
				Equals("  M  file1").IsSelected(),
				Equals("  UU file2"),
			)

		t.FileSystem().FileContent("file1", Equals("resolved by tool\n"))
		// Git's own backup is removed, since mergetool.keepBackup is unset.
		t.FileSystem().PathNotPresent("file1.orig")
		t.FileSystem().PathNotPresent("file2.orig")
	},
})
