package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CherryPickRange = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry pick range of commits from the subcommits view, without conflicts",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "recency"
	},
	SetupRepo: func(shell *Shell) {
		shell.
			EmptyCommit("base").
			NewBranch("first-branch").
			NewBranch("second-branch").
			Checkout("first-branch").
			EmptyCommit("one").
			EmptyCommit("two").
			Checkout("second-branch").
			EmptyCommit("three").
			EmptyCommit("four").
			Checkout("first-branch")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		destinationOID := t.Git().GetCommitHash("first-branch")
		threeOID := t.Git().GetCommitHash("second-branch^")
		fourOID := t.Git().GetCommitHash("second-branch")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("first-branch"),
				Contains("second-branch"),
				Contains("master"),
			).
			SelectNextItem().
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("four").IsSelected(),
				Contains("three"),
				Contains("base"),
			).
			// copy commits 'four' and 'three'
			Press(keys.Universal.RangeSelectDown).
			Lines(
				Contains("four").IsSelected(),
				Contains("three").IsSelected(),
				Contains("base"),
			).
			Press(keys.Commits.CherryPickCopy)

		t.Views().Information().Content(Contains("2 commits copied"))

		t.Views().Commits().
			Focus().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
				Contains("base"),
			).
			Press(keys.Commits.PasteCommits).
			Tap(func() {
				t.Common().ChooseMarkedCherryPickSource()
				t.Common().ConfirmCherryPickList("refs/heads/first-branch", destinationOID,
					CherryPickReviewRow(1, threeOID, "three"),
					CherryPickReviewRow(2, fourOID, "four"),
				)
			}).
			Tap(func() {
				t.Views().Information().Content(DoesNotContain("commits copied"))
			}).
			Lines(
				Contains("four"),
				Contains("three"),
				Contains("two").IsSelected(),
				Contains("one"),
				Contains("base"),
			)
	},
})
