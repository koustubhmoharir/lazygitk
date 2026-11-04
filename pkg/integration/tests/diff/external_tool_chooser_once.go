package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ExternalToolChooserOnce = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With no diff tool configured, d w asks for one and Just this time launches it without saving",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("sample.txt", "before\n")
		shell.Commit("before")
		shell.UpdateFile("sample.txt", "working\n")
		// Only a user-defined tool is available, so the chooser lists it. The
		// script records that the tool ran.
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
			Select(Contains("fake")).
			Confirm()
		t.ExpectPopup().Menu().
			Title(Equals("Use fake")).
			Select(Contains("Just this time")).
			Confirm()

		t.FileSystem().PathPresent(".git/fake-tool-ran")
		t.Views().Files().IsFocused()

		// Nothing was saved: the global config still has no diff.tool.
		t.Shell().RunShellCommand("git config --global --get diff.tool > .git/global-diff-tool || true")
		t.FileSystem().FileContent(".git/global-diff-tool", Equals(""))
	},
})
