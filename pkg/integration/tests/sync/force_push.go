package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForcePush = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push to a remote with new commits, requiring a force push",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")

		// remove the 'two' commit so that we have something to pull from the remote
		shell.HardReset("HEAD^")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/master")
		leaseOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/force-push-head && cp .git/index .git/force-push-index && git diff --binary > .git/force-push-worktree")

		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Equals("↓1 repo → master"))

		t.Views().Files().IsFocused().Press(keys.Universal.Push)
		menu := t.ReviewBranchForcePush("master", "master", sourceOID, "refs/remotes/origin/master", leaseOID)
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/master)\" = \"" + leaseOID + "\"")
		menu.Select(Contains("Push with reviewed lease")).Confirm()

		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Equals("✓ repo → master"))

		t.Views().Remotes().Focus().
			Lines(Contains("origin")).
			PressEnter()

		t.Views().RemoteBranches().IsFocused().
			Lines(Contains("master")).
			PressEnter()

		t.Views().SubCommits().IsFocused().
			Lines(Contains("one"))
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/master)\" = \"" + sourceOID + "\" && test \"$(git rev-parse HEAD)\" = \"$(cat .git/force-push-head)\" && cmp .git/index .git/force-push-index && git diff --binary > .git/force-push-worktree-after && cmp .git/force-push-worktree .git/force-push-worktree-after")
	},
})
