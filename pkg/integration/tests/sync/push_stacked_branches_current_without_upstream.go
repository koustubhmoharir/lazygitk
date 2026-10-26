package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushStackedBranchesCurrentWithoutUpstream = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Choose a remote and push only the current branch from a stack without setting its upstream",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createRebasedStackOfBranches(shell)
		shell.NewBranch("branch4")
		shell.EmptyCommit("four")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/branch4")
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		branch3OID := t.Git().GetCommitHash("refs/heads/branch3")
		remoteBranch1OID := t.Git().GetCommitHash("refs/remotes/origin/branch1")
		remoteBranch2OID := t.Git().GetCommitHash("refs/remotes/origin/branch2")
		remoteBranch3OID := t.Git().GetCommitHash("refs/remotes/origin/branch3")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-no-upstream-head-before && cp .git/index .git/push-stack-no-upstream-index-before && cp .git/config .git/push-stack-no-upstream-config-before && git diff --binary > .git/push-stack-no-upstream-worktree-before")

		t.Views().Files().IsFocused().Press(keys.Universal.Push)
		t.ExpectPopup().Menu().Title(Equals("Push selected branch"))
		t.Views().Menu().Content(DoesNotContain("Push all").DoesNotContain("stacked below"))
		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).Select(MatchesRegexp("^Choose remote")).Confirm()
		t.Shell().RunShellCommand("! git --git-dir=../origin show-ref --verify --quiet refs/heads/branch4 && test \"$(git --git-dir=../origin rev-parse refs/heads/branch1)\" = \"" + remoteBranch1OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch2)\" = \"" + remoteBranch2OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch3)\" = \"" + remoteBranch3OID + "\"")
		t.ExpectPopup().Prompt().Title(Equals("Remote name")).InitialText(Equals("origin")).Confirm()

		// There is deliberately no saved upstream, so status cannot acquire a
		// tracking checkmark. Wait for the newly pushed destination's history.
		t.Views().Remotes().Focus().NavigateToLine(Contains("origin")).PressEnter()
		t.Views().RemoteBranches().IsFocused().EventuallyContent(Contains("branch4")).
			NavigateToLine(Contains("branch4")).PressEnter()
		t.Views().SubCommits().IsFocused().EventuallyContent(Contains("four"))
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/branch4)\" = \"" + sourceOID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch1)\" = \"" + remoteBranch1OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch2)\" = \"" + remoteBranch2OID + "\" && test \"$(git --git-dir=../origin rev-parse refs/heads/branch3)\" = \"" + remoteBranch3OID + "\" && ! git --git-dir=../origin show-ref --verify --quiet refs/heads/local-only")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-stack-no-upstream-head-after && cp .git/index .git/push-stack-no-upstream-index-after && cp .git/config .git/push-stack-no-upstream-config-after && git diff --binary > .git/push-stack-no-upstream-worktree-after && cmp .git/push-stack-no-upstream-head-before .git/push-stack-no-upstream-head-after && cmp .git/push-stack-no-upstream-index-before .git/push-stack-no-upstream-index-after && cmp .git/push-stack-no-upstream-config-before .git/push-stack-no-upstream-config-after && cmp .git/push-stack-no-upstream-worktree-before .git/push-stack-no-upstream-worktree-after")
		t.Git().CurrentBranchName("branch4")
		for _, branch := range []struct{ name, oid string }{{"branch1", branch1OID}, {"branch2", branch2OID}, {"branch3", branch3OID}} {
			if got := t.Git().GetCommitHash("refs/heads/" + branch.name); got != branch.oid {
				t.Fail("Pushing branch4 changed " + branch.name)
			}
		}
		t.Shell().RunShellCommand("! git config --get branch.branch4.remote && ! git config --get branch.branch4.merge")
	},
})
