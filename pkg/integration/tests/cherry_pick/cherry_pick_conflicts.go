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

		t.Views().SubCommits().
			IsFocused().
			TopLines(
				Contains("second-change-branch unrelated change"),
				Contains("second change"),
			).
			Press(keys.Commits.CherryPickCopy).
			Tap(func() {
				t.Views().Information().Content(Contains("1 commit copied"))
			}).
			SelectNextItem().
			Press(keys.Commits.CherryPickCopy)

		t.Views().Information().Content(Contains("2 commits copied"))

		t.Views().Commits().
			Focus().
			TopLines(
				Contains("first change").IsSelected(),
			).
			Press(keys.Commits.PasteCommits)

		t.Common().ChooseMarkedCherryPickSource()
		t.Common().ConfirmCherryPickList("refs/heads/first-change-branch", firstChangeOID,
			CherryPickReviewRow(1, secondChangeOID, "second change"),
			CherryPickReviewRow(2, unrelatedOID, "second-change-branch unrelated change"),
		)

		t.Common().AcknowledgeConflicts()

		// cherry pick selection is not cleared when there are conflicts, so that the user
		// is able to abort and try again without having to re-copy the commits
		t.Views().Information().Content(Contains("2 commits copied"))

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

				t.Views().Information().Content(Contains("2 commits copied"))
			}).
			PressEscape().
			Tap(func() {
				t.Views().Information().Content(DoesNotContain("commits copied"))
			})
	},
})
