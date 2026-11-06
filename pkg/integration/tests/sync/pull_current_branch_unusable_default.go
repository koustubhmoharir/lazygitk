package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullCurrentBranchUnusableDefault = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "An unsupported pull.rebase value crosses out the configured default after review; explicit methods still integrate",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.RunCommand([]string{"git", "config", "branch.master.remote", "origin"})
		shell.RunCommand([]string{"git", "config", "branch.master.merge", "refs/heads/master"})
		shell.NewBranch("publisher")
		shell.CreateFileAndAdd("incoming-file", "incoming from origin\n").Commit("incoming master")
		shell.RunCommand([]string{"git", "push", "origin", "HEAD:refs/heads/master"})
		shell.Checkout("master")
		shell.SetConfig("pull.rebase", "bogus")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		incomingOID := t.Git().GetCommitHash("refs/heads/publisher")

		t.Views().Files().Focus().Press(keys.Universal.Pull)
		t.ExpectPopup().Menu().
			Title(Equals("Pull checked-out branch")).
			Select(Contains("Fetch upstream for review")).
			Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Review upstream fetch")).
			Content(Contains("refs/remotes/origin/master")).
			Confirm()

		integrationMenu := t.ExpectPopup().Menu().Title(Equals("Integrate reviewed incoming commit"))
		integrationMenu.
			Select(Contains("Use Git's configured pull method")).
			Tooltip(Contains(`unsupported pull.rebase value "bogus"; choose a method explicitly`))
		integrationMenu.Select(Contains("Merge (fast-forward if possible)")).Confirm()

		t.ExpectPopup().Confirmation().
			Title(Equals("Review current-branch integration")).
			Content(Contains(incomingOID[:7])).
			Confirm()
		t.Views().Commits().EventuallyContent(Contains("incoming master"))

		if got := t.Git().GetCommitHash("HEAD"); got != incomingOID {
			t.Fail("Explicit merge did not fast-forward to the reviewed commit")
		}
	},
})
