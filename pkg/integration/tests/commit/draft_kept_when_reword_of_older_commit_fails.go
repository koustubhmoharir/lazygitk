package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftKeptWhenRewordOfOlderCommitFails = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A reword of an older commit that fails keeps its text as a draft, so it can be recalled and submitted again",
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
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			Type(" new text").
			Confirm()
		t.ExpectPopup().Alert().Title(Equals("Error")).Content(Contains("exit status 1")).Confirm()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		// The failure leaves the rebase stopped, so abort it first.
		t.GlobalPress(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"o"})
		t.ExpectPopup().Menu().Title(Equals("Rebase options")).Select(Contains("abort")).Confirm()

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			SelectPreviousMessage().
			Content(Equals("commit 2 new text")).
			Cancel()
		t.FileSystem().PathPresent(".git/LAZYGITK_COMMIT_DRAFTS")

		t.Shell().DeleteFile(".git/hooks/commit-msg")
		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			SelectPreviousMessage().
			Content(Equals("commit 2 new text")).
			Confirm()

		t.Views().Commits().Lines(
			Contains("commit 3"),
			Contains("commit 2 new text"),
			Contains("initial commit"),
		)
		t.FileSystem().PathNotPresent(".git/LAZYGITK_COMMIT_DRAFTS")
	},
})
