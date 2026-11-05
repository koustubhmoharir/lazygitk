package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftClearedWhenUsed = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Submitting the shown draft uses it up; text typed in a box that did not open with a draft keeps its draft for the next Up",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.CreateFileAndAdd("myfile", "myfile content")
		shell.CreateFile("otherfile", "other content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Submitting the shown draft uses it up, and keeps the text Up replaced.
		t.Views().Files().IsFocused().PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			Type("commit draft").
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		t.Views().Files().Focus().PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit draft")).
			Type(" more").
			SelectPreviousMessage().
			Content(Equals("commit draft")).
			Confirm()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		// Submitting that text as well, from a recall, leaves no drafts.
		t.Views().Commits().Focus().NavigateToLine(Contains("commit draft")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit draft")).
			SelectPreviousMessage().
			Content(Equals("commit draft more")).
			Confirm()
		t.FileSystem().PathNotPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		// Text typed in a box that did not open with a draft keeps its draft.
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Type(" draft to keep").
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		t.Views().Files().Focus().NavigateToLine(Contains("otherfile")).PressPrimaryAction()
		t.Views().Files().Focus().PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			Type("other text").
			Confirm()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		// The kept draft is still recalled by Up, in the reword box of the new HEAD.
		t.Views().Commits().Focus().NavigateToLine(Contains("other text")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("other text")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft to keep")).
			Cancel()
	},
})
