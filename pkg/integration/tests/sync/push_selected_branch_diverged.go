package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchDiverged = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A plain push is disabled when the destination's tracking ref has commits the branch lacks; force with the reviewed lease is selected by default",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("topic").EmptyCommit("local-topic-tip")
		shell.Checkout("master")
		// Another clone adds a commit to the remote topic that the local branch lacks.
		shell.RunShellCommand("git clone -q ../origin ../remote-work && git -C ../remote-work checkout -q -b topic origin/master && git -C ../remote-work commit -q --allow-empty -m remote-topic-tip && git -C ../remote-work push -q origin topic")
		shell.RunCommand([]string{"git", "fetch", "origin"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		remoteTipOID := t.Git().GetCommitHash("refs/remotes/origin/topic")
		disabledText := "origin/topic has commits that topic does not have; use Force with reviewed lease…"

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("topic")).
			Press(keys.Universal.Push)

		// The plain push is shown but disabled, and pressing it explains why.
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("Remote: \"origin\"")).
			Select(Contains("Push this branch")).
			Tooltip(Equals("Disabled: " + disabledText)).
			Confirm().
			Tap(func() {
				t.ExpectToast(Equals("Disabled: " + disabledText))
			})

		// Force with the reviewed lease stays available. Its confirmation
		// selects the push by default, so it is confirmed without Select.
		t.ReviewBranchForcePush("topic", "topic", topicOID, "refs/remotes/origin/topic", remoteTipOID).Confirm()

		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/topic > .git/assert-push-selected-branch-diverged-topic")
		t.FileSystem().FileContent(".git/assert-push-selected-branch-diverged-topic", Equals(topicOID+"\n"))
	},
})
