package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchHistory = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push the branch shown in its history without checking it out",
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

		t.Views().Branches().Focus().NavigateToLine(Contains("topic")).PressEnter()
		t.Views().SubCommits().
			IsFocused().
			SelectedLine(Contains("topic-tip")).
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

		// Confirm waits for GUI idle before reading the actual remote ref.
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-selected-branch-history-topic")
		t.FileSystem().FileContent(".git/assert-push-selected-branch-history-topic", Equals(topicOID+"\n"))
		t.Views().SubCommits().
			IsFocused().
			SelectedLine(Contains("topic-tip"))
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Pushing from branch history changed HEAD")
		}
	},
})
