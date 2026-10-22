package worktree

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardWorktreeBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward the current branch after review and refuse a selected branch occupied by another worktree",
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
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		oldMybranch := t.Git().GetCommitHash("mybranch")
		mybranchTarget := t.Git().GetCommitHash("refs/remotes/origin/mybranch")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Views().Branches().Focus().Press(keys.Branches.FastForward)
		t.ExpectPopup().Menu().Title(Equals("Review selected branch fast-forward")).Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("mybranch", "origin", "mybranch")
		review := t.ExpectPopup().Confirmation().Title(Equals("Review current-branch integration")).Content(Contains("mybranch").Contains(oldMybranch).Contains(mybranchTarget).Contains("Fast-forward only"))
		if got := t.Git().GetCommitHash("mybranch"); got != oldMybranch || t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("fetching and reviewing changed the current branch before confirmation")
		}
		review.Confirm()
		t.Views().Branches().ContainsLines(Contains("mybranch").Contains("✓").IsSelected())
		if got := t.Git().GetCommitHash("mybranch"); got != mybranchTarget || t.Git().GetCommitHash("HEAD") != mybranchTarget {
			t.Fail("confirming current-branch integration did not fast-forward mybranch")
		}
		t.Shell().RunShellCommand("cp .git/index .git/fast-forward-worktree-branch-main-index && cp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-worktree-branch-linked-index && cp README.md .git/fast-forward-worktree-branch-main-file && cp ../linked-worktree/README.md .git/fast-forward-worktree-branch-linked-file")

		oldNewbranch := t.Git().GetCommitHash("newbranch")
		newbranchTarget := t.Git().GetCommitHash("refs/remotes/origin/newbranch")
		mybranchAfterIntegration := t.Git().GetCommitHash("mybranch")
		t.Views().Branches().Lines(Contains("mybranch").Contains("✓"), Contains("newbranch (worktree linked-worktree)").Contains("↓1"))
		t.Views().Branches().NavigateToLine(Contains("newbranch (worktree linked-worktree)")).Press(keys.Branches.FastForward)
		t.ExpectPopup().Menu().Title(Equals("Review selected branch fast-forward")).Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("newbranch", "origin", "newbranch")
		t.ExpectPopup().Alert().Title(Equals("Error")).Content(Contains("branch \"newbranch\" is reserved by another worktree").Contains("cannot be moved here")).Confirm()
		if got := t.Git().GetCommitHash("refs/remotes/origin/newbranch"); got != newbranchTarget {
			t.Fail("approved fetch did not update newbranch's tracking ref to the reviewed tip")
		}
		if got := t.Git().GetCommitHash("newbranch"); got != oldNewbranch {
			t.Fail("occupied newbranch moved after refusal")
		}
		if t.Git().GetCommitHash("mybranch") != mybranchAfterIntegration {
			t.Fail("refusing the occupied branch changed the unrelated current branch")
		}
		t.Shell().RunShellCommand("cmp .git/index .git/fast-forward-worktree-branch-main-index && cmp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-worktree-branch-linked-index && cmp README.md .git/fast-forward-worktree-branch-main-file && cmp ../linked-worktree/README.md .git/fast-forward-worktree-branch-linked-file && git -C ../linked-worktree diff --cached --quiet && test -z \"$(git -C ../linked-worktree status --porcelain --untracked-files=all)\"")
		t.Views().Branches().Lines(Contains("mybranch").Contains("✓"), Contains("newbranch (worktree linked-worktree)").Contains("↓1").IsSelected())
	},
})
