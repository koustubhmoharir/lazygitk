package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RewordHeadAfterExternalCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A reword of HEAD is refused when another commit lands while the message panel is open, and nothing is changed",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Lines(
				Contains("commit 2").IsSelected(),
				Contains("initial commit"),
			).
			Press(keys.Commits.RenameCommit)

		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Type(" new text")

		t.Shell().EmptyCommit("external commit")

		t.ExpectPopup().CommitMessagePanel().Confirm()
		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("The branch changed since the reword started").Contains("Nothing was changed")).
			Confirm()

		t.Views().Commits().
			Lines(
				Contains("external commit"),
				Contains("commit 2"),
				Contains("initial commit"),
			)
		// The refused text is kept as a draft: reopening reword on the commit shows it
		// through Up, while the original message is still the box's initial text.
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(keys.Commits.RenameCommit)
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			SelectPreviousMessage().
			Content(Equals("commit 2 new text")).
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")
	},
})
