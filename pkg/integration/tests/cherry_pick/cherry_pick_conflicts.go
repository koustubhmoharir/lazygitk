package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var CherryPickConflicts = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry pick commits from the subcommits view, with conflicts",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "recency"
		config.GetUserConfig().Git.AutoStageResolvedConflicts = true
	},
	SetupRepo: func(shell *Shell) {
		shared.MergeConflictsSetup(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		firstChangeOID := t.Git().GetCommitHash("first-change-branch")
		secondChangeOID := t.Git().GetCommitHash("second-change-branch^")
		unrelatedOID := t.Git().GetCommitHash("second-change-branch")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("first-change-branch"),
				Contains("second-change-branch"),
				Contains("original-branch"),
			).
			SelectNextItem().
			PressEnter()

		subCommits := t.Views().SubCommits().
			IsFocused().
			TopLines(
				Contains("second-change-branch unrelated change"),
				Contains("second change"),
			).
			// select 'unrelated change' and 'second change'
			Press(keys.Universal.RangeSelectDown).
			TopLines(
				Contains("second-change-branch unrelated change").IsSelected(),
				Contains("second change").IsSelected(),
			)

		t.Common().CherryPickSelection(subCommits)
		t.Common().ConfirmCherryPickList("refs/heads/first-change-branch", firstChangeOID,
			CherryPickReviewRow(1, secondChangeOID, "second change"),
			CherryPickReviewRow(2, unrelatedOID, "second-change-branch unrelated change"),
		)

		t.Common().AcknowledgeConflicts()

		// The pick stopped on conflicts, so the selection is kept, letting the user
		// abort and try again without selecting the commits again.
		subCommits.TopLines(
			Contains("second-change-branch unrelated change").IsSelected(),
			Contains("second change").IsSelected(),
		)

		t.Views().Files().
			IsFocused().
			SelectedLine(Contains("file")).
			PressEnter()

		t.Views().MergeConflicts().
			IsFocused().
			// picking 'Second change'
			SelectNextItem().
			PressPrimaryAction()

		t.Common().ContinueOnConflictsResolved("cherry-pick")

		t.Views().Files().IsEmpty()

		t.Views().Commits().
			Focus().
			TopLines(
				Contains("second-change-branch unrelated change"),
				Contains("second change").IsSelected(),
				Contains("first change"),
			).
			Tap(func() {
				// because we picked 'Second change' when resolving the conflict,
				// we now see this commit as having replaced First Change with Second Change,
				// as opposed to replacing 'Original' with 'Second change'
				t.Views().Main().
					Content(Contains("-First Change")).
					Content(Contains("+Second Change"))
			})
	},
})
