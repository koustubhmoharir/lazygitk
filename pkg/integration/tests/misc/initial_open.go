package misc

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var InitialOpen = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Confirms a popup appears on first opening Lazygit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().DisableStartupPopups = false
	},
	SetupRepo: func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.ExpectPopup().Confirmation().
			Title(Equals("")).
			// Fork: the popup names the fork and keeps the note on thanking lazygit's author.
			Content(Contains("lazygitk is a fork of lazygit").Contains("donate button")).
			Confirm()

		t.Views().Files().IsFocused()
	},
})
