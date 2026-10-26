package worktree

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardWorktreeBranchShouldNotPolluteCurrentWorktree = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward the checked out branch without a prompt and refuse an occupied linked-worktree branch without changing either worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.NewBranch("mybranch")
		shell.CreateFileAndAdd("README.md", "hello world")
		shell.Commit("initial commit")
		shell.EmptyCommit("two")
		shell.EmptyCommit("three")
		shell.NewBranch("newbranch")
		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("mybranch", "origin/mybranch")
		shell.SetBranchUpstream("newbranch", "origin/newbranch")
		shell.HardReset("HEAD^")
		shell.Checkout("mybranch")
		shell.HardReset("HEAD^")
		shell.AddWorktreeCheckout("newbranch", "../linked-worktree")
		shell.CreateFile("../linked-worktree/mine", "mine")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		oldMybranch := t.Git().GetCommitHash("mybranch")
		mybranchTarget := t.Git().GetCommitHash("refs/remotes/origin/mybranch")
		t.Views().Branches().Focus().NavigateToLine(Contains("mybranch")).Press(keys.Branches.FastForward)
		t.Views().Branches().ContainsLines(Contains("mybranch").Contains("✓").IsSelected())
		if got := t.Git().GetCommitHash("mybranch"); got != mybranchTarget || t.Git().GetCommitHash("HEAD") != mybranchTarget {
			t.Fail("current-branch fast-forward did not advance mybranch from " + oldMybranch)
		}
		mainHead := t.Git().GetCommitHash("HEAD")
		t.FileSystem().FileContent("README.md", Equals("hello world"))
		t.Shell().RunShellCommand("cp .git/index .git/fast-forward-no-pollution-main-index && cp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-no-pollution-linked-index && cp README.md .git/fast-forward-no-pollution-main-file && cp ../linked-worktree/README.md .git/fast-forward-no-pollution-linked-file")

		oldBranch := t.Git().GetCommitHash("newbranch")
		newbranchTarget := t.Git().GetCommitHash("refs/remotes/origin/newbranch")
		t.Views().Branches().Focus().NavigateToLine(Contains("newbranch")).Press(keys.Branches.FastForward)
		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).Content(Contains("newbranch: ").Contains("branch \"newbranch\" is reserved by another worktree").Contains("cannot be moved here")).Confirm()
		if got := t.Git().GetCommitHash("refs/remotes/origin/newbranch"); got != newbranchTarget {
			t.Fail("fast-forward changed newbranch's tracking ref")
		}
		if got := t.Git().GetCommitHash("newbranch"); got != oldBranch {
			t.Fail("occupied branch moved after refusal")
		}
		if t.Git().GetCommitHash("HEAD") != mainHead {
			t.Fail("refusing to move the occupied branch changed current HEAD or index")
		}
		t.Shell().RunShellCommand("test \"$(git -C ../linked-worktree rev-parse HEAD)\" = \"$(git rev-parse refs/heads/newbranch)\" && cmp .git/index .git/fast-forward-no-pollution-main-index && cmp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-no-pollution-linked-index && cmp README.md .git/fast-forward-no-pollution-main-file && cmp ../linked-worktree/README.md .git/fast-forward-no-pollution-linked-file && git -C ../linked-worktree diff --cached --quiet && test \"$(git -C ../linked-worktree status --porcelain --untracked-files=all)\" = '?? mine'")
		t.Views().Branches().ContainsLines(Contains("newbranch (worktree linked-worktree)").Contains("↓1").IsSelected())
	},
})
