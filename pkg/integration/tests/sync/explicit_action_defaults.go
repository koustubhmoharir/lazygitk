package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ExplicitActionDefaults = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Automatic fetch and file refresh are disabled by default while external ref polling remains active",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(appConfig *config.AppConfig) {
		userConfig := appConfig.GetUserConfig()
		defaults := config.GetDefaultConfig()
		// The shared harness disables all three routines explicitly. Restore
		// their actual defaults here rather than testing those harness overrides.
		userConfig.Git.AutoFetch = defaults.Git.AutoFetch
		userConfig.Git.AutoRefresh = defaults.Git.AutoRefresh
		userConfig.Git.AutoDetectExternalChanges = defaults.Git.AutoDetectExternalChanges
		userConfig.Refresher.FetchInterval = 1
		userConfig.Refresher.RefreshInterval = 1
		userConfig.Refresher.ExternalChangeCheckInterval = 1
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
		t.Views().Branches().EventuallyContent(Contains("master").DoesNotContain("↓"))
		t.Wait(2200)
		t.Shell().RunShellCommand("test \"$(git rev-parse origin/master)\" = \"$(git rev-parse HEAD)\" && test \"$(git rev-parse --verify FETCH_HEAD)\" = \"$(git rev-parse refs/test/remote-ahead)\"")

		t.Shell().CreateFile("untracked-default-check", "content")
		t.Views().Files().Content(DoesNotContain("untracked-default-check"))
		t.Wait(2200)
		t.Views().Files().Content(DoesNotContain("untracked-default-check"))

		t.Shell().EmptyCommit("external ref update")
		t.Views().Commits().EventuallyContent(Contains("external ref update"))
	},
})
