package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushStackedBranches = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Force push only the current branch from a rebased stack after reviewing its lease",
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
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-head-before && cp .git/index .git/push-stack-index-before && cp .git/config .git/push-stack-config-before && git diff --binary > .git/push-stack-worktree-before && git status --porcelain --untracked-files=all > .git/push-stack-status-before")

		t.Views().Files().IsFocused().Press(keys.Universal.Push)
		t.ExpectPopup().Menu().Title(Equals("Push selected branch"))
		t.Views().Menu().Content(DoesNotContain("Push all").DoesNotContain("stacked below"))
		menu := t.ReviewBranchForcePush("branch3", "branch3", sourceOID, "refs/remotes/origin/branch3", leaseOID)
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/branch3)\" = \"" + leaseOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch1)\" = \"" + remoteBranch1OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch2)\" = \"" + remoteBranch2OID + "\"")
		menu.Select(Contains("Push with reviewed lease")).Confirm()

		t.Views().Status().EventuallyContent(Contains("✓ repo → branch3"))
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/branch3)\" = \"" + sourceOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch1)\" = \"" + remoteBranch1OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch2)\" = \"" + remoteBranch2OID + "\" && ! git --git-dir=../origin show-ref --verify --quiet refs/heads/local-only")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-head-after && cp .git/index .git/push-stack-index-after && cp .git/config .git/push-stack-config-after && git diff --binary > .git/push-stack-worktree-after && git status --porcelain --untracked-files=all > .git/push-stack-status-after && cmp .git/push-stack-head-before .git/push-stack-head-after && cmp .git/push-stack-index-before .git/push-stack-index-after && cmp .git/push-stack-config-before .git/push-stack-config-after && cmp .git/push-stack-worktree-before .git/push-stack-worktree-after && cmp .git/push-stack-status-before .git/push-stack-status-after")
		t.Git().CurrentBranchName("branch3")
		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID || t.Git().GetCommitHash("refs/heads/branch2") != branch2OID {
			t.Fail("Pushing branch3 changed a related local branch")
		}
	},
})
