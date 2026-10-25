package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchPreserveTracking = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push a selected branch without changing ambiguous upstream tracking",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.NewBranch("topic").EmptyCommit("topic-baseline")
		shell.CloneIntoRemote("origin")
		shell.Checkout("topic").EmptyCommit("topic-tip")
		shell.Checkout("master").EmptyCommit("master-tip")
		shell.RunCommand([]string{"git", "config", "branch.topic.remote", "origin"})
		shell.RunCommand([]string{"git", "config", "branch.topic.merge", "refs/heads/old-one"})
		shell.RunCommand([]string{"git", "config", "--add", "branch.topic.merge", "refs/heads/old-two"})
		shell.RunCommand([]string{"git", "config", "branch.topic.pushRemote", "origin"})
		shell.RunCommand([]string{"git", "config", "push.autoSetupRemote", "true"})
		shell.RunCommand([]string{"git", "config", "push.default", "current"})
		shell.CreateFileAndAdd("tracked", "staged\n")
		shell.CreateFile("tracked", "unstaged\n")
		shell.CreateFile("untracked-binary", "\x00\x01binary\xff")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		stagedBlobOID := t.Git().GetCommitHash(":tracked")
		remoteTopicOID := t.Git().GetCommitHash("refs/remotes/origin/topic")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		t.Shell().RunShellCommand("cp .git/config .git/assert-before-upstream-preserving-push")

		assertRemoteUnchanged := func() {
			t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-upstream-preserving-topic")
			t.FileSystem().FileContent(".git/assert-upstream-preserving-topic", Equals(remoteTopicOID+"\n"))
			t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-upstream-preserving-master")
			t.FileSystem().FileContent(".git/assert-upstream-preserving-master", Equals(remoteMasterOID+"\n"))
		}
		assertConfigUnchanged := func() {
			t.Shell().RunShellCommand("cmp .git/config .git/assert-before-upstream-preserving-push")
		}
		openPushFlow := func() {
			t.Views().Branches().Focus().NavigateToLine(Contains("topic")).Press(keys.Universal.Push)
			t.ExpectPopup().Menu().
				Title(Equals("Push selected branch")).
				Select(Contains("Toggle set upstream for selected local branch")).
				Confirm()
			t.ExpectPopup().Menu().
				Title(Equals("Push selected branch")).
				Select(Contains("Push this branch")).
				Confirm()
		}

		openPushFlow()
		guard := t.ExpectPopup().Menu().Title(Equals("Upstream settings need attention"))
		t.Views().Menu().SelectedLine(Contains("Cancel"))
		assertRemoteUnchanged()
		assertConfigUnchanged()
		guard.Confirm()

		openPushFlow()
		fallback := t.ExpectPopup().Menu().
			Title(Equals("Upstream settings need attention"))
		// Dropping set-upstream makes this an ordinary push, so the menu summary
		// is the review and the push runs without a second confirmation.
		assertRemoteUnchanged()
		assertConfigUnchanged()
		fallback.Select(Contains("Push without changing tracking")).Confirm()

		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-upstream-preserving-topic")
		t.FileSystem().FileContent(".git/assert-upstream-preserving-topic", Equals(topicOID+"\n"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-upstream-preserving-master")
		t.FileSystem().FileContent(".git/assert-upstream-preserving-master", Equals(remoteMasterOID+"\n"))
		assertConfigUnchanged()
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
