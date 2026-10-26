package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var WorkingFileDiffUntracked = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Working-file comparisons are disabled with a reason for untracked files and clean tracked files",
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
		shell.CreateFileAndAdd("staged.txt", "before\n")
		shell.Commit("initial")
		// Staged changes only: the file has no unstaged changes to compare.
		shell.UpdateFileAndAdd("staged.txt", "after\n")
		shell.CreateFile("new.txt", "untracked\n")
		shell.SetConfig("difftool.prompt", "false")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			NavigateToLine(Contains("new.txt"))

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.ExpectToast(Equals("Disabled: untracked file: Git has no base version to compare"))
		t.Views().Files().IsFocused()

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"s"})
		t.ExpectToast(Equals("Disabled: untracked file: Git has no base version to compare"))
		t.Views().Files().IsFocused()

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"c"})
		t.ExpectToast(Equals("Disabled: untracked file: Git has no base version to compare"))
		t.Views().Files().IsFocused()

		t.Views().Files().
			NavigateToLine(Contains("staged.txt"))
		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"s"})
		t.ExpectToast(Equals("Disabled: no unstaged changes"))
		t.Views().Files().IsFocused()
	},
})
