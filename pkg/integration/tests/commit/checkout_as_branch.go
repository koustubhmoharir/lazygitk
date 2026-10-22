package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CheckoutAsBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Check out a selected commit under a new or existing local branch name while carrying safe dirty changes",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("changed.txt", "commit-01\n").
			CreateFileAndAdd("carry.txt", "carry baseline\n").
			Commit("commit-01").
			UpdateFileAndAdd("changed.txt", "commit-02\n").
			Commit("commit-02").
			NewBranch("topic").
			Checkout("master").
			UpdateFileAndAdd("changed.txt", "commit-03\n").
			Commit("commit-03").
			RunCommand([]string{"git", "config", "branch.topic.remote", "."}).
			RunCommand([]string{"git", "config", "branch.topic.merge", "refs/heads/master"}).
			UpdateFileAndAdd("carry.txt", "staged\n").
			UpdateFile("carry.txt", "working\n").
			CreateFile("loosefile", "loose bytes\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits()
		branches := t.Views().Branches()
		reflog := t.Views().ReflogCommits()
		oldestOID := t.Git().GetCommitHash("HEAD~2")
		middleOID := t.Git().GetCommitHash("HEAD~1")
		latestOID := t.Git().GetCommitHash("HEAD")
		baselineIndex := func() {
			t.Shell().RunShellCommand("git show :carry.txt > .git/named-checkout-index")
			t.FileSystem().FileContent(".git/named-checkout-index", Equals("staged\n"))
		}
		assertDirtyBytes := func() {
			t.FileSystem().
				FileContent("carry.txt", Equals("working\n")).
				FileContent("loosefile", Equals("loose bytes\n"))
			baselineIndex()
		}
		assertNoStash := func() {
			t.Shell().RunShellCommand("git stash list > .git/named-checkout-stashes")
			t.FileSystem().FileContent(".git/named-checkout-stashes", Equals(""))
		}
		assertBranchOID := func(name, oid string) {
			t.Shell().RunShellCommand("git rev-parse refs/heads/" + name + " > .git/named-checkout-ref")
			t.FileSystem().FileContent(".git/named-checkout-ref", Equals(oid+"\n"))
		}
		openNamedCheckout := func(name, sourceOID string) {
			t.ExpectPopup().Prompt().
				Title(Equals("Local branch name for checkout at " + sourceOID[:8])).
				Type(name).
				Confirm()
		}
		selectOldestCommit := func() {
			commits.Focus().NavigateToLine(Contains("commit-01"))
			commits.Press(keys.Commits.CheckoutCommit)
			t.ExpectPopup().Menu().
				Title(Equals("Checkout branch or commit")).
				Select(Contains("Checkout as branch")).
				Confirm()
		}
		assertCurrent := func(name, oid string) {
			t.Git().CurrentBranchName(name)
			if got := t.Git().GetCommitHash("HEAD"); got != oid {
				t.Fail("Expected HEAD " + oid + ", got " + got)
			}
		}
		assertTopicUpstream := func() {
			t.Shell().RunShellCommand("git config --get branch.topic.remote > .git/named-checkout-upstream-remote && git config --get branch.topic.merge > .git/named-checkout-upstream-merge")
			t.FileSystem().
				FileContent(".git/named-checkout-upstream-remote", Equals(".\n")).
				FileContent(".git/named-checkout-upstream-merge", Equals("refs/heads/master\n"))
		}
		assertNoStash()

		selectOldestCommit()
		openNamedCheckout("named", oldestOID)
		branches.IsFocused()
		branches.SelectedLine(Contains("named"))
		assertCurrent("named", oldestOID)
		t.FileSystem().FileContent("changed.txt", Equals("commit-01\n"))
		assertDirtyBytes()
		assertBranchOID("named", oldestOID)
		assertBranchOID("topic", middleOID)

		// The reflog selection captures the original commit-03 OID before the
		// branch destination is reviewed; confirmation must apply that selection.
		reflog.Focus().NavigateToLine(Contains("commit: commit-03"))
		reflog.Press(keys.Commits.CheckoutCommit)
		t.ExpectPopup().Menu().
			Title(Equals("Checkout branch or commit")).
			Select(Contains("Checkout as branch")).
			Confirm()
		openNamedCheckout("topic", latestOID)
		confirmation := t.ExpectPopup().Confirmation()
		confirmation.
			Title(Equals("Move branch and check out?")).
			Content(Contains("topic").Contains(middleOID[:7]).Contains(latestOID[:7]))
		confirmation.Cancel()
		reflog.IsFocused()
		assertCurrent("named", oldestOID)
		assertBranchOID("named", oldestOID)
		assertBranchOID("topic", middleOID)
		assertDirtyBytes()
		assertNoStash()

		reflog.Focus().NavigateToLine(Contains("commit: commit-03"))
		reflog.Press(keys.Commits.CheckoutCommit)
		t.ExpectPopup().Menu().Title(Equals("Checkout branch or commit")).Select(Contains("Checkout as branch")).Confirm()
		openNamedCheckout("topic", latestOID)
		confirmation = t.ExpectPopup().Confirmation()
		confirmation.
			Title(Equals("Move branch and check out?")).
			Content(Contains("topic").Contains(middleOID[:7]).Contains(latestOID[:7]))
		confirmation.Confirm()
		branches.IsFocused()
		branches.SelectedLine(Contains("topic"))
		assertCurrent("topic", latestOID)
		t.FileSystem().FileContent("changed.txt", Equals("commit-03\n"))
		assertDirtyBytes()
		assertBranchOID("named", oldestOID)
		assertBranchOID("topic", latestOID)
		assertTopicUpstream()
		assertNoStash()

		// Moving the current destination back to commit-01 requires the same
		// fresh review, and leaves the branch checked out after the combined op.
		commits.Focus().NavigateToLine(Contains("commit-01"))
		commits.Press(keys.Commits.CheckoutCommit)
		t.ExpectPopup().Menu().Title(Equals("Checkout branch or commit")).Select(Contains("Checkout as branch")).Confirm()
		openNamedCheckout("topic", oldestOID)
		confirmation = t.ExpectPopup().Confirmation()
		confirmation.
			Title(Equals("Move branch and check out?")).
			Content(Contains("topic").Contains(latestOID[:7]).Contains(oldestOID[:7]))
		confirmation.Confirm()
		branches.IsFocused()
		branches.SelectedLine(Contains("topic"))
		assertCurrent("topic", oldestOID)
		t.FileSystem().FileContent("changed.txt", Equals("commit-01\n"))
		assertDirtyBytes()
		assertTopicUpstream()
		assertNoStash()

		// At the selected tip, attaching the already existing branch has no
		// pointer-move warning, but still performs a branch checkout.
		selectOldestCommit()
		openNamedCheckout("named", oldestOID)
		branches.IsFocused()
		branches.SelectedLine(Contains("named"))
		assertCurrent("named", oldestOID)
		assertDirtyBytes()
		assertBranchOID("named", oldestOID)
		assertBranchOID("topic", oldestOID)
		assertTopicUpstream()
		assertNoStash()
	},
})
