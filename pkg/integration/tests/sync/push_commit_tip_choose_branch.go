package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushCommitTipChooseBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Choose which of several local branches at the selected commit to push",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.EmptyCommit("master-tip")
		shell.NewBranch("other")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		remoteMasterOID := t.Git().GetCommitHash("refs/remotes/origin/master")

		// Both master and other point at the selected commit. Master is checked
		// out, so it is listed first and is the default choice.
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("master-tip")).
			Press(keys.Universal.Push)

		chooser := t.ExpectPopup().Menu().
			Title(Equals("Push which branch?")).
			ContainsLines(Equals("master"), Equals("other"))
		chooser.Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("Source \"master\"")).
			Cancel()

		t.Views().Commits().
			NavigateToLine(Contains("master-tip")).
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().
			Title(Equals("Push which branch?")).
			Select(Equals("other")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("Source \"other\"")).
			Select(Contains("Push this branch")).
			Confirm()

		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/other > .git/assert-push-choose-other")
		t.FileSystem().FileContent(".git/assert-push-choose-other", Equals(headOID+"\n"))
		// Master was never chosen, so origin's master stays at the base commit.
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-choose-master")
		t.FileSystem().FileContent(".git/assert-push-choose-master", Equals(remoteMasterOID+"\n"))
		t.Git().CurrentBranchName("master")
	},
})
