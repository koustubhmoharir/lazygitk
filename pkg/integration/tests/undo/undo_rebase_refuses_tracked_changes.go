package undo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// Fork: undo of a rebase never stashes. With tracked changes it is refused, and
// the changes stay where they are.
var UndoRebaseRefusesTrackedChanges = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Undo of a rebase is refused while tracked files have changes, and nothing is stashed",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "one")
		shell.Commit("one")
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

		t.Shell().UpdateFile("file", "changed")
		t.GlobalPress(keys.Universal.Refresh)
		t.Views().Files().
			Lines(
				Contains("M file"),
			)

		t.Views().Commits().Focus().
			Press(keys.Universal.Undo)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Commit or discard the changes to tracked files first. Undo never stashes.")).
			Confirm()

		t.Views().Commits().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			)
		t.Views().Files().
			Lines(
				Contains("M file"),
			)
		t.Views().Stash().IsEmpty()
	},
})
