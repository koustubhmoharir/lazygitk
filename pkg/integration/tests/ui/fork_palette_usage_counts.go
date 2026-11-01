package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPaletteUsageCounts = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Palette and shortcut uses are counted, the reset command clears the counts, and none of this reorders the ? palette",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to get the real default keymap, as the default
		// keymap scenarios do.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		files := t.Views().Files()
		openPalette := func() *MenuDriver {
			files.Press(keys.Universal.OptionMenu)
			return t.ExpectPopup().Menu().Title(Equals("Keybindings"))
		}
		countsPath, err := config.CommandUsageFilePath()
		if err != nil {
			t.Fail(err.Error())
		}
		toggleWhitespace := `"diff\.toggle-whitespace":\s*1`
		refresh := `"app\.refresh":\s*-1`

		// Each accepted palette run adds one use.
		openPalette().Filter("Toggle whitespace").Confirm()
		t.FileSystem().FileContentEventually(countsPath, MatchesRegexp(toggleWhitespace))

		// A shortcut subtracts one.
		files.Press(config.Keybinding{"<f5>"})
		t.FileSystem().FileContentEventually(countsPath, MatchesRegexp(refresh))

		// The counts do not reorder the palette: the focused pane still comes first.
		openPalette().
			TopLines(Contains("─── Files and global")).
			Cancel()

		// Cancelling the reset keeps the counts.
		openPalette().Filter("Reset command usage counts").Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Reset command usage counts")).
			Content(Contains("Reset all command usage counts?")).
			Cancel()
		t.FileSystem().FileContentEventually(countsPath, MatchesRegexp(toggleWhitespace))

		// Accepting the reset clears the counts.
		openPalette().Filter("Reset command usage counts").Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Reset command usage counts")).
			Content(Contains("Reset all command usage counts?")).
			Confirm()
		t.FileSystem().FileContentEventually(countsPath, DoesNotContain("diff.toggle-whitespace").DoesNotContain("app.refresh"))
	},
})
