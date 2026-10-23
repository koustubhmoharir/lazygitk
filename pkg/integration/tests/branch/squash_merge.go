package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SquashMerge = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Squash merge a branch both with and without committing",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.NewBranch("original-branch").
			EmptyCommit("one").
			NewBranch("change-worktree-branch").
			CreateFileAndAdd("work", "content").
			Commit("work").
			Checkout("original-branch").
			NewBranch("change-commit-branch").
			CreateFileAndAdd("file", "content").
			Commit("file").
			Checkout("original-branch")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().TopLines(Contains("one"))

		t.Views().Branches().Focus().
			Lines(
				Contains("original-branch").IsSelected(),
				Contains("change-commit-branch"),
				Contains("change-worktree-branch"),
			).
			NavigateToLine(Contains("change-commit-branch")).Press(keys.Branches.MergeIntoCurrentBranch)
		sourceOID := t.Git().GetCommitHash("refs/heads/change-commit-branch")

		t.ExpectPopup().Menu().Title(Equals("Merge")).ContainsLines(
			Contains("Merge using configured options"),
			Contains("Merge (allow fast-forward)"),
			Contains("Merge (no fast-forward)"),
			Contains("Squash merge and leave uncommitted"),
			Contains("Squash merge and commit"),
		).Select(MatchesRegexp(`^[[:space:]]*S Squash merge and commit`)).Confirm()

		t.ExpectPopup().Prompt().Title(Equals("Squash commit message")).
			InitialText(Equals("Squash merge change-commit-branch into original-branch")).Confirm()
		t.Common().ReviewBranchMerge("original-branch", "change-commit-branch", "Squash merge and commit")

		t.Views().Commits().EventuallyContent(Contains("Squash merge change-commit-branch into original-branch")).TopLines(
			Contains("Squash merge change-commit-branch into original-branch").IsSelected(),
			Contains("one"),
		)
		if got := t.Git().GetCommitHash("refs/heads/change-commit-branch"); got != sourceOID {
			t.Fail("selected source branch ref changed during squash merge")
		}

		t.Views().Branches().Focus().NavigateToLine(Contains("change-worktree-branch")).Press(keys.Branches.MergeIntoCurrentBranch)
		t.ExpectPopup().Menu().Title(Equals("Merge")).
			Select(MatchesRegexp(`^[[:space:]]*s Squash merge and leave uncommitted`)).Confirm()
		t.Common().ReviewBranchMerge("original-branch", "change-worktree-branch", "Squash merge and leave uncommitted")

		t.Views().Files().Focus().EventuallyContent(Contains("work")).Lines(Contains("work"))
	},
})
