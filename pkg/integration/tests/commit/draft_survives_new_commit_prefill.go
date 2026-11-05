package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftSurvivesNewCommitPrefill = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A cancelled new commit message still reopens prefilled, and Up does nothing because the draft equals it",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("myfile", "myfile content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			PressEnter()

		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			Type("my commit message").
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGIT_PENDING_COMMIT")
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		t.Views().Files().
			Focus().
			PressEnter()

		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("my commit message")).
			SelectPreviousMessage().
			Content(Equals("my commit message")).
			Confirm()
		t.FileSystem().PathNotPresent(".git/LAZYGIT_PENDING_COMMIT")
		t.FileSystem().PathNotPresent(".git/LAZYGITK_COMMIT_DRAFTS")
	},
})
