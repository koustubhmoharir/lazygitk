package reflog

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var Reset = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Hard reset to a reflog commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")
		shell.EmptyCommit("three")
		shell.HardReset("HEAD^^")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		targetOID := t.Git().GetCommitHash("HEAD@{1}")
		t.Views().ReflogCommits().
			Focus().
			Lines(
				Contains("reset: moving to HEAD^^").IsSelected(),
				Contains("commit: three"),
				Contains("commit: two"),
				Contains("commit (initial): one"),
			).
			SelectNextItem().
			Press(keys.Commits.ViewResetOptions).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Contains("Reset to")).
					Select(Contains("Hard reset")).
					Confirm()

				review := t.ExpectPopup().Confirmation().Title(Equals("Hard reset")).
					Content(Contains(headOID).Contains(targetOID).Contains("untracked or ignored"))
				if t.Git().GetCommitHash("HEAD") != headOID {
					t.Fail("Reviewing a hard reset moved HEAD before approval")
				}
				review.Confirm()
			}).
			EventuallyContent(Contains("reset: moving to "+targetOID)).
			TopLines(
				Contains("reset: moving to").IsSelected(),
				Contains("reset: moving to HEAD^^"),
			)

		t.Views().Commits().
			Focus().
			EventuallyContent(Contains("three")).
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
			)
	},
})
