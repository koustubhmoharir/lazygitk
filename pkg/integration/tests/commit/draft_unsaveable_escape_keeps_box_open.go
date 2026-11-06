package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftUnsaveableEscapeKeepsBoxOpen = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Escape with the drafts file unusable keeps the message box open with its text and shows the error",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.EmptyCommit("commit 3")
		shell.CreateFileAndAdd("myfile", "myfile content")
		// The drafts file cannot be read or written while a directory is in its place.
		shell.CreateDir(".git/LAZYGITK_COMMIT_DRAFTS")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Type(" draft").
			Cancel()

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(MatchesRegexp("^The draft could not be saved, so the message box stays open: ")).
			Confirm()

		// The box is still open, so the text is not lost.
		t.ExpectPopup().CommitMessagePanel().
			Content(Equals("commit 2 draft")).
			Cancel()
		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(MatchesRegexp("^The draft could not be saved, so the message box stays open: ")).
			Confirm()

		// Once the drafts location is usable again, Escape saves the draft and closes the box.
		t.Shell().DeleteFile(".git/LAZYGITK_COMMIT_DRAFTS")
		t.ExpectPopup().CommitMessagePanel().
			Content(Equals("commit 2 draft")).
			Cancel()

		t.Views().Files().
			Focus().
			PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft")).
			Cancel()
	},
})
