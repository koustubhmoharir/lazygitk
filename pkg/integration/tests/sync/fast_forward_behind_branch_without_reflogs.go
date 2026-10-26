package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardBehindBranchWithoutReflogs = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward a branch that is behind its upstream branch in a repo that keeps no reflogs",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		// This is the default in a bare repository, so it is what people with a
		// bare repo and linked worktrees work with
		shell.SetConfig("core.logAllRefUpdates", "false")

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

		t.Views().Branches().
			Lines(
				Contains("master"),
				Contains("feature ✓").IsSelected(),
			)
		if got := t.Git().GetCommitHash("refs/heads/feature"); got != targetOID {
			t.Fail("fast-forward did not move feature to its upstream: " + got)
		}
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("off-branch fast-forward changed HEAD: " + got)
		}
		t.Shell().RunShellCommand("cmp .git/index .git/ff-review-index")
	},
})
