package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultFilesCContinueMerge = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Files c during a merge with a resolved, staged conflict continues the merge and completes it",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the c route comes from the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("file", "base\n").
			Commit("base").
			NewBranch("topic").
			UpdateFileAndAdd("file", "topic\n").
			Commit("topic change").
			Checkout("master").
			UpdateFileAndAdd("file", "main\n").
			Commit("main change").
			RunCommandExpectError([]string{"git", "merge", "--no-edit", "topic"})
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Lines(
				Contains("UU file").IsSelected(),
			)
		t.Views().Status().Content(Contains("(merging)"))

		t.Shell().UpdateFile("file", "resolved\n")
		t.Shell().GitAdd("file")

		// In Files, c names the operation it continues.
		t.Views().Options().Content(Contains("Continue merge"))
		t.Views().Files().Press(config.Keybinding{"c"})

		t.Views().Status().Content(DoesNotContain("merging"))
		t.Views().Commits().
			TopLines(
				Contains("Merge branch 'topic'"),
			)
	},
})
