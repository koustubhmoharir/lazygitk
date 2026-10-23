package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var MergeFastForward = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Allow-fast-forward merge a selected divergent branch into current",
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
			NavigateToLine(Contains("branch1")).Press(keys.Branches.MergeIntoCurrentBranch)

		firstSourceOID := t.Git().GetCommitHash("refs/heads/branch1")
		t.ExpectPopup().Menu().Title(Equals("Merge")).
			Select(MatchesRegexp(`^[[:space:]]*f Merge \(allow fast-forward\)`)).Confirm()
		t.Common().ReviewBranchMerge("original-branch", "branch1", "Merge (allow fast-forward)")
		t.Views().Commits().EventuallyContent(Contains("branch1"))
		if got := t.Git().GetCommitHash("HEAD"); got != firstSourceOID {
			t.Fail("allow-fast-forward merge did not fast-forward current HEAD")
		}
		t.Views().Branches().Focus().NavigateToLine(Contains("branch2")).Press(keys.Branches.MergeIntoCurrentBranch)

		headOID := t.Git().GetCommitHash("HEAD")
		unrelatedOID := t.Git().GetCommitHash("refs/heads/branch1")
		sourceOID := t.Git().GetCommitHash("refs/heads/branch2")
		t.ExpectPopup().Menu().Title(Equals("Merge")).
			ContainsLines(
				Contains("Merge using configured options"),
				Contains("Merge (allow fast-forward)"),
				Contains("Merge (no fast-forward)"),
				Contains("Squash merge and leave uncommitted"),
				Contains("Squash merge and commit"),
			).
			Select(MatchesRegexp(`^[[:space:]]*f Merge \(allow fast-forward\)`)).Confirm()
		t.Common().ReviewBranchMerge("original-branch", "branch2", "Merge (allow fast-forward)")

		t.Views().Commits().EventuallyContent(Contains("branch2")).ContainsLines(
			Contains(`Merge branch "branch2" into "original-branch"`).IsSelected(),
			Contains("branch2"),
		)
		t.Git().CurrentBranchName("original-branch")
		if got := t.Git().GetCommitHash("HEAD"); got == headOID || got == sourceOID {
			t.Fail("allow-fast-forward merge of divergent branches did not create a merge commit")
		}
		if got := t.Git().GetCommitHash("refs/heads/branch1"); got != unrelatedOID {
			t.Fail("unexpectedly changed unrelated branch tip")
		}
		if got := t.Git().GetCommitHash("refs/heads/branch2"); got != sourceOID {
			t.Fail("selected source branch ref changed during merge")
		}
	},
})
