package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultFilesCRebaseEdit = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Files c in a rebase stopped at an edit commit commits the staged changes, and continues the rebase when nothing is staged",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the e and c routes come from the fork catalogue.
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
		shell.CreateNCommits(2)
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("commit-01")).
			Press(config.Keybinding{"e"})
		t.Views().Status().Content(Contains("(rebasing)"))

		// With a staged change, c commits it at the edit stop and the rebase stays stopped.
		// Auto refresh is off in this test, so F5 shows the change made outside lazygitk.
		t.Shell().CreateFileAndAdd("staged.txt", "staged\n")
		t.Views().Files().
			Focus().
			Press(config.Keybinding{"<f5>"}).
			Lines(
				Contains("staged.txt").IsSelected(),
			).
			Press(config.Keybinding{"c"})
		t.ExpectPopup().CommitMessagePanel().Type("edit stop commit").Confirm()
		t.Views().Commits().Content(Contains("edit stop commit"))
		t.Views().Status().Content(Contains("(rebasing)"))

		// With nothing staged, c continues the rebase to its end.
		t.Views().Files().IsEmpty()
		t.Views().Options().Content(Contains("Continue rebase"))
		t.Views().Files().Press(config.Keybinding{"c"})
		t.Views().Status().Content(DoesNotContain("rebasing"))
		t.Views().Commits().TopLines(
			Contains("commit-02"),
			Contains("edit stop commit"),
			Contains("commit-01"),
		)
	},
})
