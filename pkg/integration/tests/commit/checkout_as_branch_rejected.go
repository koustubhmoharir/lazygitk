package commit

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CheckoutAsBranchRejected = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Reject named checkout when Git cannot safely update files and refresh after a failing checkout hook",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("changed.txt", "old\n").
			CreateFileAndAdd("blocked.txt", "tracked in history\n").
			Commit("commit-01").
			UpdateFileAndAdd("changed.txt", "new\n").
			Commit("commit-02").
			RunCommand([]string{"git", "branch", "topic"}).
			Checkout("master").
			DeleteFileAndAdd("blocked.txt").
			Commit("commit-03").
			RunCommand([]string{"git", "config", "--local", "core.excludesFile", ".git/info/exclude"}).
			CreateFile(".git/info/exclude", "blocked.txt\n").
			CreateFile("blocked.txt", "ignored collision bytes\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		shell := t.Shell()
		commits := t.Views().Commits()
		branches := t.Views().Branches()
		targetOID := t.Git().GetCommitHash("HEAD~2")
		topicOID := t.Git().GetCommitHash("topic")
		headOID := t.Git().GetCommitHash("HEAD")

		shell.RunShellCommand("git rev-parse HEAD > .git/named-checkout-original-head").
			RunShellCommand("git show :changed.txt > .git/named-checkout-original-index").
			RunShellCommand("git status --porcelain=v1 > .git/named-checkout-original-status")

		selectTarget := func() {
			commits.Focus().NavigateToLine(Contains("commit-01"))
		}
		openNamedCheckout := func() {
			selectTarget()
			commits.Press(keys.Commits.CheckoutCommit)
			t.ExpectPopup().Menu().
				Title(Equals("Checkout branch or commit")).
				Select(Contains("Checkout as branch")).
				Confirm()
		}
		enterName := func(name string) {
			t.ExpectPopup().Prompt().
				Title(Contains("Local branch name for checkout")).
				Type(name).
				Confirm()
		}
		assertOriginalCheckoutState := func() {
			t.Git().CurrentBranchName("master")
			if got := t.Git().GetCommitHash("HEAD"); got != headOID {
				t.Fail("Rejected named checkout moved HEAD")
			}
			shell.RunShellCommand("git show :changed.txt > .git/named-checkout-index").
				RunShellCommand("git status --porcelain=v1 > .git/named-checkout-status")
			t.FileSystem().
				FileContent(".git/named-checkout-original-head", Equals(headOID+"\n")).
				FileContent(".git/named-checkout-original-index", Equals("new\n")).
				FileContent(".git/named-checkout-original-status", Equals("")).
				FileContent(".git/named-checkout-index", Equals("new\n")).
				FileContent(".git/named-checkout-status", Equals(""))
		}
		assertNoStash := func() {
			shell.RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/stash"})
		}
		assertBranchAbsent := func(name string) {
			shell.RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/heads/" + name})
		}

		openNamedCheckout()
		enterName("rejected-new")
		alert := t.ExpectPopup().Alert()
		alert.Title(Equals("Error")).Content(Contains("blocked.txt").Contains("may already have changed"))
		alert.Confirm()
		assertOriginalCheckoutState()
		assertBranchAbsent("rejected-new")
		t.FileSystem().FileContent("blocked.txt", Equals("ignored collision bytes\n"))
		assertNoStash()
		commits.IsFocused().SelectedLine(Contains("commit-01"))

		// This ignored file was created solely to exercise the overwrite guard.
		if err := os.Remove("blocked.txt"); err != nil {
			t.Fail("Removing ignored checkout collision fixture: " + err.Error())
			return
		}
		shell.UpdateFileAndAdd("changed.txt", "staged local bytes\n").
			UpdateFile("changed.txt", "working local bytes\n")
		openNamedCheckout()
		enterName("topic")
		moveConfirmation := t.ExpectPopup().Confirmation()
		moveConfirmation.Title(Equals("Move branch and check out?")).
			Content(Contains("topic").Contains(topicOID[:7]).Contains(targetOID[:7])).
			Confirm()
		alert = t.ExpectPopup().Alert()
		alert.Title(Equals("Error")).Content(Contains("changed.txt"))
		alert.Confirm()
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Rejected existing-branch checkout moved HEAD")
		}
		if got := t.Git().GetCommitHash("topic"); got != topicOID {
			t.Fail("Rejected existing-branch checkout moved the destination branch")
		}
		shell.RunShellCommand("git show :changed.txt > .git/named-checkout-dirty-index")
		t.FileSystem().
			FileContent("changed.txt", Equals("working local bytes\n")).
			FileContent(".git/named-checkout-dirty-index", Equals("staged local bytes\n"))
		assertNoStash()
		commits.IsFocused().SelectedLine(Contains("commit-01"))

		// Restore only the dirty test fixture before exercising the post-checkout hook.
		shell.RunCommand([]string{"git", "restore", "--staged", "--worktree", "--", "changed.txt"})
		shell.CreateFile(".git/hooks/post-checkout", "#!/bin/sh\nprintf 'run\\n' >> \"$(git rev-parse --absolute-git-dir)/hook-count\"\necho 'named checkout hook failed: index.lock' >&2\nexit 7\n")
		shell.MakeExecutable(".git/hooks/post-checkout")
		openNamedCheckout()
		enterName("hooked")
		alert = t.ExpectPopup().Alert()
		alert.Title(Equals("Error")).Content(Contains("named checkout hook failed: index.lock"))
		alert.Confirm()
		if got := t.Git().GetCommitHash("HEAD"); got != targetOID {
			t.Fail("Failed post-checkout hook did not leave HEAD at the successfully checked out target")
		}
		t.Git().CurrentBranchName("hooked")
		t.FileSystem().
			FileContent("changed.txt", Equals("old\n")).
			FileContent("blocked.txt", Equals("tracked in history\n")).
			FileContent(".git/hook-count", Equals("run\n"))
		if got := t.Git().GetCommitHash("hooked"); got != targetOID {
			t.Fail("Hook failure left the new branch at the wrong commit")
		}
		branches.SelectedLine(Contains("hooked"))
		commits.IsFocused()
		assertNoStash()
	},
})
