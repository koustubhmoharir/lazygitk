package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

func reviewCurrentPull(t *TestDriver, keys config.KeybindingConfig) string {
	sourceOID := t.Git().GetCommitHash("HEAD")
	targetOID := t.Git().GetCommitHash("refs/remotes/origin/master")
	t.Views().Files().IsFocused().Press(keys.Universal.Pull)
	review := t.ReviewCurrentBranchPull("master", "origin", "master", sourceOID, targetOID)
	if got := t.Git().GetCommitHash("HEAD"); got != sourceOID {
		t.Fail("Fetching or reviewing changed HEAD before integration approval")
	}
	review.Confirm()
	return targetOID
}

var Pull = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Pull a commit from the remote",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")

		// remove the 'two' commit so that we have something to pull from the remote
		shell.HardReset("HEAD^")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Equals("↓1 repo → master"))

		targetOID := reviewCurrentPull(t, keys)

		t.Views().Commits().
			EventuallyContent(Contains("two")).
			Lines(
				Contains("two"),
				Contains("one"),
			)

		t.Views().Status().EventuallyContent(Equals("✓ repo → master"))
		if got := t.Git().GetCommitHash("HEAD"); got != targetOID {
			t.Fail("Pull did not integrate the reviewed incoming commit")
		}
	},
})
