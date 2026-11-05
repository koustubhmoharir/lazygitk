package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftClearedTextRemovesDraft = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Clearing a shown draft removes it, so Down returns to the original text and Up has nothing to recall",
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

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 3")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 3")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft")).
			Clear().
			SelectNextMessage().
			Content(Equals("commit 3")).
			SelectPreviousMessage().
			Content(Equals("commit 3")). // the draft is gone: nothing changes
			Cancel()

		t.FileSystem().PathNotPresent(".git/LAZYGITK_COMMIT_DRAFTS")
	},
})
