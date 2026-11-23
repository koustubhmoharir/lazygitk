package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PlaceAtRemoteBranchPrefillsName = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Create/move branch here from a remote branch prefills the name prompt with the branch name without its remote, and creates the branch at the remote branch commit without changing HEAD",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("master-01").
			CloneIntoRemote("origin").
			NewBranch("feature").
			EmptyCommit("feature-01").
			PushBranch("origin", "feature").
			Checkout("master").
			RunCommand([]string{"git", "branch", "-D", "feature"}).
			EmptyCommit("master-02")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		remoteFeatureOID := t.Git().GetCommitHash("refs/remotes/origin/feature")
		if remoteFeatureOID == headOID {
			t.Fail("Expected origin/feature to differ from HEAD")
		}

		t.Views().Remotes().Focus().NavigateToLine(Contains("origin")).PressEnter()
		remoteBranches := t.Views().RemoteBranches()
		remoteBranches.IsFocused().NavigateToLine(Contains("feature")).
			Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Filter("Create/move branch here").
			Confirm()
		t.ExpectPopup().Prompt().
			Title(Contains("Local branch name at")).
			InitialText(Equals("feature")).
			Confirm()

		remoteBranches.IsFocused()
		// The branch is created by a worker: wait for it with an eventual check.
		t.Shell().RunShellCommand("git rev-parse refs/heads/feature > .git/placement-feature-tip")
		t.FileSystem().FileContent(".git/placement-feature-tip", Equals(remoteFeatureOID+"\n"))
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Branch placement changed HEAD to " + got)
		}
		t.Git().CurrentBranchName("master")
	},
})
