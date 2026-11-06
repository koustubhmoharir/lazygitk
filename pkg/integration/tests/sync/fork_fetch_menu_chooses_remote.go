package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFetchMenuChoosesRemote = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With two remotes and no origin, f asks which one to fetch, and the chosen one is the only one fetched",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateNCommits(1)
		shell.Clone("alpha")
		shell.RunCommand([]string{"git", "remote", "add", "alpha", "../alpha"})
		shell.Clone("beta")
		shell.RunCommand([]string{"git", "remote", "add", "beta", "../beta"})

		shell.CloneNonBare("work")
		shell.RunCommand([]string{"git", "-C", "../work", "commit", "--allow-empty", "-m", "alpha commit"})
		shell.RunCommand([]string{"git", "-C", "../work", "push", "../alpha", "HEAD:refs/heads/alpha-only"})
		shell.RunCommand([]string{"git", "-C", "../work", "commit", "--allow-empty", "-m", "beta commit"})
		shell.RunCommand([]string{"git", "-C", "../work", "push", "../beta", "HEAD:refs/heads/beta-only"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Files.Fetch)

		t.ExpectPopup().Menu().
			Title(Equals("Fetch")).
			Content(Contains("alpha")).
			Content(Contains("Fetch all remotes")).
			Select(Contains("beta")).
			Confirm()

		t.Views().Remotes().
			Focus().
			NavigateToLine(Contains("beta")).
			PressEnter()
		t.Views().RemoteBranches().
			Content(Contains("beta-only"))

		t.Views().Remotes().
			Focus().
			NavigateToLine(Contains("alpha")).
			PressEnter()
		t.Views().RemoteBranches().
			Content(DoesNotContain("alpha-only"))
	},
})
