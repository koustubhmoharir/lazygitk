package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushCommitTipBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push the local branch whose tip is the selected commit, not the checked-out branch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.CreateFileAndAdd("tracked", "second\n").Commit("second")
		shell.NewBranch("topic")
		shell.Checkout("master")
		shell.EmptyCommit("third")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")

		// The topic tip is the selected commit, so Push targets topic even
		// though master is checked out.
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("second")).
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("Source: topic")).
			Select(Contains("Push this branch")).
			Confirm()

		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-commit-tip-topic")
		t.FileSystem().FileContent(".git/assert-push-commit-tip-topic", Equals(topicOID+"\n"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-commit-tip-master")
		t.FileSystem().FileContent(".git/assert-push-commit-tip-master", Equals(remoteMasterOID+"\n"))
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Pushing the commit's branch changed HEAD")
		}
		if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID {
			t.Fail("Pushing the commit's branch changed the local master ref")
		}

		// A commit with no branch tip keeps the checked-out branch as the target.
		t.Views().Commits().
			NavigateToLine(Contains("base")).
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("Source: master")).
			Cancel()
	},
})
