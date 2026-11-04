package reflog

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CherryPick = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry pick a reflog commit",
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
		destinationOID := t.Git().GetCommitHash("master")
		threeOID := t.Git().GetCommitHash("HEAD@{1}")
		reflog := t.Views().ReflogCommits().
			Focus().
			Lines(
				Contains("reset: moving to HEAD^^").IsSelected(),
				Contains("commit: three"),
				Contains("commit: two"),
				Contains("commit (initial): one"),
			).
			SelectNextItem()

		t.Common().CherryPickSelection(reflog)
		t.Common().ConfirmCherryPickList("refs/heads/master", destinationOID,
			CherryPickReviewRow(1, threeOID, "three"),
		)

		t.Views().Commits().
			Lines(
				Contains("three"),
				Contains("one").IsSelected(),
			)
	},
})
