package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShowInCommitsRemoteBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w v on a remote branch from Remote branches shows its commit in Commits, selected, and makes the side area half",
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
		shell.CreateFileAndAdd("base.txt", "base").
			Commit("base").
			CreateFileAndAdd("m1.txt", "m1").
			Commit("m1").
			CreateFileAndAdd("m2.txt", "m2").
			Commit("m2").
			CloneIntoRemote("origin").
			// origin/feat is a remote-tracking ref at m1, with no local branch there.
			RunCommand([]string{"git", "update-ref", "refs/remotes/origin/feat", "HEAD~1"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// v lists the remote's branches, as in the remote branch scope test.
		t.Views().Remotes().Focus().
			Lines(Contains("origin").IsSelected()).
			Press(config.Keybinding{"v"})

		t.Views().RemoteBranches().
			IsFocused().
			NavigateToLine(Contains("feat"))
		t.Views().RemoteBranches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"v"})

		t.Views().Commits().IsFocused().
			Lines(Contains("m2"), Contains("m1"), Contains("base")).
			SelectedLines(Contains("m1"))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	},
})
