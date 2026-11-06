package ui

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterScopeClearBeyondLimit = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Escape clears a w b scope on a branch older than the first 300 commits: the log loads further and the cursor stays on the branch's commit",
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
		// old is at base, the root of the history. base is more than 300 commits below
		// master's tip, so the limited log does not load it.
		shell.CreateFileAndAdd("base.txt", "base").
			Commit("base").
			NewBranch("old").
			Checkout("master")
		for i := range 305 {
			shell.EmptyCommit(fmt.Sprintf("c%03d", i))
		}
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// w b on old applies its scope directly: old has no tracking branch and no other
		// branch points at base, so the scope is base alone.
		t.Views().Branches().Focus().NavigateToLine(Contains("old"))
		t.Views().Branches().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"b"})

		commits := t.Views().Commits()
		commits.IsFocused().
			LineCount(EqualsInt(1)).
			SelectedLines(Contains("base"))

		// Escape clears the scope. The unfiltered log is reloaded beyond its first 300
		// commits, so base is loaded, and the cursor stays on it.
		commits.Press(config.Keybinding{"<esc>"})
		commits.IsFocused().
			LineCount(EqualsInt(306)).
			SelectedLines(Contains("base"))
	},
})
