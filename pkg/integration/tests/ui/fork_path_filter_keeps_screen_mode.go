package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPathFilterKeepsScreenMode = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A path filter never changes the screen mode: a normal screen stays normal, and a half screen chosen with + survives clearing the filter",
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
		shell.CreateFileAndAdd("filterFile", "1").
			Commit("c1").
			CreateFileAndAdd("otherFile", "1").
			Commit("c2").
			UpdateFileAndAdd("filterFile", "2").
			Commit("c3")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		filterByPath := func() {
			commits.Press(config.Keybinding{"w"})
			t.ExpectPopup().Menu().Title(Equals("w"))
			t.Views().Menu().Press(config.Keybinding{"p"})
			t.ExpectPopup().Prompt().Title(Equals("Enter path:")).Type("filterFile").Confirm()
			commits.IsFocused().Lines(Contains("c3"), Contains("c1"))
		}

		// Applying a path filter to a normal screen leaves it normal.
		filterByPath()
		t.ExpectScreenMode(types.SCREEN_NORMAL, types.SCREEN_AREA_SIDE)
		commits.Press(config.Keybinding{"<esc>"})
		commits.LineCount(EqualsInt(3))
		t.ExpectScreenMode(types.SCREEN_NORMAL, types.SCREEN_AREA_SIDE)

		// A half screen chosen with + stays through the filter and its clearing.
		commits.Press(keys.Universal.NextScreenMode)
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
		filterByPath()
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
		commits.Press(config.Keybinding{"<esc>"})
		commits.LineCount(EqualsInt(3))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	},
})
