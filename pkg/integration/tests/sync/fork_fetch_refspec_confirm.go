package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFetchRefspecConfirm = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "f shows the fetch refspecs that write into local branches before fetching; Cancel fetches nothing, confirm fetches",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("topic")
		shell.Checkout("master")
		shell.RunCommand([]string{"git", "config", "--add", "remote.origin.fetch", "+refs/heads/topic:refs/heads/topic"})

		// origin's topic moves on, and the local topic is still at the base commit.
		shell.CloneNonBare("work")
		shell.RunCommand([]string{"git", "-C", "../work", "checkout", "topic"})
		shell.RunCommand([]string{"git", "-C", "../work", "commit", "--allow-empty", "-m", "topic moved"})
		shell.RunCommand([]string{"git", "-C", "../work", "push", "../origin", "topic"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Files.Fetch)

		t.ExpectPopup().Confirmation().
			Title(Equals("Fetch will move local branches")).
			Content(Contains("- origin: +refs/heads/topic:refs/heads/topic")).
			Cancel()

		t.Git().SameCommitAs("topic", "master")

		t.Views().Files().
			Focus().
			Press(keys.Files.Fetch)

		t.ExpectPopup().Confirmation().
			Title(Equals("Fetch will move local branches")).
			Content(Contains("- origin: +refs/heads/topic:refs/heads/topic")).
			Confirm()

		t.Git().SameCommitAs("topic", "origin/topic")
	},
})
