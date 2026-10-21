package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PlaceAtCommitOtherContexts = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Create a branch at selected commits from SubCommits and reflog without changing HEAD",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("master-01").
			EmptyCommit("master-02").
			EmptyCommit("master-03").
			RunCommand([]string{"git", "checkout", "--orphan", "otherbranch"}).
			EmptyCommit("other-01").
			EmptyCommit("other-02").
			EmptyCommit("other-03").
			Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		otherOlderOID := t.Git().GetCommitHash("refs/heads/otherbranch~2")
		masterOlderOID := t.Git().GetCommitHash("HEAD~1")

		openPlacement := func(view *ViewDriver, commitText, name string) {
			view.IsFocused().NavigateToLine(Contains(commitText)).Press(keys.Universal.OptionMenu)
			t.ExpectPopup().Menu().
				Title(Equals("Keybindings")).
				Filter("Create/move branch here").
				Confirm()
			t.ExpectPopup().Prompt().
				Title(Contains("Local branch name at")).
				Type(name).
				Confirm()
		}
		assertCreatedBranch := func(name, expectedOID string) {
			path := ".git/placement-" + name + "-tip"
			t.Shell().RunShellCommand("git rev-parse refs/heads/" + name + " > " + path)
			t.FileSystem().FileContent(path, Equals(expectedOID+"\n"))
			path = ".git/placement-" + name + "-reflog"
			t.Shell().RunShellCommand("git reflog -1 --format=%H refs/heads/" + name + " > " + path)
			t.FileSystem().FileContent(path, Equals(expectedOID+"\n"))
		}
		assertHeadUnchanged := func() {
			t.Git().CurrentBranchName("master")
			if got := t.Git().GetCommitHash("HEAD"); got != headOID {
				t.Fail("Branch placement changed HEAD")
			}
		}

		// Open the independent otherbranch history and choose a commit that is
		// not reachable from master.
		t.Views().Branches().Focus().NavigateToLine(Contains("otherbranch")).PressEnter()
		subCommits := t.Views().SubCommits()
		openPlacement(subCommits, "other-01", "from-subcommits")
		subCommits.IsFocused().SelectedLine(Contains("other-01"))
		assertCreatedBranch("from-subcommits", otherOlderOID)
		assertHeadUnchanged()

		// A reflog record for master-02 is distinct from the checked-out tip.
		reflog := t.Views().ReflogCommits()
		reflog.Focus()
		openPlacement(reflog, "master-02", "from-reflog")
		reflog.IsFocused().SelectedLine(Contains("master-02"))
		assertCreatedBranch("from-reflog", masterOlderOID)
		assertHeadUnchanged()
	},
})
