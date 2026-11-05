package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftRecallStepsThroughList = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Up and Down step through every cancelled draft, newest first, and return to the text the box opened with",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.EmptyCommit("commit 3")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		for _, suffix := range []string{" draft one", " draft two", " draft three"} {
			t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
			t.ExpectPopup().CommitMessagePanel().
				InitialText(Equals("commit 2")).
				Type(suffix).
				Cancel()
		}

		t.Views().Commits().Focus().NavigateToLine(Contains("commit 2")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft three"))
		t.Views().CommitMessage().Subtitle(Contains("Draft 1 of 3"))

		t.ExpectPopup().CommitMessagePanel().
			SelectPreviousMessage().
			Content(Equals("commit 2 draft two")).
			SelectPreviousMessage().
			Content(Equals("commit 2 draft one"))
		t.Views().CommitMessage().Subtitle(Contains("Draft 3 of 3"))

		t.ExpectPopup().CommitMessagePanel().
			SelectPreviousMessage().
			Content(Equals("commit 2 draft one")) // the oldest: nothing changes
		t.Views().CommitMessage().Subtitle(Contains("Draft 3 of 3"))

		t.ExpectPopup().CommitMessagePanel().
			SelectNextMessage().
			Content(Equals("commit 2 draft two")).
			SelectNextMessage().
			Content(Equals("commit 2 draft three")).
			SelectNextMessage().
			Content(Equals("commit 2"))
		t.Views().CommitMessage().Subtitle(DoesNotContain("Draft"))

		t.ExpectPopup().CommitMessagePanel().Cancel()
	},
})
