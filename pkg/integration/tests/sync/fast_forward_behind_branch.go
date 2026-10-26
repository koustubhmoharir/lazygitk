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
		targetOID := t.Git().GetCommitHash("refs/remotes/origin/feature")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("feature ↓2"),
			).
			SelectNextItem().
			Press(keys.Branches.FastForward)

		// The branch moves straight away, without a review popup.
		t.Views().Branches().
			Lines(
				Contains("master"),
				Contains("feature ✓").IsSelected(),
			).
			PressEnter()

		if got := t.Git().GetCommitHash("refs/heads/feature"); got != targetOID {
			t.Fail("fast-forward did not move feature to its upstream: " + got)
		}
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
