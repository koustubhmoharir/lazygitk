package tag

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ResetToDuplicateNamedBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Hard reset to a tag when a branch shares the same name",
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
		targetOID := t.Git().GetCommitHash("refs/tags/other-branch")
		t.Views().Commits().Lines(
			Contains("current-branch commit"),
			Contains("other-branch commit"),
			Contains("other-branch-tag commit"),
		)

		t.Views().Tags().
			Focus().
			Lines(
				Contains("other-branch").IsSelected(),
			).
			Press(keys.Commits.ViewResetOptions)

		t.ExpectPopup().Menu().
			Title(Contains("Reset to other-branch")).
			Select(Contains("Hard reset")).
			Confirm()

		review := t.ExpectPopup().Confirmation().Title(Equals("Hard reset")).
			Content(Contains(headOID).Contains(targetOID).Contains("untracked or ignored"))
		if t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Reviewing a hard reset moved HEAD before approval")
		}
		review.Confirm()
		t.Views().Commits().EventuallyContent(Contains("other-branch-tag commit").DoesNotContain("other-branch commit"))

		t.Views().Commits().Lines(
			Contains("other-branch-tag commit"),
		)
	},
})
