package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardBranchBeingRebasedInWorktree = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to fast-forward a branch reserved by an in-progress rebase in another worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "content")
		shell.Commit("one")
		shell.EmptyCommit("two")
		shell.EmptyCommit("three")
		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")
		shell.HardReset("HEAD~1")
		shell.NewBranch("other")
		shell.AddWorktreeCheckout("master", "../linked-worktree")
		shell.RunCommandExpectError([]string{"git", "-C", "../linked-worktree", "rebase", "--exec", "false", "HEAD~1"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		oldMaster := t.Git().GetCommitHash("master")
		target := t.Git().GetCommitHash("refs/remotes/origin/master")
		other := t.Git().GetCommitHash("other")
		t.Shell().RunShellCommand("cp .git/index .git/fast-forward-rebase-main-index && cp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-rebase-linked-index && cp file .git/fast-forward-rebase-main-file && cp ../linked-worktree/file .git/fast-forward-rebase-linked-file")
		t.Views().Branches().Focus().NavigateToLine(Contains("master")).Press(keys.Branches.FastForward)
		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).Content(Contains("master: ").Contains("branch \"master\" is reserved by another worktree").Contains("cannot be moved here")).Confirm()
		if got := t.Git().GetCommitHash("refs/remotes/origin/master"); got != target {
			t.Fail("fast-forward changed master's tracking ref")
		}
		if got := t.Git().GetCommitHash("master"); got != oldMaster {
			t.Fail("master moved after refusal during its linked-worktree rebase")
		}
		if t.Git().GetCommitHash("other") != other {
			t.Fail("refusing the occupied branch changed unrelated branch other")
		}
		t.Shell().RunShellCommand("cmp .git/index .git/fast-forward-rebase-main-index && cmp \"$(git -C ../linked-worktree rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-rebase-linked-index && cmp file .git/fast-forward-rebase-main-file && cmp ../linked-worktree/file .git/fast-forward-rebase-linked-file")
		t.Views().Worktrees().Focus().NavigateToLine(Contains("linked-worktree")).Press(keys.Universal.Select)
		t.Views().Information().Content(Contains("Rebasing"))
		t.Views().Commits().Lines(
			Contains("─── Pending rebase todos"),
			Contains("─── Commits"),
			Contains("two"),
			Contains("one"),
		)
	},
})
