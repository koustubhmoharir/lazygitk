package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardBehindBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward a branch that is behind its upstream branch, without checking it out",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.NewBranch("feature")
		shell.EmptyCommit("two")
		shell.EmptyCommit("three")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("feature", "origin/feature")

		// remove the two commits so that there is something to fast-forward to
		shell.HardReset("HEAD~2")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("cp .git/index .git/ff-review-index")
		sourceOID := t.Git().GetCommitHash("feature")
		targetOID := t.Git().GetCommitHash("refs/remotes/origin/feature")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("feature ↓2"),
			).
			SelectNextItem().
			Press(keys.Branches.FastForward)

		t.ReviewFastForwardFetch("feature", "origin", "feature")
		review := t.ExpectFastForwardReview("feature", sourceOID, targetOID, false)
		if got := t.Git().GetCommitHash("refs/heads/feature"); got != sourceOID {
			t.Fail("branch moved before approval: " + got)
		}
		review.Confirm()

		t.Views().Branches().
			Lines(
				Contains("master"),
				Contains("feature ✓").IsSelected(),
			).
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("off-branch fast-forward changed HEAD: " + got)
		}
		t.Shell().RunShellCommand("cmp .git/index .git/ff-review-index")
	},
})
