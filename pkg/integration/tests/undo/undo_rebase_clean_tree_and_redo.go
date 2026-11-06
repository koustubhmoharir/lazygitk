package undo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// Fork: undo and redo of a rebase on a clean tree each show one confirmation,
// then run the captured hard reset. The reflog markers keep redo working.
var UndoRebaseCleanTreeAndRedo = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Undo and redo of a rebase on a clean tree show one confirmation each and reset",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")
		shell.EmptyCommit("three")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().Focus().
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
			).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().Confirmation().
					Title(Equals("Drop commit")).
					Content(Equals("Are you sure you want to drop the selected commit(s)?")).
					Confirm()
			}).
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			)

		t.Views().Commits().Focus().
			Press(keys.Universal.Undo)

		t.ExpectPopup().Confirmation().
			Title(Equals("Undo")).
			Content(Contains("Undo performs a hard reset")).
			Confirm()

		t.Views().Commits().
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
			)

		t.Views().Commits().Focus().
			Press(keys.Universal.Redo)

		t.ExpectPopup().Confirmation().
			Title(Equals("Redo")).
			Content(Contains("Redo performs a hard reset")).
			Confirm()

		t.Views().Commits().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			)
	},
})
