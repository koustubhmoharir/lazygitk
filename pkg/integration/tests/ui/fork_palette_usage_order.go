package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPaletteUsageOrder = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "The ? palette orders commands by usage, tags their scope, and resets the counts",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to get the real default keymap, as the default
		// keymap scenarios do. The shared fixture selects the static palette, so
		// this scenario sets the usage order explicitly.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.Gui.CommandPaletteOrder = "usage"
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
		// A global, reversible command that opens no popup.
		toggleWhitespace := "Toggle whitespace"

		// Unused commands keep declaration order, and the Navigation section stays at the bottom.
		openPalette().
			TopLines(DoesNotContain(toggleWhitespace)).
			Content(DoesNotContain("─── Local").DoesNotContain("─── Global")).
			Content(MatchesRegexp("(?s)" + toggleWhitespace + ".*global.*─── Navigation")).
			Cancel()

		// Each accepted palette run adds one use.
		openPalette().Filter(toggleWhitespace).Confirm()
		openPalette().Filter(toggleWhitespace).Confirm()

		// The most used command is first and shows its global scope.
		openPalette().
			TopLines(Contains(toggleWhitespace).Contains("global")).
			Content(Contains("─── Navigation")).
			Cancel()

		// Shortcut uses subtract one each. Three presses of refresh (f5) put it below never-used
		// commands such as the shell command prompt.
		files.Press(config.Keybinding{"<f5>"}).Press(config.Keybinding{"<f5>"}).Press(config.Keybinding{"<f5>"})
		openPalette().
			Content(MatchesRegexp("(?s)" + toggleWhitespace + ".*Execute shell command.*Refresh")).
			Cancel()

		// A palette run that opens a prompt and is cancelled is not a use. If the
		// shell command had been counted, it would be the second row.
		openPalette().Filter("Execute shell command").Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Shell command:")).Cancel()
		openPalette().
			TopLines(Contains(toggleWhitespace), DoesNotContain("Execute shell command")).
			Cancel()

		// Cancelling the reset confirmation keeps the counts.
		openPalette().Filter("Reset command usage counts").Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Reset command usage counts")).
			Content(Contains("Reset all command usage counts?")).
			Cancel()
		openPalette().TopLines(Contains(toggleWhitespace)).Cancel()

		// Accepting the reset restores declaration order.
		openPalette().Filter("Reset command usage counts").Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Reset command usage counts")).
			Content(Contains("Reset all command usage counts?")).
			Confirm()
		openPalette().
			TopLines(DoesNotContain(toggleWhitespace)).
			Content(MatchesRegexp("(?s)Execute shell command.*" + toggleWhitespace + ".*─── Navigation")).
			Cancel()
	},
})
