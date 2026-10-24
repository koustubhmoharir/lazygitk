package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CherryPickRangeAfterPaste = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Regression test: range-copy multiple commits after a previous paste",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "recency"
	},
	SetupRepo: func(shell *Shell) {
		shell.
			EmptyCommit("base").
			NewBranch("target").
			NewBranch("source").
			EmptyCommit("one").
			EmptyCommit("two").
			EmptyCommit("three").
			EmptyCommit("four").
			EmptyCommit("five").
			Checkout("target")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		baseOID := t.Git().GetCommitHash("master")
		fiveOID := t.Git().GetCommitHash("source")
		twoOID := t.Git().GetCommitHash("source~3")
		threeOID := t.Git().GetCommitHash("source~2")
		fourOID := t.Git().GetCommitHash("source~1")
		var targetAfterFirstPaste string
		t.Views().Branches().
			Focus().
			Lines(
				Contains("target").IsSelected(),
				Contains("source"),
				Contains("master"),
			).
			SelectNextItem().
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("five").IsSelected(),
				Contains("four"),
				Contains("three"),
				Contains("two"),
				Contains("one"),
				Contains("base"),
			).
			Press(keys.Commits.CherryPickCopy)

		t.Views().Commits().
			Focus().
			Lines(
				Contains("base").IsSelected(),
			).
			Press(keys.Commits.PasteCommits).
			Tap(func() {
				t.Common().ChooseMarkedCherryPickSource()
				t.Common().ConfirmCherryPickList("refs/heads/target", baseOID,
					CherryPickReviewRow(1, fiveOID, "five"),
				)
			}).
			Lines(
				Contains("five"),
				Contains("base").IsSelected(),
			).
			Tap(func() {
				targetAfterFirstPaste = t.Git().GetCommitHash("target")
				// After paste, CherryPicking.DidPaste is true, so it looks to the user as if no
				// commits are copied:
				t.Views().Information().Content(DoesNotContain("commits copied"))
			})

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("source")).
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			NavigateToLine(Contains("four")).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Commits.CherryPickCopy).
			Tap(func() {
				t.Views().Information().Content(Contains("3 commits copied"))
			})

		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("base")).
			Press(keys.Commits.PasteCommits).
			Tap(func() {
				t.Common().ChooseMarkedCherryPickSource()
				t.Common().ConfirmCherryPickList("refs/heads/target", targetAfterFirstPaste,
					CherryPickReviewRow(1, twoOID, "two"),
					CherryPickReviewRow(2, threeOID, "three"),
					CherryPickReviewRow(3, fourOID, "four"),
				)
			})

		t.Views().Commits().Lines(
			Contains("four"),
			Contains("three"),
			Contains("two"),
			Contains("five"),
			Contains("base").IsSelected(),
		)
	},
})
