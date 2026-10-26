package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardBranchesInWorktreesWithOneFailing = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward the unoccupied branch of a selection and list the branch occupied by another worktree as not fast-forwarded",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "content")
		shell.Commit("one")
		shell.NewBranch("branch-a")
		shell.CreateFileAndAdd("a-file", "a")
		shell.Commit("a")
		shell.Checkout("master")
		shell.NewBranch("branch-b")
		shell.CreateFileAndAdd("b-file", "b")
		shell.Commit("b")
		shell.Checkout("master")
		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("branch-a", "origin/branch-a")
		shell.SetBranchUpstream("branch-b", "origin/branch-b")
		shell.RunCommand([]string{"git", "branch", "-f", "branch-a", "branch-a~1"})
		shell.RunCommand([]string{"git", "branch", "-f", "branch-b", "branch-b~1"})
		shell.AddWorktreeCheckout("branch-a", "../worktree-a")
		shell.CreateFile("../worktree-a/a-file", "mine")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branchA := t.Git().GetCommitHash("branch-a")
		targetB := t.Git().GetCommitHash("refs/remotes/origin/branch-b")
		t.Shell().RunShellCommand("cp .git/index .git/fast-forward-multi-main-index && cp \"$(git -C ../worktree-a rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-multi-a-index && cp file .git/fast-forward-multi-main-file && cp ../worktree-a/file .git/fast-forward-multi-a-file")
		t.Views().Branches().Focus().Lines(
			Contains("master").IsSelected(),
			Contains("branch-a (worktree worktree-a) ↓1"),
			Contains("branch-b ↓1"),
		).SelectNextItem().Press(keys.Universal.ToggleRangeSelect).SelectNextItem().Press(keys.Branches.FastForward)
		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).Content(Contains("branch-a: ").Contains("branch \"branch-a\" is reserved by another worktree").Contains("cannot be moved here").DoesNotContain("branch-b")).Confirm()

		if got := t.Git().GetCommitHash("refs/heads/branch-b"); got != targetB {
			t.Fail("unoccupied branch-b was not fast-forwarded to its upstream: " + got)
		}
		if t.Git().GetCommitHash("branch-a") != branchA {
			t.Fail("occupied branch-a moved after refusal")
		}
		t.Views().Branches().Lines(
			Contains("master"),
			Contains("branch-a (worktree worktree-a) ↓1").IsSelected(),
			Contains("branch-b ✓").IsSelected(),
		)
		t.Shell().RunShellCommand("test \"$(git -C ../worktree-a rev-parse HEAD)\" = \"$(git rev-parse refs/heads/branch-a)\" && cmp .git/index .git/fast-forward-multi-main-index && cmp \"$(git -C ../worktree-a rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-multi-a-index && cmp file .git/fast-forward-multi-main-file && cmp ../worktree-a/file .git/fast-forward-multi-a-file && git -C ../worktree-a diff --cached --quiet && test \"$(git -C ../worktree-a status --porcelain --untracked-files=all)\" = '?? a-file'")
		t.FileSystem().FileContent("../worktree-a/a-file", Equals("mine"))
	},
})
