package shared

import (
	"github.com/jesseduffield/lazygit/pkg/config"
)

// UseDefaultKeymap gives a test the real default keymap for the platform, as
// the user sees it. The shared compatibility config carries legacy keybinding
// fields, which would otherwise hide the default paths. Appearance is kept.
// Automatic refresh, fetch and external-change detection stay off, so a test
// sees only what it does.
func UseDefaultKeymap(appConfig *config.AppConfig) {
	loaded := appConfig.GetUserConfig()
	appearance := loaded.Gui
	*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
	loaded.Gui = appearance
	loaded.DisableStartupPopups = true
	loaded.PromptToReturnFromSubprocess = false
	loaded.Git.AutoDetectExternalChanges = false
	loaded.Git.AutoFetch = false
	loaded.Git.AutoRefresh = false
	loaded.Git.Log.ShowGraph = "never"
}
