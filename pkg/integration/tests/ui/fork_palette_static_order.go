package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPaletteStaticOrder = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "The ? palette keeps its Local, Global and Navigation sections with the static option",
	SetupConfig: func(appConfig *config.AppConfig) {
		appConfig.GetUserConfig().Gui.CommandPaletteOrder = "static"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).
			Content(MatchesRegexp("(?s)─── Local.*─── Global.*─── Navigation")).
			Cancel()
	},
})
