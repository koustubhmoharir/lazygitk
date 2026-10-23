package worktree

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DetachWorktreeFromBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to delete a branch that's checked out in another worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.NewBranch("mybranch")
		shell.CreateFileAndAdd("README.md", "hello world")
		shell.Commit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.EmptyCommit("commit 3")
		shell.AddWorktree("mybranch", "../linked-worktree", "newbranch")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branchOID := t.Git().GetCommitHash("refs/heads/newbranch")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("mybranch").IsSelected(),
				Contains("newbranch (worktree linked-worktree)"),
			).
			NavigateToLine(Contains("newbranch")).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.Common().ReviewBranchDeleteMenu("Delete branch 'newbranch'?", "refs/heads/newbranch", branchOID, "refs/heads/mybranch", t.Git().GetCommitHash("refs/heads/mybranch"), true).Cancel()
			}).
			Lines(
				Contains("mybranch"),
				Contains("newbranch (worktree linked-worktree)").IsSelected(),
			)
		t.FileSystem().FileContent("../linked-worktree/README.md", Equals("hello world"))
		t.Common().AssertRefOID("HEAD", headOID).AssertRefOID("refs/heads/newbranch", branchOID)

		t.Views().Worktrees().
			Focus().
			NavigateToLine(Contains("linked-worktree")).
			Lines(
				Contains("(main worktree)"),
				Contains("linked-worktree").IsSelected(),
			)
	},
})
