package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFilterPathKey = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default w p filters the log by the selected file, prefilled in the prompt, and w c clears it; w c and w o are not offered without a filter",
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
			Commit("c3").
			UpdateFileAndAdd("otherFile", "2").
			Commit("c4").
			// An unstaged change keeps filterFile in the Files panel to select.
			UpdateFile("filterFile", "3")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// Without a filter, the w menu offers neither the clear entry nor the
		// filtering menu, which has no key of its own now.
		t.Views().Files().Focus().
			NavigateToLine(Contains("filterFile"))
		t.Views().Files().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w")).
			Content(Contains("Filter by path").DoesNotContain("Clear the log filter").DoesNotContain("View filter options"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})

		// w p from Files opens the path prompt with the selected file in it.
		t.Views().Files().Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"p"})

		t.ExpectPopup().Prompt().
			Title(Equals("Enter path:")).
			InitialText(Equals("filterFile")).
			Confirm()

		// Only the commits that touched filterFile are left: c3 and c1.
		commits := t.Views().Commits().IsFocused()
		commits.Lines(Contains("c3"), Contains("c1")).
			SelectedLines(Contains("c3"))

		// While the filter is active, w offers the clear entry, and w c removes it.
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w")).
			Content(Contains("Clear the log filter"))
		t.Views().Menu().Press(config.Keybinding{"c"})

		commits.Lines(Contains("c4"), Contains("c3"), Contains("c2"), Contains("c1"))
	},
})
