package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CherryPickDuringRebase = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry pick commits from the subcommits view during a rebase",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.Log.ShowGraph = "never"
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
		threeOID := t.Git().GetCommitHash("second-branch^")
		var pausedHeadOID string

		// Pause a rebase of first-branch, then pick 'three' into it.
		t.Views().Commits().
			Focus().
			Lines(
				Contains("CI two").IsSelected(),
				Contains("CI one"),
				Contains("CI base"),
			).
			SelectNextItem().
			Press(keys.Universal.Edit).
			Lines(
				Contains("─── Pending rebase todos"),
				Contains("pick CI two"),
				Contains("─── Commits"),
				Contains("     CI one").IsSelected(),
				Contains("     CI base"),
			).
			Tap(func() { pausedHeadOID = t.Git().GetCommitHash("HEAD") })

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("second-branch")).
			PressEnter()

		subCommits := t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("four").IsSelected(),
				Contains("three"),
				Contains("base"),
			).
			// select 'three'
			SelectNextItem()

		t.Common().CherryPickSelection(subCommits)
		t.Common().ConfirmCherryPickList("HEAD (detached)", pausedHeadOID,
			CherryPickReviewRow(1, threeOID, "three"),
		)

		t.Views().Commits().Lines(
			Contains("─── Pending rebase todos"),
			Contains("pick CI two"),
			Contains("─── Commits"),
			Contains("     CI three"),
			Contains("     CI one").IsSelected(),
			Contains("     CI base"),
		)

		t.Common().ContinueRebase()

		t.Views().Commits().Lines(
			Contains("CI two"),
			Contains("CI three"),
			Contains("CI one").IsSelected(),
			Contains("CI base"),
		)
	},
})
