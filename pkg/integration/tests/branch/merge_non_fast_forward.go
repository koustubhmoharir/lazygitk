package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var MergeNonFastForward = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Merge selected branches with configured and no-fast-forward modes",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		shell.NewBranch("original-branch").
			EmptyCommit("one").
			NewBranch("branch1").
			EmptyCommit("branch1").
			Checkout("original-branch").
			NewBranchFrom("branch2", "original-branch").
			EmptyCommit("branch2").
			Checkout("original-branch")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().Focus().
			Lines(Contains("original-branch").IsSelected(), Contains("branch1"), Contains("branch2")).
			NavigateToLine(Contains("branch2")).Press(keys.Branches.MergeIntoCurrentBranch)

		t.ExpectPopup().Menu().Title(Equals("Merge")).
			ContainsLines(
				Contains("Merge using configured options"),
				Contains("Merge (allow fast-forward)"),
				Contains("Merge (no fast-forward)"),
				Contains("Squash merge and leave uncommitted"),
				Contains("Squash merge and commit"),
			).
			Select(MatchesRegexp(`^[[:space:]]*n Merge \(no fast-forward\)`)).Confirm()
		t.Common().ReviewBranchMerge("original-branch", "branch2", "Merge (no fast-forward)")

		t.Views().Commits().EventuallyContent(Contains(`Merge branch "branch2" into "original-branch"`)).ContainsLines(
			Contains(`Merge branch "branch2" into "original-branch"`).IsSelected(),
			Contains("branch2"),
		)

		t.Views().Branches().Focus().NavigateToLine(Contains("branch1")).Press(keys.Branches.MergeIntoCurrentBranch)
		t.ExpectPopup().Menu().Title(Equals("Merge")).
			Select(MatchesRegexp(`^[[:space:]]*m Merge using configured options`)).Confirm()
		t.Common().ReviewBranchMerge("original-branch", "branch1", "Merge using configured options")

		t.Views().Commits().EventuallyContent(Contains(`Merge branch "branch1" into "original-branch"`)).ContainsLines(
			Contains(`Merge branch "branch1" into "original-branch"`).IsSelected(),
			Contains("branch1"),
			Contains(`Merge branch "branch2" into "original-branch"`),
		)
	},
})
