package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var AutomationOptIn = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Explicitly enabled automatic fetch and file refresh update the repository",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.AutoFetch = true
		config.GetUserConfig().Git.AutoRefresh = true
		config.GetUserConfig().Refresher.FetchInterval = 1
		config.GetUserConfig().Refresher.RefreshInterval = 1
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("local base")
		shell.EmptyCommit("remote ahead")
		shell.RunCommand([]string{"git", "update-ref", "refs/test/remote-ahead", "HEAD"})
		shell.CloneIntoRemote("origin")
		shell.HardReset("HEAD^")
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/master", "HEAD"})
		shell.SetBranchUpstream("master", "origin/master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().EventuallyContent(Contains("master ↓1"))
		t.Shell().RunShellCommand("test \"$(git rev-parse origin/master)\" = \"$(git rev-parse refs/test/remote-ahead)\"")

		t.Shell().CreateFile("untracked-opt-in-check", "content")
		t.Views().Files().EventuallyContent(Contains("untracked-opt-in-check"))
	},
})
