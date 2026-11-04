package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CherryPick = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry pick commits from the subcommits view, without conflicts",
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
		baseOID := t.Git().GetCommitHash("master")
		firstBranchOID := t.Git().GetCommitHash("first-branch")
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
		t.Common().ConfirmCherryPickList("refs/heads/first-branch", firstBranchOID,
			CherryPickReviewRow(1, threeOID, "three"),
			CherryPickReviewRow(2, fourOID, "four"),
		)

		t.Views().Commits().
			Lines(
				Contains("four"),
				Contains("three"),
				Contains("two").IsSelected(),
				Contains("one"),
				Contains("base"),
			)

		// The successful pick cleared the selection, so the range is no longer shown.
		subCommits.SelectedLines(Contains("three"))

		// Pasting on master reviews only the commit selected there.
		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("master")).
			PressPrimaryAction()

		commits := t.Views().Commits().
			Focus().
			Lines(
				Contains("base").IsSelected(),
			)
		t.Common().CherryPickSelection(commits)
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(Contains(baseOID).DoesNotContain(threeOID).DoesNotContain(fourOID)).
			Cancel()
		commits.Lines(
			Contains("base").IsSelected(),
		)
	},
})
