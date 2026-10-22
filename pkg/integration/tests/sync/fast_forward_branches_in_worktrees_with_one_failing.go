package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardBranchesInWorktreesWithOneFailing = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse a selected-branch fast-forward batch when its first branch is occupied by another worktree",
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
		shell.AddWorktreeCheckout("branch-b", "../worktree-b")
		shell.CreateFile("../worktree-a/a-file", "mine")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branchA := t.Git().GetCommitHash("branch-a")
		branchB := t.Git().GetCommitHash("branch-b")
		targetA := t.Git().GetCommitHash("refs/remotes/origin/branch-a")
		t.Shell().RunShellCommand("cp .git/index .git/fast-forward-multi-main-index && cp \"$(git -C ../worktree-a rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-multi-a-index && cp \"$(git -C ../worktree-b rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-multi-b-index && cp file .git/fast-forward-multi-main-file && cp ../worktree-a/file .git/fast-forward-multi-a-file && cp ../worktree-b/file .git/fast-forward-multi-b-file")
		t.Views().Branches().Focus().Lines(
			Contains("master").IsSelected(),
			Contains("branch-a (worktree worktree-a) ↓1"),
			Contains("branch-b (worktree worktree-b) ↓1"),
		).SelectNextItem().Press(keys.Universal.ToggleRangeSelect).SelectNextItem().Press(keys.Branches.FastForward)
		t.ExpectPopup().Menu().Title(Equals("Review selected branch fast-forward")).Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("branch-a", "origin", "branch-a")
		t.ExpectPopup().Alert().Title(Equals("Error")).Content(Contains("branch \"branch-a\" is reserved by another worktree").Contains("cannot be moved here")).Confirm()
		if got := t.Git().GetCommitHash("refs/remotes/origin/branch-a"); got != targetA {
			t.Fail("approved fetch did not update branch-a's tracking ref to the reviewed tip")
		}
		if t.Git().GetCommitHash("branch-a") != branchA || t.Git().GetCommitHash("branch-b") != branchB {
			t.Fail("refused batch fast-forward moved a local branch")
		}
		t.Views().Branches().Lines(
			Contains("master"),
			Contains("branch-a (worktree worktree-a) ↓1").IsSelected(),
			Contains("branch-b (worktree worktree-b) ↓1").IsSelected(),
		)
		t.Shell().RunShellCommand("test \"$(git -C ../worktree-a rev-parse HEAD)\" = \"$(git rev-parse refs/heads/branch-a)\" && test \"$(git -C ../worktree-b rev-parse HEAD)\" = \"$(git rev-parse refs/heads/branch-b)\" && cmp .git/index .git/fast-forward-multi-main-index && cmp \"$(git -C ../worktree-a rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-multi-a-index && cmp \"$(git -C ../worktree-b rev-parse --path-format=absolute --git-path index)\" .git/fast-forward-multi-b-index && cmp file .git/fast-forward-multi-main-file && cmp ../worktree-a/file .git/fast-forward-multi-a-file && cmp ../worktree-b/file .git/fast-forward-multi-b-file && git -C ../worktree-a diff --cached --quiet && git -C ../worktree-b diff --cached --quiet && test \"$(git -C ../worktree-a status --porcelain --untracked-files=all)\" = '?? a-file' && test -z \"$(git -C ../worktree-b status --porcelain --untracked-files=all)\"")
		t.FileSystem().FileContent("../worktree-a/a-file", Equals("mine"))
	},
})
