package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFlatCheckoutMenus = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default s menu contents: Status lists only the repository entries, other panes have no repository entries, s b does nothing, and s p with blocking changes shows Git's refusal instead of a stash offer",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "base\n").Commit("base")
		// feature changes the same file, so checking it out over a local change to
		// file is refused by Git.
		shell.NewBranch("feature").
			UpdateFileAndAdd("file", "feature\n").
			Commit("feature change").
			Checkout("master")
		// The unstaged change to file is the local change that blocks s p.
		shell.UpdateFile("file", "local change\n")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// Opens the real default s prefix menu, which is then checked by the caller.
		openS := func(view *ViewDriver) {
			view.Press(config.Keybinding{"s"})
			t.ExpectPopup().Menu().Title(Equals("s"))
		}

		// 1. In Status the s menu lists only the repository entries.
		status := t.Views().Status()
		status.Focus()
		openS(status)
		t.Views().Menu().
			Content(DoesNotContain("Switch to previous branch")).
			Content(DoesNotContain("Checkout")).
			Lines(
				Contains("Switch to a recent repo"),
				Contains("Open repository by path"),
				Contains("Cancel"),
			)
		t.ExpectPopup().Menu().Title(Equals("s")).Cancel()

		// 2. Files has no repository entries in its s menu.
		files := t.Views().Files()
		files.Focus()
		openS(files)
		t.Views().Menu().
			Content(DoesNotContain("Switch to a recent repo")).
			Content(DoesNotContain("Open repository by path"))
		t.ExpectPopup().Menu().Title(Equals("s")).Cancel()

		// 3. Branches has no repository entries either. Pressing b in its s menu
		// does nothing, so the menu stays open until it is cancelled.
		branches := t.Views().Branches()
		branches.Focus()
		openS(branches)
		t.Views().Menu().
			Content(DoesNotContain("Switch to a recent repo")).
			Content(DoesNotContain("Open repository by path"))
		t.Views().Menu().Press(config.Keybinding{"b"})
		t.ExpectPopup().Menu().Title(Equals("s")).Cancel()
		t.Views().Menu().IsInvisible()
		branches.IsFocused()
		t.Git().CurrentBranchName("master")

		// 4. s p with a local change that blocks the switch shows Git's refusal. No
		// stash offer appears, so the first popup after the chord is the error.
		files.Focus()
		files.Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"p"})
		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("would be overwritten")).
			Confirm()
		t.Views().Confirmation().IsInvisible()
		t.Git().CurrentBranchName("master")
	},
})
