package ui

import (
	"os"
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPaletteUsageCorrupt = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "An unreadable usage counts file is removed at startup and the ? palette keeps the default order",
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

		// Simulate a previous session that left an unparseable counts file.
		dir := appConfig.GetUserConfigDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "command_usage.json"), []byte("not json"), 0o644); err != nil {
			panic(err)
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// The corrupt file is removed rather than kept, so it cannot be read again.
		path, err := config.CommandUsageFilePath()
		if err != nil {
			t.Fail(err.Error())
		}
		t.FileSystem().PathNotPresent(path)

		// With no counts, the palette keeps declaration order: the first local
		// command in Commits comes first, and the Navigation section stays at the bottom.
		t.Views().Commits().Focus().Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).
			TopLines(Contains("Copy abbreviated commit hash to clipboard")).
			Content(Contains("─── Navigation")).
			Cancel()
	},
})
