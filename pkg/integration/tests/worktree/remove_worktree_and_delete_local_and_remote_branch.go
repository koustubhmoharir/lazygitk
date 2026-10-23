package worktree

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RemoveWorktreeAndDeleteLocalAndRemoteBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to delete local and remote refs for a branch checked out in another worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CloneIntoRemote("origin")
		shell.CreateFileAndAdd("README.md", "initial bytes")
		shell.Commit("initial commit")
		shell.NewBranch("mybranch")
		shell.EmptyCommit("commit on mybranch")
		shell.PushBranchAndSetUpstream("origin", "mybranch")
		shell.EmptyCommit("commit not pushed to the remote") // so mybranch isn't fully merged
		shell.Checkout("master")
		shell.AddWorktreeCheckout("mybranch", "../linked-worktree")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branchOID := t.Git().GetCommitHash("refs/heads/mybranch")
		headOID := t.Git().GetCommitHash("HEAD")
		remoteOID := t.Git().GetCommitHash("refs/remotes/origin/mybranch")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("mybranch (worktree linked-worktree)"),
			).
			NavigateToLine(Contains("mybranch")).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.Common().ReviewBranchDeleteMenu("Delete branch 'mybranch'?", "refs/heads/mybranch", branchOID, "refs/heads/master", t.Git().GetCommitHash("refs/heads/master"), true).Cancel()
			}).
			Lines(
				Contains("master"),
				Contains("mybranch (worktree linked-worktree)").IsSelected(),
			)

		t.Common().AssertRefOID("HEAD", headOID).AssertRefOID("refs/heads/mybranch", branchOID)
		t.FileSystem().FileContent("../linked-worktree/README.md", Equals("initial bytes"))

		// The remote branch and linked worktree remain present.
		t.Views().Remotes().Focus().Lines(Contains("origin")).PressEnter()
		t.Views().RemoteBranches().Lines(Contains("mybranch"))
		t.Common().AssertRefOID("refs/remotes/origin/mybranch", remoteOID)
		t.Views().Worktrees().Focus().NavigateToLine(Contains("linked-worktree")).Lines(
			Contains("(main worktree)"),
			Contains("linked-worktree").IsSelected(),
		)
		t.FileSystem().PathPresent("../linked-worktree")
	},
})
