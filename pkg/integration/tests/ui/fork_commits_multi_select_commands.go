package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCommitsMultiSelectCommands = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "While Commits holds an explicit set, commands that act on the cursor commit or a range are disabled with 'Clear the selection first (Escape)', set-aware commands still work, and after Escape a lone Shift range drops again",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.Gui.ShowRootItemInFileTree = true
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		// Newest first in the list: foxtrot, echo, delta, charlie, bravo, alpha, base.
		shell.CreateFileAndAdd("base.txt", "base\n").Commit("base").
			CreateFileAndAdd("a.txt", "a\n").Commit("alpha").
			CreateFileAndAdd("b.txt", "b\n").Commit("bravo").
			CreateFileAndAdd("c.txt", "c\n").Commit("charlie").
			CreateFileAndAdd("d.txt", "d\n").Commit("delta").
			CreateFileAndAdd("e.txt", "e\n").Commit("echo").
			CreateFileAndAdd("f.txt", "f\n").Commit("foxtrot")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		space := config.Keybinding{"<space>"}
		down := config.Keybinding{"j"}
		shiftDown := config.Keybinding{"J"}
		esc := config.Keybinding{"<esc>"}
		reword := config.Keybinding{"<f2>"}
		drop := config.Keybinding{"<delete>"}
		disabled := Equals("Disabled: Clear the selection first (Escape)")

		// A non-contiguous set: foxtrot and charlie, with delta between them.
		commits.NavigateToLine(Contains("foxtrot")).Press(space)
		commits.Press(down).Press(down).Press(down).Press(space)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("charlie"))

		// Commands on the cursor commit or on a range are disabled, and the set stays.
		commits.Press(reword)
		t.ExpectToast(disabled)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("charlie"))
		commits.LineCount(EqualsInt(7))

		commits.Press(drop)
		t.ExpectToast(disabled)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("charlie"))
		commits.LineCount(EqualsInt(7))

		// Commands bound outside the Commits controllers are guarded too: g b, y i.
		commits.Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"b"})
		t.ExpectToast(disabled)
		commits.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"i"})
		t.ExpectToast(disabled)
		commits.ExplicitSelection(Contains("foxtrot"), Contains("charlie"))

		// Set-aware commands work on the set: c y marks both commits.
		commits.Press(config.Keybinding{"c"})
		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"y"})
		t.Views().Information().Content(MatchesRegexp(`(^|[^0-9])2 commits? copied`))
		commits.ExplicitSelection(Contains("foxtrot"), Contains("charlie"))

		// Escape clears the set and keeps the marks. The cursor is on charlie.
		commits.Press(esc)
		commits.ExplicitSelection()
		t.Views().Information().Content(MatchesRegexp(`(^|[^0-9])2 commits? copied`))

		// With the set gone, a lone Shift range is a range again: Delete drops charlie
		// and bravo, as upstream does.
		commits.Press(shiftDown)
		commits.ExplicitSelection(Contains("charlie"), Contains("bravo"))
		commits.Press(drop)
		t.ExpectPopup().Confirmation().
			Title(Equals("Drop commit")).
			Content(Contains("Are you sure you want to drop the selected commit(s)?")).
			Confirm()
		commits.Lines(
			Contains("foxtrot"),
			Contains("echo"),
			Contains("delta"),
			Contains("alpha"),
			Contains("base"),
		)
	},
})
