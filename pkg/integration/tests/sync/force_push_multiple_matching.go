package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForcePushMultipleMatching = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Force push only the reviewed branch despite push.default matching",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
	},
	SetupRepo: func(shell *Shell) {
		shell.SetConfig("push.default", "matching")

		createTwoBranchesReadyToForcePush(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/master")
		leaseOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		otherRemoteOID := t.Git().GetCommitHash("refs/remotes/origin/other_branch")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/force-push-matching-head && cp .git/index .git/force-push-matching-index && git diff --binary > .git/force-push-matching-worktree")
		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Equals("↓1 repo → master"))

		t.Views().Branches().
			Lines(
				Contains("master ↓1"),
				Contains("other_branch ↓1"),
			)

		t.Views().Files().IsFocused().Press(keys.Universal.Push)

		menu := t.ReviewBranchForcePush("master", "master", sourceOID, "refs/remotes/origin/master", leaseOID)
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/master)\" = \"" + leaseOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/other_branch)\" = \"" + otherRemoteOID + "\"")
		menu.Select(Contains("Push with reviewed lease")).Confirm()

		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Equals("✓ repo → master"))

		t.Views().Branches().
			Lines(
				Contains("master ✓"),
				Contains("other_branch ↓1"),
			)
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/master)\" = \"" + sourceOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/other_branch)\" = \"" + otherRemoteOID + "\" && test \"$(git rev-parse HEAD)\" = \"$(cat .git/force-push-matching-head)\" && cmp .git/index .git/force-push-matching-index && git diff --binary > .git/force-push-matching-worktree-after && cmp .git/force-push-matching-worktree .git/force-push-matching-worktree-after")
	},
})
