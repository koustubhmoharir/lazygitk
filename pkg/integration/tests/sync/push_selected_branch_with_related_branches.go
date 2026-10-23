package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchWithRelatedBranches = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push only the selected branch, leaving related lower branches unpublished and the current worktree untouched",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("lower").EmptyCommit("lower")
		shell.RunCommand([]string{"git", "config", "branch.lower.pushRemote", "origin"})
		shell.NewBranch("topic").EmptyCommit("topic")
		shell.RunCommand([]string{"git", "config", "branch.topic.pushRemote", "origin"})
		shell.Checkout("master").EmptyCommit("independent master")
		shell.CreateFileAndAdd("tracked", "staged\n")
		shell.CreateFile("tracked", "unstaged\n")
		shell.CreateFile("untracked-binary", "\x00\x01binary\xff")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Shell().RunShellCommand("cp .git/HEAD .git/push-selected-stack-review-head")
		t.Shell().RunShellCommand("cp .git/index .git/push-selected-stack-review-index")
		t.Shell().RunShellCommand("cp .git/config .git/push-selected-stack-review-config")
		// Keep raw snapshots as fixture files and compare them after the dialog flow.

		headOID := t.Git().GetCommitHash("HEAD")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		lowerOID := t.Git().GetCommitHash("refs/heads/lower")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		stagedBlobOID := t.Git().GetCommitHash(":tracked")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		assertSnapshotsUnchanged := func() {
			t.Shell().RunShellCommand("cmp .git/HEAD .git/push-selected-stack-review-head")
			t.Shell().RunShellCommand("cmp .git/index .git/push-selected-stack-review-index")
			t.Shell().RunShellCommand("cmp .git/config .git/push-selected-stack-review-config")
		}
		assertNoStackPush := func() {
			t.Shell().RunShellCommand("! git --git-dir=../origin show-ref --verify --quiet refs/heads/topic")
			t.Shell().RunShellCommand("! git --git-dir=../origin show-ref --verify --quiet refs/heads/lower")
			assertSnapshotsUnchanged()
		}
		assertLocalStateUnchanged := func() {
			t.Git().CurrentBranchName("master")
			if got := t.Git().GetCommitHash("HEAD"); got != headOID {
				t.Fail("Reviewing or pushing the selected stack changed HEAD")
			}
			if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID {
				t.Fail("Reviewing or pushing the selected stack changed the unrelated master ref")
			}
			if got := t.Git().GetCommitHash("refs/heads/lower"); got != lowerOID {
				t.Fail("Reviewing or pushing the selected stack changed the lower branch ref")
			}
			if got := t.Git().GetCommitHash("refs/heads/topic"); got != topicOID {
				t.Fail("Reviewing or pushing the selected stack changed the topic branch ref")
			}
			if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
				t.Fail("Reviewing or pushing the selected stack changed the index")
			}
			t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
			t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
			assertSnapshotsUnchanged()
		}
		assertNoSavedUpstreams := func() {
			t.Shell().RunShellCommand("! git config --get branch.topic.remote")
			t.Shell().RunShellCommand("! git config --get branch.topic.merge")
			t.Shell().RunShellCommand("! git config --get branch.lower.remote")
			t.Shell().RunShellCommand("! git config --get branch.lower.merge")
		}

		t.Views().Branches().Focus().NavigateToLine(Contains("topic")).Press(keys.Universal.Push)
		t.ExpectPopup().Menu().Title(Equals("Push selected branch"))
		t.Views().Menu().Content(DoesNotContain("stacked below").DoesNotContain("Push all"))
		assertNoStackPush()
		review := t.ReviewBranchPush("topic", "origin", "topic", topicOID, false)
		assertNoStackPush()
		assertNoSavedUpstreams()
		review.Confirm()
		t.Views().Remotes().Focus().NavigateToLine(Contains("origin")).PressEnter()
		t.Views().RemoteBranches().IsFocused().NavigateToLine(Contains("topic")).PressEnter()
		t.Views().SubCommits().IsFocused().EventuallyContent(Contains("topic"))
		t.Views().SubCommits().Lines(Contains("topic"), Contains("lower"), Contains("base"))

		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/push-selected-stack-review-remote-topic")
		t.FileSystem().FileContent(".git/push-selected-stack-review-remote-topic", Equals(topicOID+"\n"))
		t.Shell().RunShellCommand("! git --git-dir=../origin show-ref --verify --quiet refs/heads/lower")
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/push-selected-stack-review-remote-master")
		t.FileSystem().FileContent(".git/push-selected-stack-review-remote-master", Equals(remoteMasterOID+"\n"))
		assertNoSavedUpstreams()
		assertLocalStateUnchanged()
	},
})
