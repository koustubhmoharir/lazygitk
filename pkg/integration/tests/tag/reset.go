package tag

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var Reset = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Hard reset to a tag",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")
		shell.RunCommand([]string{"git", "tag", "-a", "tag", "-m", "tag", "HEAD^"}) // creating annotated tag on commit "one"
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		targetOID := t.Git().GetCommitHash("refs/tags/tag^{commit}")
		t.Views().Commits().Lines(
			Contains("two"),
			Contains("one"),
		)

		t.Views().Tags().
			Focus().
			Lines(
				Contains("tag").IsSelected(),
			).
			Press(keys.Commits.ViewResetOptions)

		t.ExpectPopup().Menu().
			Title(Contains("Reset to tag")).
			Select(Contains("Hard reset")).
			Confirm()

		review := t.ExpectPopup().Confirmation().Title(Equals("Hard reset")).
			Content(Contains(headOID).Contains(targetOID).Contains("untracked or ignored"))
		if t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Reviewing a hard reset moved HEAD before approval")
		}
		review.Confirm()
		t.Views().Commits().EventuallyContent(Contains("one").DoesNotContain("two"))

		t.Views().Commits().Lines(
			Contains("one"),
		)
	},
})
