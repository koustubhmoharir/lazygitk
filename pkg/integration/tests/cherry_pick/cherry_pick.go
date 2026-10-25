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

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("four").IsSelected(),
				Contains("three"),
				Contains("base"),
			).
			// copy commits 'four' and 'three'
			Press(keys.Commits.CherryPickCopy).
			Tap(func() {
				t.Views().Information().Content(Contains("1 commit copied"))
			}).
			SelectNextItem().
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
				// cherry-picked commits will be deleted after confirmation
				t.Views().Information().Content(Contains("2 commits copied"))
			}).
			Tap(func() {
				t.Common().ChooseMarkedCherryPickSource()
				t.Common().ConfirmCherryPickList("refs/heads/first-branch", firstBranchOID,
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

		// Fork: a successful marked paste clears the marks instead of keeping a
		// hidden re-paste buffer, so pasting on master reviews only the selection.
		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("master")).
			PressPrimaryAction()

		t.Views().Commits().
			Focus().
			Lines(
				Contains("base").IsSelected(),
			).
			Press(keys.Commits.PasteCommits)
		t.ExpectPopup().Confirmation().
			Title(Equals("Cherry-pick")).
			Content(Contains(baseOID).DoesNotContain(threeOID).DoesNotContain(fourOID)).
			Cancel()
		t.Views().Commits().
			Lines(
				Contains("base").IsSelected(),
			)
	},
})
