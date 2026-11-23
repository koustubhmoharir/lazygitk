package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullCommitTipBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Pull the local branch whose tip is the selected commit, not the checked-out branch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("topic")
		shell.RunCommand([]string{"git", "push", "origin", "refs/heads/topic:refs/heads/topic"})
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/topic", "refs/heads/topic"})
		shell.RunCommand([]string{"git", "config", "branch.topic.remote", "origin"})
		shell.RunCommand([]string{"git", "config", "branch.topic.merge", "refs/heads/topic"})
		shell.Checkout("master")
		shell.EmptyCommit("master-independent-tip")
		shell.NewBranchFrom("publisher", "refs/heads/topic")
		shell.CreateFileAndAdd("incoming-file", "incoming from origin\n").Commit("incoming topic")
		shell.RunCommand([]string{"git", "push", "origin", "HEAD:refs/heads/topic"})
		shell.Checkout("master")
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/topic", "refs/heads/topic"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		masterOID := t.Git().GetCommitHash("refs/heads/master")

		// The selected commit is topic's tip, which is behind origin/topic.
		// Pull opens the review for topic rather than for the checked-out master.
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("base")).
			Press(keys.Universal.Pull)

		t.ExpectPopup().Menu().
			Title(Equals("Update selected branch without checkout")).
			Content(Contains("Local branch \"topic\"")).
			Cancel()

		if got := t.Git().GetCommitHash("refs/heads/topic"); got != topicOID {
			t.Fail("Cancelling the pull changed the selected branch")
		}
		if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID {
			t.Fail("Cancelling the pull changed master")
		}
		t.Git().CurrentBranchName("master")
	},
})
