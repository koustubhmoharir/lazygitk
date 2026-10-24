package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPrefixOverrides = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Prefix accelerators override configured menu controls while ordinary menus retain them",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Keybinding.Universal.PrevItem = config.Keybinding{"r"}
		loaded.Keybinding.Universal.ConfirmMenu = config.Keybinding{"e"}
		loaded.Keybinding.Universal.Return = config.Keybinding{"x"}
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("first commit").EmptyCommit("second commit")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		files := t.Views().Files().Focus()

		// The e accelerator enters the nested group even though e is the normal
		// menu confirmation key. Physical Escape backs up one level at a time.
		files.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"e"})
		t.ExpectPopup().Menu().Title(Equals("g e"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		files.IsFocused()

		// Physical arrows and Enter navigate the prefix list to the g e group.
		files.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().NavigateToLine(Contains("Commit changes using git editor")).
			Press(config.Keybinding{"<down>"}).Press(config.Keybinding{"<up>"}).
			SelectedLine(Contains("Commit changes using git editor")).Press(config.Keybinding{"<enter>"})
		t.ExpectPopup().Menu().Title(Equals("g e"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		files.IsFocused()

		// An invalid continuation that would be a global quit key is consumed by
		// the prefix menu and does not run an operation or leave the menu.
		head := t.Git().GetCommitHash("HEAD")
		files.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"q"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"<esc>"})
		files.IsFocused()
		t.Views().Confirmation().IsInvisible()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("invalid prefix continuation changed HEAD")
		}

		// r is the configured previous-item key, but g r b must use r as the
		// prefix accelerator and mark the selected commit as the rebase base.
		commits := t.Views().Commits().Focus().NavigateToLine(Contains("second commit"))
		commits.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"r"})
		t.ExpectPopup().Menu().Title(Equals("g r"))
		t.Views().Menu().Press(config.Keybinding{"b"})
		commits.IsFocused().SelectedLine(Contains("second commit"))
		t.Views().Information().Content(Contains("Marked a base commit for rebase"))
		t.Views().Confirmation().IsInvisible()

		// Ordinary command-palette menus continue to use configured e and x.
		commits.Press(config.Keybinding{"?"})
		palette := t.ExpectPopup().Menu().Title(Equals("Keybindings"))
		palette.Select(Contains("Focus").Contains("Files")).Confirm()
		files.IsFocused()
		files.Press(config.Keybinding{"?"})
		t.ExpectPopup().Menu().Title(Equals("Keybindings"))
		t.Views().Menu().Press(config.Keybinding{"x"})
		t.Views().Menu().IsInvisible()
		files.IsFocused()
	},
})
