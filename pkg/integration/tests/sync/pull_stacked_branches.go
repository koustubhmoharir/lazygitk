package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullStackedBranches = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Review a rewritten checked-out branch and refuse its rewritten lower branch without moving local refs",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createStackRewrittenOnTheRemote(shell)
		shell.Checkout("branch3")
		shell.SetConfig("pull.rebase", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		branch3OID := t.Git().GetCommitHash("refs/heads/branch3")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		target3OID := t.Git().GetCommitHash("refs/remotes/origin/branch3")

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
		diverged := t.ExpectPopup().Menu().Title(Equals("Review branch fast-forward"))
		t.Views().Menu().Content(Contains("local branch \"branch2\" has diverged"))
		t.Views().Menu().Content(Contains("will not reset").Contains("Cancel"))
		diverged.Cancel()

		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID ||
			t.Git().GetCommitHash("refs/heads/branch2") != branch2OID ||
			t.Git().GetCommitHash("refs/heads/branch3") != branch3OID ||
			t.Git().GetCommitHash("refs/heads/master") != masterOID {
			t.Fail("Reviewing the rewritten stack moved a local branch")
		}
	},
})
