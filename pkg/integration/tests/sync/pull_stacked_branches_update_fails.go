package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullStackedBranchesUpdateFails = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Stop a reviewed stack update when a lower branch's approved remote fetch fails",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createStackRewrittenOnTheRemote(shell)
		// Make branch2's remote tip a fast-forward from its local tip, so review reaches branch1's fetch.
		shell.RunCommand([]string{"git", "push", "--force", "origin", "refs/heads/branch3:refs/heads/branch2"})
		shell.Checkout("branch3")
		shell.SetConfig("pull.rebase", "true")
		// The remote deletion is intentionally unobserved by the local remote-tracking ref.
		shell.RunCommand([]string{"git", "-C", "../origin", "branch", "-D", "branch1"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		branch3OID := t.Git().GetCommitHash("refs/heads/branch3")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		target3OID := t.Git().GetCommitHash("refs/remotes/origin/branch3")
		target2OID := t.Git().GetCommitHash("refs/remotes/origin/branch2")

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
			Content(Contains("couldn't find remote ref")).Confirm()

		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID ||
			t.Git().GetCommitHash("refs/heads/branch2") != branch2OID ||
			t.Git().GetCommitHash("refs/heads/branch3") != branch3OID ||
			t.Git().GetCommitHash("refs/heads/master") != masterOID {
			t.Fail("A failed lower-branch fetch moved a local branch")
		}
	},
})
