package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullStackedBranchesCheckedOutInOtherWorktree = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse a reviewed stack update when a lower branch is occupied by another worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createStackRewrittenOnTheRemote(shell)
		// Make branch2's remote tip a fast-forward from its local tip, so review reaches occupied branch1.
		shell.RunCommand([]string{"git", "push", "--force", "origin", "refs/heads/branch3:refs/heads/branch2"})
		// Occupancy, rather than divergence, must be the reason branch1's
		// otherwise valid fast-forward is refused.
		shell.RunCommand([]string{"git", "push", "--force", "origin", "refs/heads/branch2:refs/heads/branch1"})
		shell.Checkout("branch3")
		shell.SetConfig("pull.rebase", "true")
		shell.AddWorktreeCheckout("branch1", "../linked-worktree")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		branch3OID := t.Git().GetCommitHash("refs/heads/branch3")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		target3OID := t.Git().GetCommitHash("refs/remotes/origin/branch3")
		target2OID := t.Git().GetCommitHash("refs/remotes/origin/branch2")
		t.Shell().RunShellCommand("cp .git/index .git/stack-occupied-main-index && cp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/stack-occupied-linked-index && git -C ../linked-worktree status --porcelain=v1 -z --untracked-files=all > .git/stack-occupied-linked-status")

		t.Views().Files().IsFocused().Press(keys.Universal.Pull)
		t.ExpectPopup().Menu().Title(Equals("Pull checked-out branch")).
			Select(Contains("Review updates for this branch and branches stacked below…")).Confirm()
		t.ExpectPopup().Menu().Title(Equals("Pull checked-out branch")).
			Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("branch3", "origin", "branch3")
		t.ExpectPopup().Menu().Title(Equals("Integrate reviewed incoming commit")).
			Select(MatchesRegexp(`Use Git's configured pull method|Configured rebase`)).Confirm()
		t.ExpectPopup().Confirmation().Title(Equals("Review current-branch integration")).
			Content(Contains(branch3OID).Contains(target3OID)).Confirm()

		t.ExpectPopup().Menu().Title(Equals("Update selected branch without checkout")).
			Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("branch2", "origin", "branch2")
		t.ExpectFastForwardReview("branch2", branch2OID, target2OID, false).Confirm()

		t.ExpectPopup().Menu().Title(Equals("Update selected branch without checkout")).
			Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("branch1", "origin", "branch1")
		t.ExpectPopup().Alert().Title(Equals("Error")).
			Content(Contains("branch \"branch1\" is reserved by another worktree").Contains("cannot be moved here")).Confirm()

		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID ||
			t.Git().GetCommitHash("refs/heads/branch2") != branch2OID ||
			t.Git().GetCommitHash("refs/heads/branch3") != branch3OID ||
			t.Git().GetCommitHash("refs/heads/master") != masterOID {
			t.Fail("Refusing the occupied branch moved a local branch")
		}
		t.Shell().RunShellCommand("test \"$(git -C ../linked-worktree rev-parse HEAD)\" = \"$(git rev-parse refs/heads/branch1)\" && cmp .git/index .git/stack-occupied-main-index && cmp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/stack-occupied-linked-index && git -C ../linked-worktree status --porcelain=v1 -z --untracked-files=all > .git/stack-occupied-linked-status-after && cmp .git/stack-occupied-linked-status .git/stack-occupied-linked-status-after")
	},
})
