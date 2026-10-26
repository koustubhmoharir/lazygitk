package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCopyOsc52 = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A copy with no os.copyToClipboardCmd writes the text through OSC 52 to the terminal clipboard",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the y routes come from the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
		// Left empty on purpose: copies must use the OSC 52 default.
		loaded.OS.CopyToClipboardCmd = ""
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("first").
			NewBranch("feature")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		branches := t.Views().Branches().
			Focus().
			NavigateToLine(Contains("feature"))

		branches.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"n"})
		t.ExpectToast(Equals("'feature' copied to clipboard"))
		t.ExpectClipboard(Equals("feature"))
	},
})
