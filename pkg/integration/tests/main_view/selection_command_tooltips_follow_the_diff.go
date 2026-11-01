package main_view

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SelectionCommandTooltipsFollowTheDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The tooltips of the selection commands describe what they do over the diff they are offered on",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "one\ntwo\nthree\n")
		shell.Commit("one")

		shell.UpdateFile("file1", "one\nTWO\nthree\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Over the working tree's diff the two keys act on the index.
		t.Views().Files().
			Focus().
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			Press(keys.Universal.OptionMenu)

		// The main view's own copy of Discard is the first match: the other panes' copies follow it.
		// The palette is filtered rather than scrolled to each item, which is far down it.
		palette := t.ExpectPopup().Menu().Title(Equals("Keybindings"))
		palette.Filter("Stage")
		t.Views().Menu().SelectedLine(Contains("<space> Stage"))
		palette.Tooltip(Equals("Toggle selection staged / unstaged."))
		// Escape clears the filter, and a second Escape closes the palette.
		palette.Cancel()
		palette.Filter("Discard")
		palette.Tooltip(Contains("discard the change using `git reset`"))
		palette.Cancel()
		palette.Cancel()

		t.Views().Main().PressEscape()

		// Over a commit's diff they build a custom patch and rewrite the commit, so
		// the index wording would be wrong; taking lines out of a commit is worth a
		// warning of its own.
		t.Views().Commits().
			Focus().
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			Press(keys.Universal.OptionMenu)

		palette = t.ExpectPopup().Menu().Title(Equals("Keybindings"))
		palette.Filter("Toggle lines in patch")
		t.Views().Menu().SelectedLine(Contains("Toggle lines in patch"))
		palette.Tooltip(Equals(""))
		palette.Cancel()
		palette.Filter("Remove lines from commit")
		t.Views().Menu().SelectedLine(Contains("Remove lines from commit"))
		palette.Tooltip(Contains("runs an interactive rebase in the background"))
		palette.Cancel()
		palette.Cancel()
	},
})
