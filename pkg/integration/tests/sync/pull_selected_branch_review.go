package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullSelectedBranchReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fetch and fast-forward a selected branch without checking it out",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("topic")
		shell.RunCommand([]string{"git", "push", "origin", "refs/heads/topic:refs/heads/topic"})
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/topic", "refs/heads/topic"})
		shell.RunCommand([]string{"git", "config", "branch.topic.remote", "origin"})
		shell.RunCommand([]string{"git", "config", "branch.topic.merge", "refs/heads/topic"})
		shell.Checkout("master")
		shell.EmptyCommit("master-independent-tip")
		shell.NewBranchFrom("publisher", "refs/heads/topic")
		shell.CreateFileAndAdd("incoming-file", "incoming from origin\n").Commit("incoming topic")
		shell.RunCommand([]string{"git", "push", "origin", "HEAD:refs/heads/topic"})
		shell.Checkout("master")
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/topic", "refs/heads/topic"})
		shell.SetConfig("pull.ff", "only")
		shell.CreateFileAndAdd("tracked", "staged\n")
		shell.CreateFile("tracked", "unstaged\n")
		shell.CreateFile("untracked-binary", "\x00\x01binary\xff")
		shell.RunShellCommand("cp .git/config .git/pull-selected-branch-review-config")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		incomingOID := t.Git().GetCommitHash("refs/heads/publisher")
		cachedOID := t.Git().GetCommitHash("refs/remotes/origin/topic")
		stagedBlobOID := t.Git().GetCommitHash(":tracked")
		if cachedOID != topicOID {
			t.Fail("Fixture did not leave the selected branch tracking cache at its old tip")
		}

		assertBeforeAdvance := func() {
			if got := t.Git().GetCommitHash("HEAD"); got != headOID {
				t.Fail("Updating the selected branch changed HEAD")
			}
			if got := t.Git().GetCommitHash("refs/heads/topic"); got != topicOID {
				t.Fail("The selected branch moved before fast-forward confirmation")
			}
			if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID {
				t.Fail("Updating the selected branch changed master")
			}
			if got := t.Git().GetCommitHash("refs/remotes/origin/master"); got != remoteMasterOID {
				t.Fail("Fetching the selected branch changed origin/master")
			}
			if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
				t.Fail("Fetching or reviewing changed the index")
			}
			t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
			t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
			t.Shell().RunShellCommand("test ! -e incoming-file")
		}

		t.Views().Branches().Focus().NavigateToLine(Contains("topic")).Press(keys.Universal.Pull)
		fetchMenu := t.ExpectPopup().Menu().
			Title(Equals("Update selected branch without checkout")).
			Select(Contains("Fetch upstream for review"))
		assertBeforeAdvance()
		fetchMenu.Confirm()

		t.ExpectPopup().Confirmation().
			Title(Equals("Review upstream fetch")).
			Content(Contains("refs/remotes/origin/topic")).
			Confirm()

		advanceReview := t.ExpectPopup().Confirmation().
			Title(Equals("Review branch fast-forward")).
			Content(Contains(topicOID[:7]).Contains(incomingOID[:7]))
		assertBeforeAdvance()
		if got := t.Git().GetCommitHash("refs/remotes/origin/topic"); got != incomingOID {
			t.Fail("Approved fetch did not update the reviewed tracking ref to the incoming commit")
		}
		t.Shell().RunShellCommand("cmp .git/config .git/pull-selected-branch-review-config")
		advanceReview.Confirm()

		if got := t.Git().GetCommitHash("refs/heads/topic"); got != incomingOID {
			t.Fail("Confirming the fast-forward did not advance topic to the reviewed commit")
		}
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Fast-forwarding the selected branch changed HEAD")
		}
		if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID {
			t.Fail("Fast-forwarding the selected branch changed master")
		}
		if got := t.Git().GetCommitHash("refs/remotes/origin/master"); got != remoteMasterOID {
			t.Fail("Fetching the selected branch changed origin/master")
		}
		if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
			t.Fail("Fast-forwarding the selected branch changed the current index")
		}
		t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
		t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
		t.Shell().RunShellCommand("test ! -e incoming-file")
		t.Shell().RunShellCommand("cmp .git/config .git/pull-selected-branch-review-config")
	},
})
