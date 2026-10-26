package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPushNewBranchTwoRemotes = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "With two remotes and no push remote, Push this branch still opens the remote prompt, and the typed remote receives the branch",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the p and push menu keys match the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked", "base\n").Commit("base")
		shell.CloneIntoRemote("origin")
		shell.CloneIntoRemote("upstream")
		shell.NewBranch("feature").EmptyCommit("feature-tip")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		featureOID := t.Git().GetCommitHash("refs/heads/feature")

		t.Views().Branches().Focus().
			NavigateToLine(Contains("feature")).
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Select(Contains("Push this branch")).
			Confirm()

		// Ambiguous: with two remotes, the prompt opens and nothing is preselected.
		t.ExpectPopup().Prompt().
			Title(Equals("Remote name")).
			InitialText(Equals("")).
			Type("upstream").
			Confirm()

		t.Shell().RunShellCommand("git --git-dir=../upstream rev-parse --verify refs/heads/feature > .git/assert-fork-push-two-remotes")
		t.FileSystem().FileContent(".git/assert-fork-push-two-remotes", Equals(featureOID+"\n"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify --quiet refs/heads/feature > .git/assert-fork-push-two-remotes-origin || true")
		t.FileSystem().FileContent(".git/assert-fork-push-two-remotes-origin", Equals(""))
		t.Git().CurrentBranchName("master")
	},
})
