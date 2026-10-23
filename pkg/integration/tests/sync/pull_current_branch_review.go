package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullCurrentBranchReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fetch and review an incoming commit before integrating the checked-out branch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.RunCommand([]string{"git", "branch", "topic", "master"})
		shell.RunCommand([]string{"git", "config", "branch.master.remote", "origin"})
		shell.RunCommand([]string{"git", "config", "branch.master.merge", "refs/heads/master"})
		shell.NewBranch("publisher")
		shell.CreateFileAndAdd("incoming-file", "incoming from origin\n").Commit("incoming master")
		shell.RunCommand([]string{"git", "push", "origin", "HEAD:refs/heads/master"})
		shell.Checkout("master")
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/master", "refs/heads/master"})
		shell.SetConfig("pull.ff", "only")
		shell.CreateFileAndAdd("tracked", "staged\n")
		shell.CreateFile("tracked", "unstaged\n")
		shell.CreateFile("untracked-binary", "\x00\x01binary\xff")
		shell.RunShellCommand("cp .git/config .git/pull-current-branch-review-config")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		incomingOID := t.Git().GetCommitHash("refs/heads/publisher")
		cachedOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		stagedBlobOID := t.Git().GetCommitHash(":tracked")
		if cachedOID != masterOID {
			t.Fail("Fixture did not leave the current branch tracking cache at its old tip")
		}

		assertBeforeIntegration := func() {
			if got := t.Git().GetCommitHash("HEAD"); got != headOID {
				t.Fail("Fetching or reviewing changed the checked-out branch")
			}
			if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID {
				t.Fail("Fetching or reviewing advanced master before integration confirmation")
			}
			if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
				t.Fail("Fetching or reviewing changed the index")
			}
			t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
			t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
			t.Shell().RunShellCommand("test ! -e incoming-file")
		}

		t.Views().Files().Focus().Press(keys.Universal.Pull)
		fetchMenu := t.ExpectPopup().Menu().
			Title(Equals("Pull checked-out branch")).
			Select(Contains("Fetch upstream for review"))
		assertBeforeIntegration()
		fetchMenu.Confirm()

		t.ExpectPopup().Confirmation().
			Title(Equals("Review upstream fetch")).
			Content(Contains("refs/remotes/origin/master")).
			Confirm()

		integrationMenu := t.ExpectPopup().Menu().Title(Equals("Integrate reviewed incoming commit"))
		assertBeforeIntegration()
		if got := t.Git().GetCommitHash("refs/remotes/origin/master"); got != incomingOID {
			t.Fail("Approved fetch did not update the reviewed tracking ref to the incoming commit")
		}
		t.Shell().RunShellCommand("cmp .git/config .git/pull-current-branch-review-config")
		integrationMenu.Select(Contains("Use Git's configured pull method")).Confirm()

		review := t.ExpectPopup().Confirmation().
			Title(Equals("Review current-branch integration")).
			Content(Contains(incomingOID[:7]))
		assertBeforeIntegration()
		t.Shell().RunShellCommand("cmp .git/config .git/pull-current-branch-review-config")
		review.Confirm()
		t.Views().Commits().EventuallyContent(Contains("incoming master"))

		if got := t.Git().GetCommitHash("HEAD"); got != incomingOID {
			t.Fail("Confirming current-branch integration did not fast-forward to the reviewed commit")
		}
		if got := t.Git().GetCommitHash("refs/heads/publisher"); got != incomingOID {
			t.Fail("Current-branch integration changed the unrelated publisher branch")
		}
		if got := t.Git().GetCommitHash("refs/heads/topic"); got != topicOID {
			t.Fail("Current-branch integration changed an unrelated local branch tip")
		}
		if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
			t.Fail("Current-branch integration changed the staged tracked content")
		}
		t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
		t.FileSystem().FileContent("incoming-file", Equals("incoming from origin\n"))
		t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
		t.Shell().RunShellCommand("cmp .git/config .git/pull-current-branch-review-config")
	},
})
