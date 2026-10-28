package ui

import (
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkShortKeysEditing = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default short keys: ] [ and Space in the staging view, , . (unbound) and > < moving commits, / filtering Branches, t s with no t item, and 0 focusing the main view",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		// Line mode makes Space stage exactly the selected change, so the two
		// one-line additions can be staged separately.
		loaded.Gui.UseHunkModeInDiffView = false
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		numbered := ""
		for i := 1; i <= 20; i++ {
			numbered += "line " + strconv.Itoa(i) + "\n"
		}
		// The branches point at base, below the commits that the move checks
		// rewrite. A branch on a rewritten commit would keep the old commit
		// listed, since the Commits view shows every branch's history.
		shell.CreateFileAndAdd("file.txt", numbered).Commit("base")
		shell.NewBranch("feature-alpha").Checkout("master")
		shell.NewBranch("bugfix").Checkout("master")
		shell.EmptyCommit("first").EmptyCommit("second").EmptyCommit("third")

		// Two far-apart one-line additions make two separate hunks in one
		// tracked file. Added lines keep each hunk to a single selectable line.
		changed := ""
		for i := 1; i <= 20; i++ {
			changed += "line " + strconv.Itoa(i) + "\n"
			if i == 2 {
				changed += "added A\n"
			}
			if i == 19 {
				changed += "added B\n"
			}
		}
		shell.UpdateFile("file.txt", changed)
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// 1. Staging: ] and [ move between the two hunks; Space stages the
		// selected change.
		t.Views().Files().
			IsFocused().
			NavigateToLine(Contains("file.txt")).
			Press(config.Keybinding{"v"})

		t.Views().Main().
			IsFocused().
			SelectedLines(Contains("+added A"))
		t.Views().Main().
			Press(config.Keybinding{"]"}).
			SelectedLines(Contains("+added B"))
		t.Views().Main().
			Press(config.Keybinding{"["}).
			SelectedLines(Contains("+added A"))
		t.Views().Main().
			Press(config.Keybinding{"<space>"})

		t.Views().Secondary().
			Content(Contains("+added A")).
			Content(DoesNotContain("+added B"))
		t.Views().Main().
			Content(Contains("+added B")).
			Content(DoesNotContain("+added A"))

		// The staging checks are done, so the leftover unstaged change is
		// removed to leave a clean tree for the commit and branch checks.
		t.Shell().RunCommand([]string{"git", "reset", "--hard"})

		// 2. Commits: , and . are unbound, so they leave the order alone;
		// > moves the selected commit down one place and < moves it back.
		commits := t.Views().Commits()
		commits.
			Focus().
			NavigateToLine(Contains("second")).
			Lines(
				Contains("third"),
				Contains("second").IsSelected(),
				Contains("first"),
				Contains("base"),
			)
		commits.Press(config.Keybinding{","}).
			Press(config.Keybinding{"."}).
			Lines(
				Contains("third"),
				Contains("second").IsSelected(),
				Contains("first"),
				Contains("base"),
			)
		commits.
			Press(config.Keybinding{">"}).
			Lines(
				Contains("third"),
				Contains("first"),
				Contains("second").IsSelected(),
				Contains("base"),
			)
		commits.
			Press(config.Keybinding{"<"}).
			Lines(
				Contains("third"),
				Contains("second").IsSelected(),
				Contains("first"),
				Contains("base"),
			)

		// 3. Branches: / filters the list; Escape clears the filter.
		branches := t.Views().Branches()
		branches.
			Focus().
			FilterOrSearch("feature").
			Lines(
				Contains("feature-alpha").IsSelected(),
			)
		branches.PressEscape()
		branches.
			IsFocused().
			Content(Contains("bugfix").Contains("feature-alpha").Contains("master"))

		// t s shows the sort menu; it has no t item, so t does nothing there.
		branches.Press(config.Keybinding{"t"})
		t.ExpectPopup().Menu().Title(Equals("t"))
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("t s"))
		t.Views().Menu().Press(config.Keybinding{"t"})
		// Escape closes the t s menu, which was opened from the t menu, so the
		// t menu is still open underneath; a second Escape closes it.
		t.ExpectPopup().Menu().Title(Equals("t s")).Cancel()
		t.ExpectPopup().Menu().Title(Equals("t")).Cancel()
		t.Views().Menu().IsInvisible()

		// 4. 0 focuses the main view from Branches; Escape returns to Branches.
		branches.IsFocused().Press(config.Keybinding{"0"})
		t.Views().Main().IsFocused()
		t.Views().Main().PressEscape()
		t.Views().Branches().IsFocused()
	},
})
