package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardDoesNotFetch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward a branch only to its local tracking ref, without fetching a newer upstream commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.NewBranch("feature")
		shell.EmptyCommit("two")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("feature", "origin/feature")

		// Push a newer commit to the remote by URL, which leaves the local
		// tracking ref at "two"
		shell.EmptyCommit("three")
		shell.RunCommand([]string{"git", "push", "../origin", "feature"})
		shell.HardReset("HEAD~2")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		trackingOID := t.Git().GetCommitHash("refs/remotes/origin/feature")
		t.Shell().RunShellCommand("git -C ../origin rev-parse refs/heads/feature > .git/ff-no-fetch-remote")

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("feature ↓1"),
			).
			SelectNextItem().
			Press(keys.Branches.FastForward)

		t.Views().Branches().
			Lines(
				Contains("master"),
				Contains("feature ✓").IsSelected(),
			)
		if got := t.Git().GetCommitHash("refs/heads/feature"); got != trackingOID {
			t.Fail("fast-forward did not move feature to the local tracking ref: " + got)
		}
		if got := t.Git().GetCommitHash("refs/remotes/origin/feature"); got != trackingOID {
			t.Fail("fast-forward changed the local tracking ref: " + got)
		}
		t.Shell().RunShellCommand("git -C ../origin rev-parse refs/heads/feature > .git/ff-no-fetch-remote-after && cmp .git/ff-no-fetch-remote .git/ff-no-fetch-remote-after")
	},
})
