package undo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var UndoCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Undo/redo a commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("other-file", "other-file-1")
		shell.Commit("one")
		shell.CreateFileAndAdd("file", "file-1")
		shell.Commit("two")
		shell.UpdateFile("other-file", "other-file-2")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		confirmUndo := func() {
			t.ExpectPopup().Confirmation().
				Title(Equals("Undo")).
				Content(MatchesRegexp(`Are you sure you want to soft reset to '.*'\?`)).
				Confirm()
		}

		confirmRedo := func() {
			t.ExpectPopup().Confirmation().
				Title(Equals("Redo")).
				Content(Contains("Redo performs a hard reset")).
				Confirm()
		}

		confirmDiscardFile := func() {
			t.ExpectPopup().Menu().
				Title(Equals("Discard changes")).
				Select(Contains("Discard all changes")).
				Confirm()
		}

		t.Views().Files().
			Lines(
				Contains(" M other-file"),
			)

		t.Views().Commits().Focus().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			).
			Press(keys.Universal.Undo).
			Tap(confirmUndo).
			Lines(
				Contains("one").IsSelected(),
			)

		t.Views().Files().
			Lines(
				Equals("▼ /"),
				Equals("  A  file"),
				Equals("   M other-file"),
			)

		// Redo would hard reset over the unstaged change to other-file. It is
		// refused, and nothing is stashed.
		t.Views().Commits().Focus().
			Press(keys.Universal.Redo)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Commit or discard the changes to tracked files first. Redo never stashes.")).
			Confirm()

		t.Views().Commits().
			Lines(
				Contains("one").IsSelected(),
			)
		t.Views().Files().
			Lines(
				Equals("▼ /"),
				Equals("  A  file"),
				Equals("   M other-file"),
			)
		t.Views().Stash().IsEmpty()

		// Discard both changes, then redo the commit on a clean tree.
		t.Views().Files().Focus().
			Lines(
				Equals("▼ /"),
				Equals("  A  file"),
				Equals("   M other-file").IsSelected(),
			).
			Press(keys.Universal.PrevItem).
			Press(keys.Universal.Remove).
			Tap(confirmDiscardFile).
			Lines(
				Equals(" M other-file"),
			).
			Press(keys.Universal.Remove).
			Tap(confirmDiscardFile)

		t.Views().Files().IsEmpty()

		t.Views().Commits().Focus().
			Press(keys.Universal.Redo).
			Tap(confirmRedo)

		t.Views().Commits().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			)

		t.Views().Files().IsEmpty()
	},
})
