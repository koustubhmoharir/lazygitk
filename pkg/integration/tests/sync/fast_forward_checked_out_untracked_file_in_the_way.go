package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardCheckedOutUntrackedFileInTheWay = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to fast-forward the checked out branch when an incoming commit would overwrite an untracked file",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.CreateFileAndAdd("incoming", "committed")
		shell.Commit("two")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")

		// remove the commit that adds the file so that it is incoming
		shell.HardReset("HEAD~1")

		// an untracked file at the path the incoming commit adds
		shell.CreateFile("incoming", "untracked user bytes")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("cp .git/index .git/ff-untracked-index")

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master ↓1").IsSelected(),
			).
			Press(keys.Branches.FastForward)

		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).
			Content(Contains("master: ").Contains("would be overwritten")).
			Confirm()

		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("refused fast-forward changed HEAD: " + got)
		}
		if got := t.Git().GetCommitHash("refs/heads/master"); got != headOID {
			t.Fail("refused fast-forward moved master: " + got)
		}
		t.Shell().RunShellCommand("cmp .git/index .git/ff-untracked-index")
		t.FileSystem().FileContent("incoming", Equals("untracked user bytes"))
		t.Views().Branches().
			Lines(
				Contains("master ↓1").IsSelected(),
			)
	},
})
