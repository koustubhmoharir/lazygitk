package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PlaceAtCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Create or move a branch to a selected commit without changing the worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("tracked.txt", "commit one\n").
			Commit("commit-01").
			UpdateFileAndAdd("tracked.txt", "commit two\n").
			Commit("commit-02").
			NewBranch("topic").
			Checkout("master").
			UpdateFileAndAdd("tracked.txt", "commit three\n").
			Commit("commit-03").
			UpdateFileAndAdd("tracked.txt", "staged bytes\n").
			UpdateFile("tracked.txt", "unstaged bytes\n").
			CreateFile("untracked.txt", "untracked bytes\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits()
		branches := t.Views().Branches()
		targetOID := t.Git().GetCommitHash("HEAD~2")
		oldTopicOID := t.Git().GetCommitHash("topic")
		headOID := t.Git().GetCommitHash("HEAD")

		selectTarget := func() {
			commits.Focus().NavigateToLine(Contains("commit-01"))
		}
		openPlacement := func() {
			commits.Press(keys.Universal.OptionMenu)
			t.ExpectPopup().Menu().
				Title(Equals("Keybindings")).
				Filter("Create/move branch here").
				Confirm()
		}
		enterBranchName := func(name string) {
			t.ExpectPopup().Prompt().
				Title(Contains("Local branch name at")).
				Type(name).
				Confirm()
		}
		assertWorktreeUnchanged := func() {
			t.Git().
				CurrentBranchName("master")
			if actual := t.Git().GetCommitHash("HEAD"); actual != headOID {
				t.Fail("Branch placement changed HEAD")
			}
			t.FileSystem().
				FileContent("tracked.txt", Equals("unstaged bytes\n")).
				FileContent("untracked.txt", Equals("untracked bytes\n"))
			t.Shell().RunShellCommand("git show :tracked.txt > .git/placement-index-content")
			t.FileSystem().FileContent(".git/placement-index-content", Equals("staged bytes\n"))
		}
		assertBranchTip := func(name, expected string) {
			path := ".git/placement-" + name + "-tip"
			t.Shell().RunShellCommand("git rev-parse refs/heads/" + name + " > " + path)
			t.FileSystem().FileContent(path, Equals(expected+"\n"))
		}
		assertBranchReflogTip := func(name, expected string) {
			path := ".git/placement-" + name + "-reflog"
			t.Shell().RunShellCommand("git reflog -1 --format=%H refs/heads/" + name + " > " + path)
			t.FileSystem().FileContent(path, Equals(expected+"\n"))
		}

		selectTarget()
		openPlacement()
		enterBranchName("placed")
		branches.EventuallyContent(Contains("placed"))
		assertBranchTip("placed", targetOID)
		assertBranchReflogTip("placed", targetOID)
		assertBranchTip("topic", oldTopicOID)
		assertWorktreeUnchanged()
		commits.IsFocused().SelectedLine(Contains("commit-01"))

		selectTarget()
		openPlacement()
		enterBranchName("topic")
		t.ExpectPopup().Confirmation().
			Title(Equals("Move existing branch?")).
			Content(Contains("topic").Contains(oldTopicOID[:7]).Contains(targetOID[:7]).Contains("does not check out the branch")).
			Cancel()
		assertBranchTip("topic", oldTopicOID)
		assertBranchReflogTip("topic", oldTopicOID)
		assertWorktreeUnchanged()
		commits.IsFocused().SelectedLine(Contains("commit-01"))

		selectTarget()
		openPlacement()
		enterBranchName("topic")
		t.ExpectPopup().Confirmation().
			Title(Equals("Move existing branch?")).
			Content(Contains("topic").Contains(oldTopicOID[:7]).Contains(targetOID[:7]).Contains("does not check out the branch")).
			Confirm()
		commits.IsFocused()
		assertBranchTip("topic", targetOID)
		assertBranchReflogTip("topic", targetOID)
		assertWorktreeUnchanged()
		commits.IsFocused().SelectedLine(Contains("commit-01"))

		selectTarget()
		openPlacement()
		enterBranchName("topic")
		t.ExpectPopup().Alert().
			Title(Equals("Branch already at selected commit")).
			Content(Contains("topic").Contains(targetOID[:7]).Contains("Nothing changed")).
			Confirm()
		assertBranchTip("topic", targetOID)
		assertBranchReflogTip("topic", targetOID)
		assertWorktreeUnchanged()
		commits.IsFocused().SelectedLine(Contains("commit-01"))
	},
})
