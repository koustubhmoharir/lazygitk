package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftClearedWhenUsed = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Submitting the shown draft clears it; submitting other text keeps the draft for the next Up",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.CreateFileAndAdd("myfile", "myfile content")
		shell.CreateFile("otherfile", "other content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Submitting the shown draft clears it.
		t.Views().Files().IsFocused().PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			Type("commit draft").
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFT")

		t.Views().Files().Focus().PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit draft")).
			Type(" more").
			SelectPreviousMessage().
			Content(Equals("commit draft")).
			Confirm()
		t.FileSystem().PathNotPresent(".git/LAZYGITK_COMMIT_DRAFT")

		// With the draft gone, a new box's Up does nothing.
		t.Views().Files().Focus().NavigateToLine(Contains("otherfile")).PressPrimaryAction()
		t.Views().Files().Focus().PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			SelectPreviousMessage().
			Content(Equals("")).
			Type("draft to keep").
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFT")

		// Submitting other text keeps the draft.
		t.Views().Files().Focus().PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("draft to keep")).
			Type(" edited").
			Confirm()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFT")

		// The kept draft is still recalled by Up, in the reword box of the new HEAD.
		t.Views().Commits().Focus().NavigateToLine(Contains("draft to keep edited")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("draft to keep edited")).
			SelectPreviousMessage().
			Content(Equals("draft to keep")).
			Cancel()
	},
})
