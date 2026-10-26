package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchTypedRemote = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push a selected branch with no configured remote, typing the remote name in the prompt, without a second confirmation",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("topic").EmptyCommit("topic-tip")
		shell.Checkout("master").EmptyCommit("master-tip")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")

		t.Views().Branches().Focus().NavigateToLine(Contains("topic")).Press(keys.Universal.Push)
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Select(MatchesRegexp("^Choose remote")).
			Confirm()
		// The prompt is prefilled with the sole remote; clear it before typing.
		t.ExpectPopup().Prompt().Title(Equals("Remote name")).InitialText(Equals("origin")).Clear().Type("origin").Confirm()

		// The menu summary is the review, so the typed remote pushes directly.
		// The assertions below wait for the push to finish.
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-typed-remote-topic")
		t.FileSystem().FileContent(".git/assert-push-typed-remote-topic", Equals(topicOID+"\n"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-typed-remote-master")
		t.FileSystem().FileContent(".git/assert-push-typed-remote-master", Equals(remoteMasterOID+"\n"))
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Pushing the selected branch with a typed remote changed HEAD")
		}
	},
})
