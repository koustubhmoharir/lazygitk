package undo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var UndoDrop = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Drop some commits and then undo/redo the actions",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")
		shell.EmptyCommit("three")
		shell.EmptyCommit("four")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		confirmCommitDrop := func() {
			t.ExpectPopup().Confirmation().
				Title(Equals("Drop commit")).
				Content(Equals("Are you sure you want to drop the selected commit(s)?")).
				Confirm()
		}

		confirmUndo := func() {
			t.ExpectPopup().Confirmation().
				Title(Equals("Undo")).
				Content(Contains("Undo performs a hard reset")).
				Confirm()
		}

		confirmRedo := func() {
			t.ExpectPopup().Confirmation().
				Title(Equals("Redo")).
				Content(Contains("Redo performs a hard reset")).
				Confirm()
		}

		t.Views().Commits().Focus().
			Lines(
				Contains("four").IsSelected(),
				Contains("three"),
				Contains("two"),
				Contains("one"),
			).
			Press(keys.Universal.Remove).
			Tap(confirmCommitDrop).
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
			).
			Press(keys.Universal.Remove).
			Tap(confirmCommitDrop).
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			).
			Press(keys.Universal.Undo).
			Tap(confirmUndo).
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
			).
			Press(keys.Universal.Undo).
			Tap(confirmUndo).
			Lines(
				Contains("four").IsSelected(),
				Contains("three"),
				Contains("two"),
				Contains("one"),
			).
			Press(keys.Universal.Redo).
			Tap(confirmRedo).
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
			).
			Press(keys.Universal.Redo).
			Tap(confirmRedo).
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			)
	},
})
