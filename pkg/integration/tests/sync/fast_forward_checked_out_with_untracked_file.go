package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardCheckedOutWithUntrackedFile = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward the checked out branch without a prompt when an unrelated untracked file is present",
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

		// an untracked file that the incoming commits do not touch
		shell.CreateFile("untracked", "mine")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		targetOID := t.Git().GetCommitHash("refs/remotes/origin/master")

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master ↓2").IsSelected(),
			).
			Press(keys.Branches.FastForward)

		// No popup is shown, so the branch list is still interactive and the
		// checked out branch has moved.
		t.Views().Branches().
			Lines(
				Contains("master ✓").IsSelected(),
			)
		if got := t.Git().GetCommitHash("HEAD"); got != targetOID {
			t.Fail("fast-forward did not move HEAD to the upstream: " + got)
		}

		t.Views().Commits().
			Lines(
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)
		t.FileSystem().FileContent("untracked", Equals("mine"))
	},
})
