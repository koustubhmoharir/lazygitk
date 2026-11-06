package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFetchPrimaryRemoteOnly = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "f from Files fetches origin only, even with a second remote and git.fetchAll on",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateNCommits(2)
		shell.CloneIntoRemote("origin")
		// second is added without a fetch, so nothing of it is known yet.
		shell.Clone("second")
		shell.RunCommand([]string{"git", "remote", "add", "second", "../second"})

		// Commits pushed to both remotes after the setup are not fetched yet.
		shell.CloneNonBare("work")
		shell.RunCommand([]string{"git", "-C", "../work", "commit", "--allow-empty", "-m", "origin commit"})
		shell.RunCommand([]string{"git", "-C", "../work", "push", "../origin", "HEAD:refs/heads/origin-only"})
		shell.RunCommand([]string{"git", "-C", "../work", "commit", "--allow-empty", "-m", "second commit"})
		shell.RunCommand([]string{"git", "-C", "../work", "push", "../second", "HEAD:refs/heads/second-only"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Files.Fetch)

		t.Views().Remotes().
			Focus().
			NavigateToLine(Contains("origin")).
			PressEnter()
		t.Views().RemoteBranches().
			Content(Contains("origin-only"))

		t.Views().Remotes().
			Focus().
			NavigateToLine(Contains("second")).
			PressEnter()
		t.Views().RemoteBranches().
			Content(DoesNotContain("second-only"))
	},
})
