package worktree

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// Fork: a branch checked out in another worktree cannot be deleted locally.
// The local and compound delete options are disabled and refuse to run, so the
// test checks that nothing is deleted or removed: the local branch, the remote
// branch and the linked worktree all remain. Upstream removed all three here.
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
				menu := t.Common().ReviewBranchDeleteMenu("Delete branch 'mybranch'?", "refs/heads/mybranch", branchOID, "refs/heads/master", t.Git().GetCommitHash("refs/heads/master"), true)
				// Confirming the disabled compound option must not reach a confirmation;
				// it reports the reservation in a toast, which must be acknowledged.
				menu.Select(MatchesRegexp(`^[[:space:]]*b → Delete local and remote branch`)).Confirm()
				t.ExpectToast(Contains("reserved by checkout"))
				t.Views().Confirmation().IsInvisible()
				menu.Cancel()
			}).
			Lines(
				Contains("master"),
				Contains("mybranch (worktree linked-worktree)").IsSelected(),
			)

		t.Common().AssertRefOID("HEAD", headOID).AssertRefOID("refs/heads/mybranch", branchOID)
		t.FileSystem().FileContent("../linked-worktree/README.md", Equals("initial bytes"))
		t.Shell().RunShellCommand("test \"$(git -C ../linked-worktree rev-parse --abbrev-ref HEAD)\" = mybranch")

		// The remote branch and linked worktree remain present.
		t.Views().Remotes().Focus().Lines(Contains("origin")).PressEnter()
		t.Views().RemoteBranches().Lines(Contains("mybranch"))
		t.Common().AssertRefOID("refs/remotes/origin/mybranch", remoteOID)
		t.Shell().RunShellCommand("test \"$(git --git-dir=../origin rev-parse refs/heads/mybranch)\" = \"" + remoteOID + "\"")
		t.Views().Worktrees().Focus().NavigateToLine(Contains("linked-worktree")).Lines(
			Contains("(main worktree)"),
			Contains("linked-worktree").IsSelected(),
		)
		t.FileSystem().PathPresent("../linked-worktree")
	},
})
