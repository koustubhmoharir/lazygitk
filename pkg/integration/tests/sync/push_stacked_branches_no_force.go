package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushStackedBranchesNoForce = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push only the current branch from an ahead-only stack through the normal final review",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.NewBranch("branch1")
		shell.NewBranch("branch2")
		shell.NewBranch("branch3")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")
		shell.SetBranchUpstream("branch1", "origin/branch1")
		shell.SetBranchUpstream("branch2", "origin/branch2")
		shell.SetBranchUpstream("branch3", "origin/branch3")

		shell.Checkout("branch1")
		shell.EmptyCommit("one")
		shell.Checkout("branch2")
		shell.HardReset("branch1")
		shell.EmptyCommit("two")
		shell.Checkout("branch3")
		shell.HardReset("branch2")
		shell.EmptyCommit("three")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/branch3")
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		remoteBranch1OID := t.Git().GetCommitHash("refs/remotes/origin/branch1")
		remoteBranch2OID := t.Git().GetCommitHash("refs/remotes/origin/branch2")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-ahead-head-before && cp .git/index .git/push-stack-ahead-index-before && cp .git/config .git/push-stack-ahead-config-before && git diff --binary > .git/push-stack-ahead-worktree-before")
		t.Views().Branches().
			Lines(
				Contains("branch3 ↑3"),
				Contains("branch1 ↑1"),
				Contains("branch2 ↑2"),
				Contains("master ✓"),
			)

		t.Views().Files().IsFocused().Press(keys.Universal.Push)
		t.ExpectPopup().Menu().Title(Equals("Push selected branch"))
		t.Views().Menu().Content(DoesNotContain("Push all").DoesNotContain("stacked below"))
		t.ReviewBranchPush("branch3", "origin", "branch3", sourceOID, false).Confirm()

		t.Views().Status().EventuallyContent(Contains("✓ repo → branch3"))
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/branch3)\" = \"" + sourceOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch1)\" = \"" + remoteBranch1OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch2)\" = \"" + remoteBranch2OID + "\" && ! git --git-dir=../origin show-ref --verify --quiet refs/heads/local-only")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-ahead-head-after && cp .git/index .git/push-stack-ahead-index-after && cp .git/config .git/push-stack-ahead-config-after && git diff --binary > .git/push-stack-ahead-worktree-after && cmp .git/push-stack-ahead-head-before .git/push-stack-ahead-head-after && cmp .git/push-stack-ahead-index-before .git/push-stack-ahead-index-after && cmp .git/push-stack-ahead-config-before .git/push-stack-ahead-config-after && cmp .git/push-stack-ahead-worktree-before .git/push-stack-ahead-worktree-after")
		if got := t.Git().GetCommitHash("refs/heads/branch1"); got != branch1OID {
			t.Fail("Pushing branch3 changed branch1")
		}
		if got := t.Git().GetCommitHash("refs/heads/branch2"); got != branch2OID {
			t.Fail("Pushing branch3 changed branch2")
		}
	},
})
