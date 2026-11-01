package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPaletteSections = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "The ? palette lists the focused pane and globals first, then the other panes in layout order, explaining commands that are not available here",
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
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n").Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		openPalette := func() *MenuDriver {
			t.Views().Branches().Focus().Press(keys.Universal.OptionMenu)
			return t.ExpectPopup().Menu().Title(Equals("Keybindings"))
		}

		// The first section is the focused pane with the globals. The other panes
		// follow in layout order, and the focused pane's navigation comes last.
		openPalette().
			TopLines(Contains("─── Local branches and global")).
			Content(MatchesRegexp("(?s)─── Local branches and global.*─── Status.*─── Files.*─── Worktrees.*─── Submodules.*─── Remotes.*─── Remote branches.*─── Tags.*─── Commits.*─── Reflog.*─── Sub-commits.*─── Commit files.*─── Stash.*─── Main panel \\(normal\\).*─── Navigation")).
			Cancel()

		// A global command is in the first section, and not repeated under another pane.
		openPalette().
			Content(MatchesRegexp("(?s)─── Local branches and global.*Toggle whitespace.*─── Status")).
			Cancel()

		// Within a section, single keys come first, then the d, s, n, c, u, y, g, t and w
		// groups, then commands without a key path.
		openPalette().
			Content(MatchesRegexp("(?s)─── Local branches and global.*f Fetch.*s c Checkout.*n b Create.*c p Cherry-pick.*y a Copy commit author.*g f Fast-forward.*t d c View diffing.*w b Show branch.*New branch.*─── Status")).
			Cancel()

		// The other panes are ordered the same way. A command of Commits, which is not
		// available from Branches, is listed under Commits.
		openPalette().
			Content(MatchesRegexp("(?s)─── Commits.*< Move commit up one.*d c Diff with commit.*d w Diff with working tree.*s c Checkout.*g b View bisect options.*g e a Amend commit attribute.*w a Show selected commits.*─── Reflog")).
			Select(Contains("<f2> Reword")).
			Confirm()
		t.ExpectToast(Equals("Disabled: Available in Commits"))
	},
})
