package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FullGraphDefault = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Commit log shows commits from local and remote refs by default",
	SetupConfig: func(appConfig *config.AppConfig) {
		// The shared harness pins the old branch-only view. Test the real default.
		appConfig.GetUserConfig().Git.Log.ShowWholeGraph = config.GetDefaultConfig().Git.Log.ShowWholeGraph
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("graph base")
		shell.NewBranch("graph-side")
		shell.EmptyCommit("graph side only")
		shell.Checkout("master")
		shell.NewBranch("graph-temporary")
		shell.EmptyCommit("graph remote only")
		shell.RunCommand([]string{"git", "update-ref", "refs/remotes/origin/graph-remote", "HEAD"})
		shell.Checkout("master")
		shell.RunCommand([]string{"git", "branch", "-D", "graph-temporary"})
		shell.EmptyCommit("graph current only")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().EventuallyContent(Contains("graph current only").Contains("graph side only").Contains("graph remote only").Contains("graph base"))
		t.Git().CurrentBranchName("master")
		t.Shell().RunShellCommand("test \"$(git symbolic-ref --short HEAD)\" = master && git show-ref --verify --quiet refs/heads/graph-side && git show-ref --verify --quiet refs/remotes/origin/graph-remote && ! git show-ref --verify --quiet refs/heads/graph-temporary")
	},
})
