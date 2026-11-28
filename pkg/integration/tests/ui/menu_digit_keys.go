package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var MenuDigitKeys = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A menu item with no key of its own is run by its digit, shown with an arrow, without navigating to it",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("topic").EmptyCommit("topic-tip")
		shell.Checkout("master").EmptyCommit("master-tip")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("topic")).
			Press(keys.Universal.Push)

		// The push menu's items have no keys of their own, so they are numbered in
		// order: Push this branch 1, Choose remote 2, Choose destination 3, and the
		// set-upstream checkbox 4. For a branch without tracking config, set-upstream
		// starts ticked.
		menu := t.ExpectPopup().Menu().Title(Equals("Push selected branch"))
		menu.Content(Contains("1 →     Push this branch"))
		menu.Content(Contains("4 → [✓] Set upstream for selected local branch"))

		// Pressing the digit runs the item without moving the selection to it. The
		// menu reopens with the checkbox flipped.
		t.GlobalPress(config.Keybinding{"4"})
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("4 → [ ] Set upstream for selected local branch")).
			Cancel()
	},
})
