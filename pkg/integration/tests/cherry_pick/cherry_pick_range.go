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

		subCommits := t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("four").IsSelected(),
				Contains("three"),
				Contains("base"),
			).
			// select commits 'four' and 'three'
			Press(keys.Universal.RangeSelectDown).
			Lines(
				Contains("four").IsSelected(),
				Contains("three").IsSelected(),
				Contains("base"),
			)

		t.Common().CherryPickSelection(subCommits)
		t.Common().ConfirmCherryPickList("refs/heads/first-branch", destinationOID,
			CherryPickReviewRow(1, threeOID, "three"),
			CherryPickReviewRow(2, fourOID, "four"),
		)

		// The successful pick cleared the range, leaving the cursor on 'three'.
		subCommits.SelectedLines(Contains("three"))

		t.Views().Commits().Lines(
			Contains("four"),
			Contains("three"),
			Contains("two").IsSelected(),
			Contains("one"),
			Contains("base"),
		)
	},
})
