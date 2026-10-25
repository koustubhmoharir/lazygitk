package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushSelectedBranchEscapeReturnsToMenu = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Escape on the remote prompt returns to the push menu, and cancelling that menu leaves the remote unchanged",
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

		t.Views().Branches().Focus().NavigateToLine(Contains("topic")).Press(keys.Universal.Push)
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Select(MatchesRegexp("^Choose remote")).
			Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Remote name")).Cancel()

		// Escape steps back to the push menu rather than cancelling the flow.
		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).Cancel()

		// Cancelling the menu pushed nothing: the remote still has only master.
		t.Shell().RunShellCommand("git --git-dir=../origin for-each-ref --format='%(refname)' refs/heads/ > .git/assert-escape-remote-refs")
		t.FileSystem().FileContent(".git/assert-escape-remote-refs", Equals("refs/heads/master\n"))
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Escape on the remote prompt changed HEAD")
		}
	},
})
