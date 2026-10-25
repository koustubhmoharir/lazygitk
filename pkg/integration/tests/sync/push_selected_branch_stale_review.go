package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchStaleReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Reopen the push menu with fresh details when the selected branch changes while the menu is open",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.RunCommand([]string{"git", "branch", "topic"})
		shell.CloneIntoRemote("origin")
		shell.Checkout("topic").EmptyCommit("topic-tip")
		shell.Checkout("master").EmptyCommit("master-tip")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		oldTopicOID := t.Git().GetCommitHash("refs/heads/topic")
		newTopicOID := t.Git().GetCommitHash("refs/heads/master")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		remoteTopicOID := t.Git().GetCommitHash("refs/remotes/origin/topic")
		assertRemoteUnchanged := func() {
			t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-stale-review-topic")
			t.FileSystem().FileContent(".git/assert-push-stale-review-topic", Equals(remoteTopicOID+"\n"))
			t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-stale-review-master")
			t.FileSystem().FileContent(".git/assert-push-stale-review-master", Equals(remoteMasterOID+"\n"))
		}

		t.Views().Branches().Focus().NavigateToLine(Contains("topic")).Press(keys.Universal.Push)
		menu := t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains(oldTopicOID[:7]))

		// The branch moves while the push menu is open. Choosing a push
		// must prepare against the reviewed OID, so it reopens the menu with
		// the fresh OID and does not contact the remote.
		t.Shell().RunCommand([]string{"git", "update-ref", "refs/heads/topic", newTopicOID, oldTopicOID})
		menu.Select(Contains("Push this branch")).Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Remote name")).Type("origin").Confirm()

		freshMenu := t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains(newTopicOID[:7]))
		assertRemoteUnchanged()

		// The remote is now known, so the fresh menu pushes directly.
		freshMenu.Select(Contains("Push this branch")).Confirm()

		// Confirm waits for the push worker and its refresh to finish.
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-stale-review-topic")
		t.FileSystem().FileContent(".git/assert-push-stale-review-topic", Equals(newTopicOID+"\n"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-stale-review-master")
		t.FileSystem().FileContent(".git/assert-push-stale-review-master", Equals(remoteMasterOID+"\n"))
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Refreshing a stale push menu changed HEAD")
		}
	},
})
