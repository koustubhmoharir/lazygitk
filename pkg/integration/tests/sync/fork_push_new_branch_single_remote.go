package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkPushNewBranchSingleRemote = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "With one remote and no push remote, pushing a new branch starts with that remote: the menu names it and Push this branch pushes without a prompt",
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
		// feature has neither an upstream nor a push remote.
		shell.NewBranch("feature").EmptyCommit("feature-tip")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		featureOID := t.Git().GetCommitHash("refs/heads/feature")

		t.Views().Branches().Focus().
			NavigateToLine(Contains("feature")).
			Press(keys.Universal.Push)

		// The menu names the only remote as the destination, so no prompt is needed.
		t.ExpectPopup().Menu().
			Title(Equals("Push selected branch")).
			Content(Contains("Remote: \"origin\"")).
			Select(Contains("Push this branch")).
			Confirm()

		// No remote prompt opens; the push completes from the menu alone.
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/feature > .git/assert-fork-push-feature")
		t.FileSystem().FileContent(".git/assert-fork-push-feature", Equals(featureOID+"\n"))
		t.Git().CurrentBranchName("master")
	},
})
