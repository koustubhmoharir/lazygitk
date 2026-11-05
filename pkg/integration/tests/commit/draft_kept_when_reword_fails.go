package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

// Fails every commit that runs the commit-msg hook, which amend does.
var failingCommitMsgHook = `#!/bin/bash

# For this test all we need is a hook that always fails
exit 1
`

var DraftKeptWhenRewordFails = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A reword that fails keeps its text as a draft, so it can be recalled and submitted again",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.EmptyCommit("commit 3")
		// Added last, because the setup commits would run it too.
		shell.CreateFile(".git/hooks/commit-msg", failingCommitMsgHook)
		shell.MakeExecutable(".git/hooks/commit-msg")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 3")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 3")).
			Type(" new text").
			Confirm()
		t.ExpectPopup().Alert().Title(Equals("Error")).Content(Contains("Git command failed")).Confirm()

		// The text is recalled in a new box, which opens with the old message.
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 3")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 3")).
			SelectPreviousMessage().
			Content(Equals("commit 3 new text")).
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		// With the hook gone, the recalled text can be submitted.
		t.Shell().DeleteFile(".git/hooks/commit-msg")
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 3")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 3")).
			SelectPreviousMessage().
			Content(Equals("commit 3 new text")).
			Confirm()

		t.Views().Commits().Lines(
			Contains("commit 3 new text"),
			Contains("commit 2"),
			Contains("initial commit"),
		)
		t.FileSystem().PathNotPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		// Nothing is left to recall.
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 3 new text")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 3 new text")).
			SelectPreviousMessage().
			Content(Equals("commit 3 new text")).
			Cancel()
	},
})
