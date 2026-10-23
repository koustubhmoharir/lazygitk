package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullStackedBranchesWithLocalCommits = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cancel a reviewed pull on a stack with local commits and preserve every branch tip",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createStackRewrittenOnTheRemote(shell)
		shell.Checkout("branch2")
		shell.EmptyCommit("mine")
		shell.Checkout("branch3")
		shell.RunCommand([]string{"git", "rebase", "--onto", "branch2", "branch2~1"})
		shell.SetConfig("pull.rebase", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		branch3OID := t.Git().GetCommitHash("refs/heads/branch3")
		target3OID := t.Git().GetCommitHash("refs/remotes/origin/branch3")

		t.Views().Files().IsFocused().Press(keys.Universal.Pull)
		review := t.ReviewCurrentBranchPull("branch3", "origin", "branch3", branch3OID, target3OID)
		review.Cancel()

		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID ||
			t.Git().GetCommitHash("refs/heads/branch2") != branch2OID ||
			t.Git().GetCommitHash("refs/heads/branch3") != branch3OID {
			t.Fail("Canceling the reviewed pull changed a local branch tip")
		}
		t.Views().Commits().Content(Contains("mine").Contains("three"))
	},
})
