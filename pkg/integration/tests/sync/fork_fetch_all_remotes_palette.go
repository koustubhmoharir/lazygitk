package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFetchAllRemotesPalette = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The palette's Fetch all remotes fetches every remote, including ones that are not origin",
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
			Press(keys.Universal.OptionMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Filter("Fetch all remotes").
			Content(Contains("Fetch all remotes")).
			Confirm()

		t.Views().Remotes().
			Focus().
			NavigateToLine(Contains("alpha")).
			PressEnter()
		t.Views().RemoteBranches().
			Content(Contains("alpha-only"))

		t.Views().Remotes().
			Focus().
			NavigateToLine(Contains("beta")).
			PressEnter()
		t.Views().RemoteBranches().
			Content(Contains("beta-only"))
	},
})
