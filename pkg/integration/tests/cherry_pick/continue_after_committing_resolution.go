package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ContinueAfterCommittingResolution = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Commit a conflict resolution in the middle of a cherry-pick sequence, then continue the sequence",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("file", "base\n").
			Commit("base").
			NewBranch("topic").
			UpdateFileAndAdd("file", "one\n").
			Commit("one").
			CreateFileAndAdd("two", "two\n").
			Commit("two").
			CreateFileAndAdd("three", "three\n").
			Commit("three").
			Checkout("master").
			UpdateFileAndAdd("file", "main\n").
			Commit("main change")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		masterOID := t.Git().GetCommitHash("master")
		oneOID := t.Git().GetCommitHash("topic^^")
		twoOID := t.Git().GetCommitHash("topic^")
		threeOID := t.Git().GetCommitHash("topic")

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("topic")).
			PressEnter()

		subCommits := t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
				Contains("base"),
			).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Universal.RangeSelectDown).
			Lines(
				Contains("three").IsSelected(),
				Contains("two").IsSelected(),
				Contains("one").IsSelected(),
				Contains("base"),
			)

		t.Common().CherryPickSelection(subCommits)
		t.Common().ConfirmCherryPickList("refs/heads/master", masterOID,
			CherryPickReviewRow(1, oneOID, "one"),
			CherryPickReviewRow(2, twoOID, "two"),
			CherryPickReviewRow(3, threeOID, "three"),
		)

		t.Common().AcknowledgeConflicts()

		// Resolve the conflict, stage the file and commit it with the commit
		// panel while the other two picks are still pending.
		t.Views().Files().
			IsFocused().
			SelectedLine(Contains("file")).
			PressEnter()

		t.Views().MergeConflicts().
			IsFocused().
			// picking 'one'
			SelectNextItem().
			PressPrimaryAction()

		t.Views().Files().
			Focus().
			SelectedLine(Contains("file")).
			PressPrimaryAction()

		t.Views().Files().
			Press(keys.Files.CommitChanges)

		t.ExpectPopup().CommitMessagePanel().Type("resolved one").Confirm()

		t.Views().Status().
			Content(Contains("(cherry-picking) repo"))

		t.Views().Commits().
			Lines(
				Contains("─── Pending cherry-picks"),
				Contains("pick").Contains("three"),
				Contains("pick").Contains("two"),
				Contains("─── Commits"),
				Contains("resolved one"),
				Contains("main change"),
				Contains("base"),
			)

		t.Views().Commits().
			Focus().
			Press(keys.Universal.CreateRebaseOptionsMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Cherry-pick options")).
			Select(Contains("skip")).
			Confirm()
		t.ExpectToast(Equals("Disabled: This pick is already committed, so there is nothing to skip"))

		t.ExpectPopup().Menu().
			Title(Equals("Cherry-pick options")).
			Select(Contains("continue")).
			Confirm()

		t.Views().Status().
			Content(DoesNotContain("cherry-picking"))

		t.Views().Commits().
			Lines(
				Contains("three"),
				Contains("two"),
				Contains("resolved one"),
				Contains("main change"),
				Contains("base"),
			)
	},
})
