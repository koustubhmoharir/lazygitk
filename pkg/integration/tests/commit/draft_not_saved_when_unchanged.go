package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftNotSavedWhenUnchanged = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Escape in the commit message box with no change does not overwrite the existing draft",
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

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Cancel()

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft")).
			Cancel()
	},
})
