package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultHintLabels = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Default-profile hints name the effective command keys: the conflict resolve hint shows v, and the command log header shows t l c",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.Gui.ShowRootItemInFileTree = true
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		// A non-textual conflict (one side deletes, the other modifies) is resolved
		// through the dialog that the conflict hint describes.
		shell.RunShellCommand(`echo 1 > foo && echo 1 > bar`)
		shell.RunShellCommand(`git checkout -b base && git add . && git commit -m base`)

		// theirs: delete foo, modify bar
		shell.RunShellCommand(`git checkout -b theirs`)
		shell.RunShellCommand(`git rm foo && echo 2 > bar && git add bar && git commit -m theirs`)

		// ours: modify foo, delete bar
		shell.RunShellCommand(`git checkout base && git checkout -b ours`)
		shell.RunShellCommand(`echo 2 > foo && git add foo && git rm bar && git commit -m ours`)

		shell.RunCommandExpectError([]string{"git", "merge", "theirs"})
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			NavigateToLine(Contains("DU bar"))

		// The conflict hint names the default files-view key for focusing the
		// main view, which is v rather than the legacy Enter/GoInto key.
		t.Views().Main().Content(Contains("Press v to resolve."))

		// The command log header names the options-menu route, t l c, in the
		// default profile.
		t.Views().Extras().Content(Contains("t l c"))
	},
})
