package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var CherryPickConflictsEmptyCommitAfterResolving = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry pick commits with conflicts, resolve them so that the commit becomes empty",
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
			Press(keys.Universal.RangeSelectDown).
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

		t.Views().Files().
			IsFocused().
			SelectedLine(Contains("file")).
			Press(keys.Universal.Remove)

		t.ExpectPopup().Menu().
			Title(Equals("Discard changes")).
			Select(Contains("Discard all changes")).
			Confirm()

		t.Common().ContinueOnConflictsResolved("cherry-pick")

		t.Views().Files().IsEmpty()

		t.Views().Commits().
			Focus().
			TopLines(
				Contains("second-change-branch unrelated change"),
				Contains("first change").IsSelected(),
				Contains("original"),
			)
	},
})
