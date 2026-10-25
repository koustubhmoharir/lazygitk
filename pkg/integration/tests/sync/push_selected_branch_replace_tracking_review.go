package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchReplaceTrackingReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Review before setting upstream replaces different existing tracking, then push",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.NewBranch("topic").EmptyCommit("topic-tip")
		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("topic", "origin/topic")
		shell.Checkout("master").EmptyCommit("master-tip")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		remoteTopicOID := t.Git().GetCommitHash("refs/remotes/origin/topic")
		t.Shell().RunShellCommand("git config --get branch.topic.merge > .git/assert-replace-tracking-before-merge")
		t.FileSystem().FileContent(".git/assert-replace-tracking-before-merge", Equals("refs/heads/topic\n"))
		assertRemoteUnchanged := func() {
			t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-replace-tracking-remote-topic")
			t.FileSystem().FileContent(".git/assert-replace-tracking-remote-topic", Equals(remoteTopicOID+"\n"))
			t.Shell().RunShellCommand("! git --git-dir=../origin show-ref --verify --quiet refs/heads/other")
		}

		t.Views().Branches().Focus().NavigateToLine(Contains("topic")).Press(keys.Universal.Push)
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Select(Contains("Toggle set upstream for selected local branch")).
			Confirm()
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Select(Contains("Choose destination branch")).
			Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Destination branch name or refs/heads/...")).Clear().Type("other").Confirm()

		// Destination other than the existing tracking replaces it, so the
		// review is still shown and nothing is pushed before confirming.
		review := t.ExpectPopup().Confirmation().
			Title(Equals("Review branch push")).
			Content(Contains(`source "topic"`).Contains(topicOID[:7]).
				Contains(`destination "refs/heads/other"`).
				Contains("Set upstream for selected local branch: true").
				Contains("Existing upstream settings before this push"))
		assertRemoteUnchanged()
		t.Shell().RunShellCommand("git config --get branch.topic.merge > .git/assert-replace-tracking-during-merge")
		t.FileSystem().FileContent(".git/assert-replace-tracking-during-merge", Equals("refs/heads/topic\n"))
		review.Confirm()

		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/other > .git/assert-replace-tracking-remote-other")
		t.FileSystem().FileContent(".git/assert-replace-tracking-remote-other", Equals(topicOID+"\n"))
		t.Shell().RunShellCommand("git config --get branch.topic.merge > .git/assert-replace-tracking-after-merge")
		t.FileSystem().FileContent(".git/assert-replace-tracking-after-merge", Equals("refs/heads/other\n"))
		t.Git().CurrentBranchName("master")
	},
})
