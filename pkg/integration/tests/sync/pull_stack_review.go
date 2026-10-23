package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullStackReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Review upstream updates for the checked-out branch and its stack before applying them",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("lower").EmptyCommit("lower")
		shell.RunCommand([]string{"git", "push", "origin", "refs/heads/lower:refs/heads/lower"})
		shell.SetBranchUpstream("lower", "origin/lower")
		shell.NewBranch("topic").EmptyCommit("topic")
		shell.RunCommand([]string{"git", "push", "origin", "refs/heads/topic:refs/heads/topic"})
		shell.SetBranchUpstream("topic", "origin/topic")
		shell.NewBranch("publisher")
		shell.CreateFileAndAdd("incoming-file", "incoming from origin\n").Commit("incoming topic")
		shell.RunCommand([]string{"git", "push", "origin", "HEAD:refs/heads/topic"})
		shell.RunCommand([]string{"git", "push", "origin", "refs/heads/topic:refs/heads/lower"})
		shell.Checkout("topic")
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/topic", "refs/heads/topic"})
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/lower", "refs/heads/lower"})
		shell.SetConfig("pull.ff", "only")
		shell.CreateFileAndAdd("tracked", "staged\n")
		shell.CreateFile("tracked", "unstaged\n")
		shell.CreateFile("untracked-binary", "\x00\x01binary\xff")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Shell().RunShellCommand("cp .git/HEAD .git/pull-stack-review-head")
		t.Shell().RunShellCommand("cp .git/index .git/pull-stack-review-index")
		t.Shell().RunShellCommand("cp .git/config .git/pull-stack-review-config")

		headOID := t.Git().GetCommitHash("HEAD")
		baseOID := t.Git().GetCommitHash("refs/heads/master")
		lowerOID := t.Git().GetCommitHash("refs/heads/lower")
		topicOID := t.Git().GetCommitHash("refs/heads/topic")
		publisherOID := t.Git().GetCommitHash("refs/heads/publisher")
		remoteBaseOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		stagedBlobOID := t.Git().GetCommitHash(":tracked")

		assertReviewState := func() {
			if got := t.Git().GetCommitHash("HEAD"); got != headOID {
				t.Fail("Reviewing stack updates changed HEAD")
			}
			if got := t.Git().GetCommitHash("refs/heads/lower"); got != lowerOID {
				t.Fail("Reviewing stack updates moved lower before final approval")
			}
			if got := t.Git().GetCommitHash("refs/heads/topic"); got != topicOID {
				t.Fail("Reviewing stack updates moved topic before final approval")
			}
			if got := t.Git().GetCommitHash("refs/heads/master"); got != baseOID {
				t.Fail("Reviewing stack updates changed the unrelated base branch")
			}
			if got := t.Git().GetCommitHash("refs/heads/publisher"); got != publisherOID {
				t.Fail("Reviewing stack updates changed the publisher branch")
			}
			if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
				t.Fail("Reviewing stack updates changed the index")
			}
			t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
			t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
			t.Shell().RunShellCommand("test ! -e incoming-file")
			t.Shell().RunShellCommand("cmp .git/HEAD .git/pull-stack-review-head")
			t.Shell().RunShellCommand("cmp .git/index .git/pull-stack-review-index")
			t.Shell().RunShellCommand("cmp .git/config .git/pull-stack-review-config")
		}

		checkFetchMenu := func(title, choice, upstream string) {
			fetchMenu := t.ExpectPopup().Menu().Title(Equals(title)).Select(Contains(choice))
			assertReviewState()
			fetchMenu.Confirm()
			fetchReview := t.ExpectPopup().Confirmation().Title(Equals("Review upstream fetch"))
			fetchReview.Content(Contains(upstream))
			assertReviewState()
			fetchReview.Confirm()
		}

		t.Views().Branches().Focus().NavigateToLine(Contains("lower"))
		t.Views().Files().Focus().Press(keys.Universal.Pull)
		currentMenu := t.ExpectPopup().Menu().Title(Equals("Pull checked-out branch"))
		currentMenu.Select(Contains("Review updates for this branch and branches stacked below…")).Confirm()
		checkFetchMenu("Pull checked-out branch", "Fetch upstream for review", "origin/topic")

		integrationMenu := t.ExpectPopup().Menu().Title(Equals("Integrate reviewed incoming commit"))
		assertReviewState()
		if got := t.Git().GetCommitHash("refs/remotes/origin/topic"); got != publisherOID {
			t.Fail("Approved current-branch fetch did not update origin/topic to the incoming commit")
		}
		integrationMenu.Select(Contains("Use Git's configured pull method")).Confirm()
		integrationReview := t.ExpectPopup().Confirmation().
			Title(Equals("Review current-branch integration")).
			Content(Contains(topicOID[:7]).Contains(publisherOID[:7]))
		assertReviewState()
		integrationReview.Confirm()

		lowerFetchMenu := t.ExpectPopup().Menu().
			Title(Equals("Update selected branch without checkout")).
			Select(Contains("Fetch upstream for review"))
		assertReviewState()
		lowerFetchMenu.Confirm()
		lowerFetchReview := t.ExpectPopup().Confirmation().
			Title(Equals("Review upstream fetch")).
			Content(Contains("refs/heads/lower").Contains("origin/lower"))
		assertReviewState()
		lowerFetchReview.Confirm()

		lowerReview := t.ExpectPopup().Confirmation().
			Title(Equals("Review branch fast-forward")).
			Content(Contains(lowerOID[:7]).Contains(topicOID[:7]))
		assertReviewState()
		if got := t.Git().GetCommitHash("refs/remotes/origin/lower"); got != topicOID {
			t.Fail("Approved lower-branch fetch did not update origin/lower to topic")
		}
		lowerReview.Confirm()
		assertReviewState()

		updatesMenu := t.ExpectPopup().Menu().Title(Equals("Review branch updates"))
		assertReviewState()
		updatesMenu.ContainsLines(
			Contains("Cancel").IsSelected(),
		)
		updatesMenu.ContainsLines(
			Contains("lower").Contains(lowerOID[:7]).Contains(topicOID[:7]),
			Contains("topic").Contains(topicOID[:7]).Contains(publisherOID[:7]).Contains("checked-out integration, last"),
		)
		updatesMenu.Select(Contains("lower")).Confirm()
		inspectLower := t.ExpectPopup().Menu().Title(Equals("Review branch updates"))
		inspectLower.ContainsLines(Contains("lower"))
		inspectLower.ContainsLines(Contains(lowerOID[:7]))
		inspectLower.ContainsLines(Contains(topicOID[:7]))
		assertReviewState()
		inspectLower.Select(Contains("Close")).Confirm()

		updatesMenu = t.ExpectPopup().Menu().Title(Equals("Review branch updates"))
		assertReviewState()
		updatesMenu.Select(Contains("Apply all reviewed branch updates")).Confirm()
		t.Views().Commits().EventuallyContent(Contains("incoming topic"))

		if got := t.Git().GetCommitHash("HEAD"); got != publisherOID {
			t.Fail("Applying reviewed stack updates did not integrate the current branch to the fetched commit")
		}
		if got := t.Git().GetCommitHash("refs/heads/topic"); got != publisherOID {
			t.Fail("Applying reviewed stack updates did not advance topic")
		}
		if got := t.Git().GetCommitHash("refs/heads/lower"); got != topicOID {
			t.Fail("Applying reviewed stack updates did not advance lower to the reviewed topic tip")
		}
		if got := t.Git().GetCommitHash("refs/heads/master"); got != baseOID {
			t.Fail("Applying reviewed stack updates changed the unrelated base branch")
		}
		if got := t.Git().GetCommitHash("refs/heads/publisher"); got != publisherOID {
			t.Fail("Applying reviewed stack updates changed the publisher branch")
		}
		if got := t.Git().GetCommitHash(":tracked"); got != stagedBlobOID {
			t.Fail("Applying reviewed stack updates changed the staged tracked content")
		}
		t.FileSystem().FileContent("tracked", Equals("unstaged\n"))
		t.FileSystem().FileContent("incoming-file", Equals("incoming from origin\n"))
		t.FileSystem().FileContent("untracked-binary", Equals("\x00\x01binary\xff"))
		t.Shell().RunShellCommand("cmp .git/HEAD .git/pull-stack-review-head")
		t.Shell().RunShellCommand("cmp .git/config .git/pull-stack-review-config")
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/pull-stack-review-remote-master")
		t.FileSystem().FileContent(".git/pull-stack-review-remote-master", Equals(remoteBaseOID+"\n"))
	},
})
