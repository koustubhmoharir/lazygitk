package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

var CheckoutAsBranchChanged = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Review a named checkout again when its branch changes or disappears while confirmation is open",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("changed.txt", "commit-01\n").
			Commit("commit-01").
			UpdateFileAndAdd("changed.txt", "commit-02\n").
			Commit("commit-02").
			RunCommand([]string{"git", "branch", "topic", "HEAD"}).
			RunCommand([]string{"git", "checkout", "--orphan", "otherbranch"}).
			CreateFileAndAdd("changed.txt", "other-10\n").
			Commit("other-10").
			Checkout("master").
			UpdateFileAndAdd("changed.txt", "commit-03\n").
			Commit("commit-03")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branches := t.Views().Branches()
		subCommits := t.Views().SubCommits()
		otherOID := t.Git().GetCommitHash("refs/heads/otherbranch")
		oldTopicOID := t.Git().GetCommitHash("refs/heads/topic")
		headOID := t.Git().GetCommitHash("HEAD")

		assertMasterHead := func() {
			t.Git().CurrentBranchName("master")
			if got := t.Git().GetCommitHash("HEAD"); got != headOID {
				t.Fail("Expected HEAD " + headOID + ", got " + got)
			}
		}
		assertTopicOID := func(expected string) {
			if got := t.Git().GetCommitHash("refs/heads/topic"); got != expected {
				t.Fail("Expected topic at " + expected + ", got " + got)
			}
		}
		selectOtherCommit := func() {
			branches.Focus().NavigateToLine(Contains("otherbranch")).PressEnter()
			subCommits.IsFocused().NavigateToLine(Contains("other-10"))
		}
		openNamedCheckout := func() {
			subCommits.IsFocused().Press(keys.Commits.CheckoutCommit)
			t.ExpectPopup().Menu().
				Title(Equals("Checkout branch or commit")).
				Select(Contains("Checkout as branch")).
				Confirm()
			t.ExpectPopup().Prompt().
				Title(Equals("Local branch name for checkout at " + utils.ShortHash(otherOID))).
				Type("topic").
				Confirm()
		}
		branchMoveConfirmation := func(currentTopicOID string) *ConfirmationDriver {
			confirmation := t.ExpectPopup().Confirmation()
			confirmation.Title(Equals("Move branch and check out?"))
			confirmation.Content(Contains("topic").Contains(currentTopicOID[:7]).Contains(otherOID[:7]))
			return confirmation
		}
		assertTopicAbsent := func() {
			t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/heads/topic"})
		}

		selectOtherCommit()
		openNamedCheckout()
		initialWarning := branchMoveConfirmation(oldTopicOID)
		t.Shell().RunCommand([]string{"git", "update-ref", "refs/heads/topic", headOID})
		initialWarning.Confirm()

		changedWarning := t.ExpectPopup().Confirmation()
		changedWarning.Title(Equals("Checkout changed; review again"))
		changedWarning.Content(Contains("topic").Contains(headOID[:7]).Contains(otherOID[:7]).Contains("Current checkout").Contains("master"))
		changedWarning.Cancel()
		assertMasterHead()
		assertTopicOID(headOID)
		subCommits.IsFocused().SelectedLine(Contains("other-10"))

		openNamedCheckout()
		warningBeforeDelete := branchMoveConfirmation(headOID)
		t.Shell().RunCommand([]string{"git", "update-ref", "-d", "refs/heads/topic"})
		warningBeforeDelete.Confirm()

		createWarning := t.ExpectPopup().Confirmation()
		createWarning.Title(Equals("Checkout changed; review again"))
		createWarning.Content(Contains("topic").Contains("Create and check out").Contains(otherOID[:7]).Contains("Current checkout").Contains("master"))
		assertTopicAbsent()
		assertMasterHead()
		createWarning.Confirm()
		branches.IsFocused()
		t.Git().CurrentBranchName("topic")
		if got := t.Git().GetCommitHash("HEAD"); got != otherOID {
			t.Fail("Expected HEAD " + otherOID + ", got " + got)
		}
		t.FileSystem().FileContent("changed.txt", Equals("other-10\n"))
	},
})
