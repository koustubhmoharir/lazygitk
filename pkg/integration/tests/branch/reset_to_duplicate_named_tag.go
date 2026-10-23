package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ResetToDuplicateNamedTag = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Hard reset to a branch when a tag shares the same name",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.NewBranch("current-branch")

		shell.EmptyCommit("other-branch-tag commit")
		shell.CreateLightweightTag("other-branch", "HEAD")

		shell.EmptyCommit("other-branch commit")
		shell.NewBranch("other-branch")

		shell.Checkout("current-branch")
		shell.EmptyCommit("current-branch commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		sourceOID := t.Git().GetCommitHash("refs/heads/other-branch")
		t.Views().Commits().Lines(
			Contains("current-branch commit"),
			Contains("other-branch commit"),
			Contains("other-branch-tag commit"),
		)

		t.Views().Branches().
			Focus().
			Lines(
				Contains("current-branch").IsSelected(),
				Contains("other-branch"),
			).
			SelectNextItem().
			Press(keys.Commits.ViewResetOptions)

		t.ExpectPopup().Menu().
			Title(Contains("Reset to other-branch")).
			Select(Contains("Hard reset")).
			Confirm()

		review := t.ExpectPopup().Confirmation().Title(Equals("Hard reset")).
			Content(Contains(headOID).Contains(sourceOID).Contains("untracked or ignored"))
		if t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Reviewing a hard reset moved HEAD before approval")
		}
		review.Confirm()
		t.Views().Commits().EventuallyContent(Contains("other-branch commit"))

		t.Views().Commits().
			Lines(
				Contains("other-branch commit"),
				Contains("other-branch-tag commit"),
			)
		t.Git().CurrentBranchName("current-branch")
		if t.Git().GetCommitHash("HEAD") != sourceOID || t.Git().GetCommitHash("refs/heads/other-branch") != sourceOID {
			t.Fail("Reset did not move only current HEAD to the reviewed source")
		}
	},
})
