package ui

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShowInCommitsBeyondLoaded = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "w v on a branch whose tip is older than the first 300 commits loads the log further and selects the tip",
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
		// old is at base, which is more than 300 commits below master's tip, so the
		// limited log does not load it.
		shell.CreateFileAndAdd("base.txt", "base").
			Commit("base").
			NewBranch("old").
			Checkout("master")
		for i := range 305 {
			shell.EmptyCommit(fmt.Sprintf("c%03d", i))
		}
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().Focus().NavigateToLine(Contains("old"))
		t.Views().Branches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"v"})

		// The log now holds all 306 commits, and base is the only selected one.
		commits := t.Views().Commits()
		commits.IsFocused().
			LineCount(EqualsInt(306)).
			SelectedLines(Contains("base"))
		t.ExpectScreenMode(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	},
})
