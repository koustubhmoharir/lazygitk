package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCopyCommitKeys = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "The flat y copy menu copies full and abbreviated hashes and the subject from the commits and reflog panels",
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
		loaded.OS.CopyToClipboardCmd = "printf '%s' {{text}} > clipboard"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("first")
		shell.EmptyCommitWithBody("My Subject", "My commit body")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		head := t.Git().GetCommitHash("HEAD")
		// Default truncateCopiedCommitHashesTo.
		abbreviated := head[:12]

		commits := t.Views().Commits().
			Focus().
			Lines(
				Contains("My Subject").IsSelected(),
				Contains("first"),
			)

		// The y menu lists the flat entries and no nested attribute menu.
		commits.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Content(
			Contains("Copy commit hash").
				Contains("Copy abbreviated commit hash").
				Contains("Copy commit subject").
				DoesNotContain("Copy commit attribute"),
		)
		t.Views().Menu().Press(config.Keybinding{"h"})
		t.ExpectToast(Equals("'" + head + "' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(head))

		commits.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"i"})
		t.ExpectToast(Equals("'" + abbreviated + "' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(abbreviated))

		commits.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectToast(Equals("Commit subject copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("My Subject"))

		// The reflog panel uses the same second keys for the same hashes.
		reflog := t.Views().ReflogCommits().
			Focus().
			ContainsLines(
				Contains("commit: My Subject").IsSelected(),
			)

		reflog.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"h"})
		t.ExpectToast(Equals("'" + head + "' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(head))

		reflog.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"i"})
		t.ExpectToast(Equals("'" + abbreviated + "' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(abbreviated))
	},
})
