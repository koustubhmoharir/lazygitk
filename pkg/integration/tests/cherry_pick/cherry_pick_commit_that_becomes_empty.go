package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CherryPickCommitThatBecomesEmpty = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry-pick a commit that becomes empty at the destination",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.
			EmptyCommit("base").
			CreateFileAndAdd("file1", "change 1\n").
			CreateFileAndAdd("file2", "change 2\n").
			Commit("two changes in one commit").
			NewBranchFrom("branch", "HEAD^").
			CreateFileAndAdd("file1", "change 1\n").
			Commit("single change").
			CreateFileAndAdd("file3", "change 3\n").
			Commit("unrelated change").
			Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		twoChangesOID := t.Git().GetCommitHash("master")
		singleChangeOID := t.Git().GetCommitHash("branch^")
		unrelatedChangeOID := t.Git().GetCommitHash("branch")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("branch"),
			).
			SelectNextItem().
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("unrelated change").IsSelected(),
				Contains("single change"),
				Contains("base"),
			).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Commits.CherryPickCopy).
			Tap(func() {
				t.Views().Information().Content(Contains("2 commits copied"))
			})

		t.Views().Commits().
			Focus().
			Lines(
				Contains("two changes in one commit").IsSelected(),
				Contains("base"),
			).
			Press(keys.Commits.PasteCommits).
			Tap(func() {
				t.Common().ChooseMarkedCherryPickSource()
				t.Common().ConfirmCherryPickList("refs/heads/master", twoChangesOID,
					CherryPickReviewRow(1, singleChangeOID, "single change"),
					CherryPickReviewRow(2, unrelatedChangeOID, "unrelated change"),
				)
			})

		if t.Git().Version().IsAtLeast(2, 45, 0) {
			t.Views().Commits().
				Lines(
					Contains("unrelated change"),
					Contains("single change"),
					Contains("two changes in one commit").IsSelected(),
					Contains("base"),
				).
				SelectPreviousItem()

			// Cherry-picked commit is empty
			t.Views().Main().Content(DoesNotContain("diff --git"))
		} else {
			// Older git versions drop the commit that became empty
			t.Views().Commits().
				Lines(
					Contains("unrelated change"),
					Contains("two changes in one commit").IsSelected(),
					Contains("base"),
				)
		}
	},
})
