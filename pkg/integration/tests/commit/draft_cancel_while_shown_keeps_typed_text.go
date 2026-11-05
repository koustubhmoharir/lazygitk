package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftCancelWhileShownKeepsTypedText = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Cancelling while a draft is shown keeps the text typed before Up as the newest draft",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.EmptyCommit("commit 3")
		shell.CreateFileAndAdd("myfile", "myfile content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Type(" draft").
			Cancel()

		t.Views().Files().
			Focus().
			PressEnter()
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			Type("typed text").
			SelectPreviousMessage().
			Content(Equals("commit 2 draft")).
			Cancel()

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			SelectPreviousMessage().
			Content(Equals("typed text")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft")).
			Cancel()
	},
})
