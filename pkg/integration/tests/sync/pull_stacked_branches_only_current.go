package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullStackedBranchesOnlyCurrent = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Pull only the checked-out branch using its configured rebase method",
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
		target3OID := t.Git().GetCommitHash("refs/remotes/origin/branch3")

		t.Views().Files().IsFocused().Press(keys.Universal.Pull)
		review := t.ReviewCurrentBranchPull("branch3", "origin", "branch3", branch3OID, target3OID)
		review.Confirm()
		t.Views().Commits().EventuallyContent(Contains("three-rewritten"))

		if got := t.Git().GetCommitHash("refs/heads/branch3"); got == branch3OID {
			t.Fail("Pulling the checked-out branch did not update its tip")
		}
		if got := t.Git().GetCommitHash("refs/remotes/origin/branch3"); got != target3OID {
			t.Fail("The approved fetch did not update branch3's reviewed tracking ref")
		}
		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID ||
			t.Git().GetCommitHash("refs/heads/branch2") != branch2OID {
			t.Fail("Pulling only the checked-out branch changed a lower branch")
		}
	},
})
