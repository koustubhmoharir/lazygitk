package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftSubmitKeepsTypedText = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Submitting a recalled draft uses it up, but the text typed before Up is kept as a draft",
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
			Type(" draft A").
			Cancel()

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 3")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 3")).
			Type(" typed T").
			SelectPreviousMessage().
			Content(Equals("commit 2 draft A")).
			Confirm()

		t.Views().Commits().Lines(
			Contains("commit 2 draft A"),
			Contains("commit 2"),
			Contains("initial commit"),
		)

		// Draft A was used up, so the text typed before Up is the only draft left.
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2 draft A")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2 draft A")).
			SelectPreviousMessage().
			Content(Equals("commit 3 typed T")).
			SelectPreviousMessage().
			Content(Equals("commit 3 typed T")). // nothing older: nothing changes
			Cancel()
	},
})
