package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultKeymapSearch = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default keys in Commits: Enter, filter and search with f3/Shift-f3/N, v and V into commit files, and the ? keybindings menu",
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
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		// Reproduces the commit made by the Files steps of
		// fork_default_keymap_files_tree, so the Commits steps have the same
		// history to search and navigate.
		shell.CreateFileAndAdd("dir/a.txt", "base a\n").CreateFileAndAdd("dir/b.txt", "base b\n").CreateFileAndAdd("root.txt", "base root\n").Commit("match first").EmptyCommit("match second").
			UpdateFileAndAdd("dir/a.txt", "edited a\n").UpdateFileAndAdd("dir/b.txt", "edited b\n").UpdateFileAndAdd("root.txt", "edited root\n").Commit("keyboard new commit done")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		commits := t.Views().Commits()
		commits.Focus().PressEnter().IsFocused()
		t.Views().CommitMessage().IsInvisible()
		t.Views().Options().Content(Contains("d w"))
		commits.FilterOrSearch("match").SelectedLine(Contains("match second")).
			Press(config.Keybinding{"<f3>"}).SelectedLine(Contains("match first")).
			Press(config.Keybinding{"<shift+f3>"}).SelectedLine(Contains("match second")).
			Press(config.Keybinding{"N"})
		t.ExpectPopup().Menu().Title(Equals("n")).Cancel()
		commits.IsFocused().PressEscape().NavigateToLine(Contains("keyboard new commit done")).Press(config.Keybinding{"v"})
		t.Views().CommitFiles().IsFocused().PressEnter().IsFocused()
		t.Views().CommitMessage().IsInvisible()
		t.Views().CommitFiles().PressEscape()
		commits.IsFocused().Press(config.Keybinding{"?"})
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Pull checked-out branch")
		t.Views().Menu().SelectedLine(Contains("g l").Contains("Pull checked-out branch"))
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		t.Views().MenuFilter().IsInvisible()
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Cancel()
		commits.IsFocused().Press(config.Keybinding{"V"})
		t.Views().CommitFiles().IsFocused()
	},
})
