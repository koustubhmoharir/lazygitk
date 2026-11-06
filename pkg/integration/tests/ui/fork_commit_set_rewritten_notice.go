package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCommitSetRewrittenNotice = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A selected commit that an amend from the shell rewrites stays selected and counted, and the refresh names it in a notice",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.Gui.ShowRootItemInFileTree = true
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("base.txt", "base\n").Commit("base").
			CreateFileAndAdd("a.txt", "a\n").Commit("alpha").
			CreateFileAndAdd("b.txt", "b\n").Commit("bravo")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		commits := t.Views().Commits().Focus()
		space := config.Keybinding{"<space>"}
		oldShort := t.Git().GetCommitHash("HEAD")[:7]

		commits.NavigateToLine(Contains("bravo")).Press(space)
		commits.ExplicitSelection(Contains("bravo"))
		commits.Subtitle(Equals("1 selected"))

		// Amending from the shell replaces bravo with a new commit. The old one is no
		// longer reached by any branch, so the refresh names it, and it stays selected.
		t.Shell().RunCommand([]string{"git", "commit", "--amend", "-m", "bravo amended"})
		commits.Press(config.Keybinding{"<f5>"})
		t.ExpectToast(Contains(oldShort))
		commits.Lines(Contains("bravo amended"), Contains("alpha"), Contains("base"))
		commits.Subtitle(Equals("1 selected, 1 not shown"))
	},
})
