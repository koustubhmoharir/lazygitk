package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchSplitPushUrl = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A plain push is not disabled by the fetched tracking ref when remote.origin.pushurl names a different repository",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		// The push destination is a separate repository that still has topic at base.
		shell.RunShellCommand("git init -q --bare ../push-target.git && git push -q ../push-target.git master:refs/heads/topic")
		shell.NewBranch("topic").EmptyCommit("local-topic-tip")
		shell.Checkout("master")
		// The fetched origin topic has a commit the local branch lacks. Only the fetch URL has it.
		shell.RunShellCommand("git clone -q ../origin ../remote-work && git -C ../remote-work checkout -q -b topic origin/master && git -C ../remote-work commit -q --allow-empty -m remote-topic-tip && git -C ../remote-work push -q origin topic")
		shell.RunCommand([]string{"git", "fetch", "origin"})
		shell.RunCommand([]string{"git", "config", "remote.origin.pushurl", "../push-target.git"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		originTopicOID := t.Git().GetCommitHash("refs/remotes/origin/topic")

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("topic")).
			Press(keys.Universal.Push)

		// The plain push is enabled: the fetch URL's tracking ref does not describe the push URL.
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("Remote: \"origin\"")).
			Select(Contains("Push this branch")).
			Confirm()

		t.Shell().RunShellCommand("git --git-dir=../push-target.git rev-parse --verify refs/heads/topic > .git/assert-push-split-url-topic")
		t.FileSystem().FileContent(".git/assert-push-split-url-topic", Equals(topicOID+"\n"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-split-url-origin")
		t.FileSystem().FileContent(".git/assert-push-split-url-origin", Equals(originTopicOID+"\n"))
	},
})
