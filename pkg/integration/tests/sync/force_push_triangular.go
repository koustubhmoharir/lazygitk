package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForcePushTriangular = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push to a remote, requiring a force push because the branch is behind the remote push branch but not the upstream",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.SetConfig("push.default", "current")

		shell.EmptyCommit("one")

		shell.CloneIntoRemote("origin")

		shell.NewBranch("feature")
		shell.SetBranchUpstream("feature", "origin/master")
		shell.EmptyCommit("two")
		shell.PushBranch("origin", "feature")

		// remove the 'two' commit so that we are behind the push branch
		shell.HardReset("HEAD^")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/feature")
		leaseOID := t.Git().GetCommitHash("refs/remotes/origin/feature")
		trackingOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/force-push-triangular-head && cp .git/index .git/force-push-triangular-index && git diff --binary > .git/force-push-triangular-worktree")
		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Contains("✓ repo → feature"))

		t.Views().Files().IsFocused().Press(keys.Universal.Push)

		menu := t.ReviewBranchForcePush("feature", "feature", sourceOID, "refs/remotes/origin/feature", leaseOID)
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/feature)\" = \"" + leaseOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/master)\" = \"" + trackingOID + "\"")
		menu.Select(Contains("Push with reviewed lease")).Confirm()

		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Contains("✓ repo → feature"))

		t.Views().Remotes().Focus().
			Lines(Contains("origin")).
			PressEnter()

		t.Views().RemoteBranches().IsFocused().
			Lines(
				Contains("feature"),
				Contains("master"),
			).
			PressEnter()

		t.Views().SubCommits().IsFocused().
			Lines(Contains("one"))
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/feature)\" = \"" + sourceOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/master)\" = \"" + trackingOID + "\" && test \"$(git rev-parse HEAD)\" = \"$(cat .git/force-push-triangular-head)\" && cmp .git/index .git/force-push-triangular-index && git diff --binary > .git/force-push-triangular-worktree-after && cmp .git/force-push-triangular-worktree .git/force-push-triangular-worktree-after")
	},
})
