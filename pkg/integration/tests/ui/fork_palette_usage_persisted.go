package ui

import (
	"os"
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPaletteUsagePersisted = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Usage counts persisted by an earlier session are kept, and new uses are added to them",
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

		// Simulate a previous session by writing the counts file before the GUI opens it.
		dir := appConfig.GetUserConfigDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
		counts := `{"version":1,"counts":{"app.refresh":5}}`
		if err := os.WriteFile(filepath.Join(dir, "command_usage.json"), []byte(counts), 0o644); err != nil {
			panic(err)
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		countsPath, err := config.CommandUsageFilePath()
		if err != nil {
			t.Fail(err.Error())
		}
		t.FileSystem().FileContent(countsPath, MatchesRegexp(`"app\.refresh":\s*5`))

		// A shortcut use subtracts one from the persisted count.
		t.Views().Files().Focus().Press(config.Keybinding{"<f5>"})
		t.FileSystem().FileContentEventually(countsPath, MatchesRegexp(`"app\.refresh":\s*4`))
	},
})
