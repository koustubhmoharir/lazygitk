package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push a selected branch without checking it out or changing the current worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("topic").EmptyCommit("topic-tip")
		shell.Checkout("master").EmptyCommit("master-tip")
		shell.CreateFileAndAdd("tracked", "staged\n")
		shell.CreateFile("tracked", "unstaged\n")
		shell.CreateFile("untracked-binary", "\x00\x01binary\xff")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		stagedBlobOID := t.Git().GetCommitHash(":tracked")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("topic")).
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Select(Contains("Push this branch")).
			Confirm()

		t.ExpectPopup().Prompt().Title(Equals("Remote name")).Type("origin").Confirm()

		t.ExpectPopup().Confirmation().
			Title(Equals("Review branch push")).
			Content(Contains(`source "topic"`).Contains(topicOID[:7]).Contains(`destination "refs/heads/topic"`)).
			Confirm()

		// Confirm waits for GUI idle, including the push worker and its refresh.
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-selected-branch-topic")
		t.FileSystem().FileContent(".git/assert-push-selected-branch-topic", Equals(topicOID+"\n"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-selected-branch-master")
		t.FileSystem().FileContent(".git/assert-push-selected-branch-master", Equals(remoteMasterOID+"\n"))
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Pushing the selected branch changed HEAD")
		}
		if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID {
			t.Fail("Pushing the selected branch changed the local master ref")
		}
		if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
			t.Fail("Pushing the selected branch changed the index")
		}
		t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
		t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
	},
})
