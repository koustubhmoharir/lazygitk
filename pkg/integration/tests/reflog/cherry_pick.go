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
		t.Views().ReflogCommits().
			Focus().
			Lines(
				Contains("reset: moving to HEAD^^").IsSelected(),
				Contains("commit: three"),
				Contains("commit: two"),
				Contains("commit (initial): one"),
			).
			SelectNextItem().
			Press(keys.Commits.CherryPickCopy)

		t.Views().Information().Content(Contains("1 commit copied"))

		t.Views().Commits().
			Focus().
			Lines(
				Contains("one").IsSelected(),
			).
			Press(keys.Commits.PasteCommits).
			Tap(func() {
				t.Common().ChooseMarkedCherryPickSource()
				t.Common().ConfirmCherryPickList("refs/heads/master", destinationOID,
					CherryPickReviewRow(1, threeOID, "three"),
				)
			}).
			Lines(
				Contains("three"),
				Contains("one").IsSelected(),
			)
	},
})
