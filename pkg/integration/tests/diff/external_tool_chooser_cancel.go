package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ExternalToolChooserCancel = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Escape in the diff tool chooser, at either step, launches nothing and saves nothing",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("sample.txt", "before\n")
		shell.Commit("before")
		shell.UpdateFile("sample.txt", "working\n")
		shell.CreateFile(".git/fake-diff-tool", `#!/bin/sh
set -eu
git_dir=$(git rev-parse --absolute-git-dir)
printf launched > "$git_dir/fake-tool-ran"
`)
		shell.MakeExecutable(".git/fake-diff-tool")
		shell.RunCommand([]string{
			"git", "config", "--local", "difftool.fake.cmd",
			`"$(git rev-parse --absolute-git-dir)/fake-diff-tool"`,
		})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			NavigateToLine(Contains("sample.txt"))

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.ExpectPopup().Menu().
			Title(Equals("No diff tool is configured")).
			Cancel()
		t.Views().Files().IsFocused()
		t.FileSystem().PathNotPresent(".git/fake-tool-ran")

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.ExpectPopup().Menu().
			Title(Equals("No diff tool is configured")).
			Select(Contains("fake")).
			Confirm()
		t.ExpectPopup().Menu().
			Title(Equals("Use fake")).
			Cancel()
		t.Views().Files().IsFocused()
		t.FileSystem().PathNotPresent(".git/fake-tool-ran")

		t.Shell().RunShellCommand("git config --global --get diff.tool > .git/global-diff-tool || true")
		t.FileSystem().FileContent(".git/global-diff-tool", Equals(""))
	},
})
