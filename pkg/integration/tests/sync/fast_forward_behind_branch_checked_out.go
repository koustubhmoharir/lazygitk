package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardBehindBranchCheckedOut = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward the checked out branch when it is behind its upstream branch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "content")
		shell.Commit("one")
		shell.EmptyCommit("two")
		shell.EmptyCommit("three")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")

		// remove the two commits so that there is something to fast-forward to
		shell.HardReset("HEAD~2")

		// a change that the fast-forward doesn't get in the way of
		shell.UpdateFile("file", "changed")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("master")
		targetOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		stagedBlobOID := t.Git().GetCommitHash(":file")
		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master ↓2").IsSelected(),
			).
			Press(keys.Branches.FastForward)

		t.ReviewFastForwardFetch("master", "origin", "master")
		review := t.ExpectFastForwardReview("master", sourceOID, targetOID, true)
		if got := t.Git().GetCommitHash("refs/heads/master"); got != sourceOID {
			t.Fail("branch moved before approval: " + got)
		}
		review.Confirm()

		t.Views().Branches().
			Lines(
				Contains("master ✓").IsSelected(),
			)

		t.Views().Commits().
			Lines(
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)

		// Moving the branch forward doesn't touch the change
		if got := t.Git().GetCommitHash(":file"); got != stagedBlobOID {
			t.Fail("fast-forward changed staged file content: " + got)
		}
		t.FileSystem().FileContent("file", Equals("changed"))
		t.Views().Files().
			Lines(
				Contains("file"),
			)
	},
})
