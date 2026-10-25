package ui

import (
	"os"
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPaletteUsagePersisted = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Usage counts persisted by an earlier session order the ? palette on startup",
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

		// Simulate a previous session by writing the counts file before the GUI opens it.
		// Refresh is a normally-late global command, and copying the abbreviated hash is
		// normally the first local command in Commits. The negative count pushes it down.
		dir := appConfig.GetUserConfigDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
		counts := `{"version":1,"counts":{"app.refresh":5,"clipboard.copy-abbreviated-commit-hash":-3}}`
		if err := os.WriteFile(filepath.Join(dir, "command_usage.json"), []byte(counts), 0o644); err != nil {
			panic(err)
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		openPalette := func() *MenuDriver {
			commits.Press(keys.Universal.OptionMenu)
			return t.ExpectPopup().Menu().Title(Equals("Keybindings"))
		}

		// The persisted count moves Refresh to the top, ahead of the local commands.
		openPalette().
			TopLines(Contains("<f5> Refresh").Contains("global")).
			Cancel()

		// The negative count sinks the copy command below the never-used command that
		// was declared after it. Without the persisted file it would be the first row.
		openPalette().
			Content(MatchesRegexp("(?s)Reset copied \\(cherry-picked\\) commits selection.*Copy abbreviated commit hash to clipboard")).
			TopLines(DoesNotContain("Copy abbreviated commit hash to clipboard")).
			Cancel()
	},
})
