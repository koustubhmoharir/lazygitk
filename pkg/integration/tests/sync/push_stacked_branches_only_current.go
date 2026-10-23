package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushStackedBranchesOnlyCurrent = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cancel a reviewed force push from a rebased stack without changing any branch or worktree state",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createRebasedStackOfBranches(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/branch3")
		leaseOID := t.Git().GetCommitHash("refs/remotes/origin/branch3")
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		remoteBranch1OID := t.Git().GetCommitHash("refs/remotes/origin/branch1")
		remoteBranch2OID := t.Git().GetCommitHash("refs/remotes/origin/branch2")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-cancel-head-before && cp .git/index .git/push-stack-cancel-index-before && cp .git/config .git/push-stack-cancel-config-before && git diff --binary > .git/push-stack-cancel-worktree-before")

		t.Views().Files().IsFocused().Press(keys.Universal.Push)
		t.ExpectPopup().Menu().Title(Equals("Push selected branch"))
		t.Views().Menu().Content(DoesNotContain("Push all").DoesNotContain("stacked below"))
		menu := t.ReviewBranchForcePush("branch3", "branch3", sourceOID, "refs/remotes/origin/branch3", leaseOID)
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/branch3)\" = \"" + leaseOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch1)\" = \"" + remoteBranch1OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch2)\" = \"" + remoteBranch2OID + "\"")
		menu.Cancel()

		t.Git().CurrentBranchName("branch3")
		if got := t.Git().GetCommitHash("refs/heads/branch3"); got != sourceOID {
			t.Fail("Canceling force-push review changed the source branch")
		}
		if got := t.Git().GetCommitHash("refs/heads/branch1"); got != branch1OID {
			t.Fail("Canceling force-push review changed branch1")
		}
		if got := t.Git().GetCommitHash("refs/heads/branch2"); got != branch2OID {
			t.Fail("Canceling force-push review changed branch2")
		}
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-cancel-head-after && cp .git/index .git/push-stack-cancel-index-after && cp .git/config .git/push-stack-cancel-config-after && git diff --binary > .git/push-stack-cancel-worktree-after && cmp .git/push-stack-cancel-head-before .git/push-stack-cancel-head-after && cmp .git/push-stack-cancel-index-before .git/push-stack-cancel-index-after && cmp .git/push-stack-cancel-config-before .git/push-stack-cancel-config-after && cmp .git/push-stack-cancel-worktree-before .git/push-stack-cancel-worktree-after && test \"$(git --git-dir=../origin rev-parse refs/heads/branch3)\" = \"" + leaseOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch1)\" = \"" + remoteBranch1OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch2)\" = \"" + remoteBranch2OID + "\" && ! git --git-dir=../origin show-ref --verify --quiet refs/heads/local-only")
	},
})
