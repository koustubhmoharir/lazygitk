package config

import (
	appconfig "github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkIgnoredKeyConfig = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Key settings in the config file are ignored: the fork defaults apply and custom commands stay in the palette without keys",
	ExtraEnvVars: map[string]string{
		// Turn off the harness switch that honours key configuration for this
		// scenario only, so the user-facing behaviour is tested.
		appconfig.HonourKeyConfigEnvVar: "",
	},
	SetupConfig: func(appConfig *appconfig.AppConfig) {
		loaded := appConfig.GetUserConfig()
		// A legacy key the fork must ignore, and a custom command with a key.
		loaded.Keybinding.Universal.Quit = appconfig.Keybinding{"<f12>"}
		loaded.CustomCommands = []appconfig.CustomCommand{
			{
				Key:         appconfig.Keybinding{"Z"},
				Context:     "files",
				Command:     "touch myfile",
				Description: "Fork ignored key custom command",
			},
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one")
	},
	Run: func(t *TestDriver, keys appconfig.KeybindingConfig) {
		files := t.Views().Files().Focus()

		// The ignored legacy quit key does nothing: the app keeps running, and
		// the default ? palette still opens.
		files.Press(appconfig.Keybinding{"<f12>"}).IsFocused()
		files.Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).
			Filter("Fork ignored key custom command")
		// The custom command is listed in the palette, without a shortcut.
		t.Views().Menu().Content(Contains("Fork ignored key custom command"))
		// The first cancel closes the filter; the second closes the palette.
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().MenuFilter().IsInvisible()
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().Menu().IsInvisible()
		files.IsFocused()

		// Its ignored key does not run the command.
		files.Press(appconfig.Keybinding{"Z"}).IsFocused()
		files.Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().Menu().IsInvisible()
		files.Content(DoesNotContain("myfile"))
	},
})
