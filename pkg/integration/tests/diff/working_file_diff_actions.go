package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var WorkingFileDiffActions = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Compare a working file in the Files panel with HEAD, the index, or a pasted commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(appConfig *config.AppConfig) {
		// Shared integration config can carry legacy alternate bindings from an
		// earlier scenario; exercise the real constructor defaults for this one.
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("sample.txt", "before\n")
		shell.Commit("before")
		shell.UpdateFileAndAdd("sample.txt", "head\n")
		shell.Commit("head")
		// HEAD is "head", the index is "staged" and the working tree is "working".
		shell.UpdateFileAndAdd("sample.txt", "staged\n")
		shell.UpdateFile("sample.txt", "working\n")

		// Record the bytes of the two endpoints so the scenario verifies which
		// concrete versions Git handed to the configured tool.
		shell.CreateFile(".git/fork-diff-tool", `#!/bin/sh
set -eu
git_dir=$(git rev-parse --absolute-git-dir)
cat "$1" > "$git_dir/diff-left"
cat "$2" > "$git_dir/diff-right"
exit 0
`)
		shell.MakeExecutable(".git/fork-diff-tool")
		shell.SetConfig("diff.tool", "fork-test")
		shell.RunCommand([]string{
			"git", "config", "--local", "difftool.fork-test.cmd",
			`"$(git rev-parse --absolute-git-dir)/fork-diff-tool" "$LOCAL" "$REMOTE"`,
		})
		shell.SetConfig("difftool.prompt", "false")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// The right side is always the working file, which has the same content
		// in every comparison here.
		assertDiffInputs := func(left string) {
			t.FileSystem().
				FileContent(".git/diff-left", Equals(left)).
				FileContent(".git/diff-right", Equals("working\n"))
		}
		beforeHash := t.Git().GetCommitHash("HEAD~1")

		t.Views().Files().
			Focus().
			NavigateToLine(Contains("sample.txt"))

		// Each action is a path route, so the second key is pressed through the
		// global prefix menu like the other fork diff routes.
		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.Views().Files().IsFocused()
		assertDiffInputs("head\n")

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"s"})
		t.Views().Files().IsFocused()
		assertDiffInputs("staged\n")

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"c"})
		t.ExpectPopup().Prompt().
			Title(Equals("Compare commit hash (left) with working file (right)")).
			Type(beforeHash[:7]).
			Confirm()
		t.Views().Files().IsFocused()
		assertDiffInputs("before\n")

		// Sticky diff mode must not change the Files comparisons.
		t.Views().Files().Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Filter("View diffing options").
			Confirm()
		t.ExpectPopup().Menu().
			Title(Equals("Diffing")).
			Select(Contains("Enter ref")).
			Confirm()
		t.ExpectPopup().Prompt().
			Title(Equals("Enter ref:")).
			Type("HEAD~1").
			Confirm()

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"s"})
		t.Views().Files().IsFocused()
		assertDiffInputs("staged\n")

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.Views().Files().IsFocused()
		assertDiffInputs("head\n")

		// The comparisons never touch the repository or the working file.
		t.FileSystem().FileContent("sample.txt", Equals("working\n"))
		t.Shell().RunShellCommand("git show :sample.txt > .git/diff-index-content")
		t.FileSystem().FileContent(".git/diff-index-content", Equals("staged\n"))
		t.Git().CurrentBranchName("master")
	},
})
