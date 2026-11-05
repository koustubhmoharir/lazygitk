package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftRecallReturnsToTypedText = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Up in the commit message box shows the cancelled draft and does nothing more; Down returns to the text typed before Up",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.EmptyCommit("commit 3")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Type(" draft").
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Type(" more").
			SelectPreviousMessage().
			Content(Equals("commit 2 draft")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft")). // already shown: nothing changes
			SelectNextMessage().
			Content(Equals("commit 2 more")).
			Cancel()
	},
})
