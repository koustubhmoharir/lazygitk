package ui

import (
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkOpenRepoByPath = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "s n opens a repository by typed path with directory suggestions and shell-style Tab; s r lists the repository left behind; invalid paths keep the current repository and the prompt text",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the s routes come from the fork catalogue.
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
		shell.EmptyCommit("first")
		shell.CloneNonBare("other")
		// A sibling directory that git cannot open. Test directories sit inside
		// lazygitk's own checkout, so a plain directory would be found within it;
		// a .git file pointing nowhere makes git fail there instead.
		shell.CreateDir("../plain")
		shell.CreateFile("../plain/.git", "gitdir: /nonexistent\n")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		parent, _ := filepath.Abs("..")
		prefilled := parent + string(filepath.Separator)

		openPathPrompt := func() {
			t.Views().Status().Focus().Press(config.Keybinding{"s"})
			t.ExpectPopup().Menu().Title(Equals("s"))
			t.Views().Menu().Press(config.Keybinding{"n"})
		}

		openPathPrompt()
		t.ExpectPopup().Prompt().
			Title(Equals("Repository path")).
			InitialText(Equals(prefilled))
		t.Views().Suggestions().Content(Contains("other/").Contains("repo/").Contains("plain/"))

		// Typing narrows the suggestions to the sibling.
		t.ExpectPopup().Prompt().Type("oth")
		t.Views().Suggestions().Content(Contains("other/").DoesNotContain("plain/"))

		// Tab completes the single match and adds the separator, staying in the prompt.
		t.Views().Prompt().PressTab()
		t.Views().Prompt().Content(Equals(prefilled + "other/"))
		t.Views().Prompt().IsFocused()

		// Enter switches to the sibling repository.
		t.ExpectPopup().Prompt().Title(Equals("Repository path")).Confirm()
		t.Views().Status().Content(Contains("other → master"))

		// s r now lists the repository we left. s r is Status-only too.
		t.Views().Status().Focus().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"r"})
		t.ExpectPopup().Menu().Title(Equals("Recent repositories")).
			Lines(
				Contains("repo").IsSelected(),
				Contains("Cancel"),
			).Cancel()

		// An invalid path reports an error, keeps the prompt's text, and leaves
		// the current repository in place.
		openPathPrompt()
		t.ExpectPopup().Prompt().Type("nope")
		t.ExpectPopup().Prompt().Title(Equals("Repository path")).Confirm()
		t.ExpectToast(Contains("Not a directory"))
		t.ExpectPopup().Prompt().
			Title(Equals("Repository path")).
			InitialText(Equals(prefilled + "nope")).
			Cancel()
		t.Views().Status().Content(Contains("other → master"))

		// A directory that is not a Git repository is refused the same way.
		openPathPrompt()
		t.ExpectPopup().Prompt().Type("plain")
		t.ExpectPopup().Prompt().Title(Equals("Repository path")).Confirm()
		t.ExpectToast(Contains("Not a Git repository"))
		t.ExpectPopup().Prompt().
			Title(Equals("Repository path")).
			InitialText(Equals(prefilled + "plain")).
			Cancel()
		t.Views().Status().Content(Contains("other → master"))
	},
})
