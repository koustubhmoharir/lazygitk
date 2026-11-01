package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultKeymapFilesTree = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default keys: Files tree arrows, scoped Space/Enter with Ctrl-word editing in the commit message, and pane/tab navigation",
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
		shell.CreateFileAndAdd("dir/a.txt", "base a\n").CreateFileAndAdd("dir/b.txt", "base b\n").CreateFileAndAdd("root.txt", "base root\n").Commit("match first").EmptyCommit("match second").
			UpdateFile("dir/a.txt", "edited a\n").UpdateFile("dir/b.txt", "edited b\n").UpdateFile("root.txt", "edited root\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		files := t.Views().Files()
		files.IsFocused().NavigateToLine(Contains("dir")).Press(config.Keybinding{"<left>"}).IsFocused().
			Content(DoesNotContain("a.txt").DoesNotContain("b.txt")).
			Press(config.Keybinding{"<left>"}).SelectedLine(Contains("/")).
			Press(config.Keybinding{"<right>"}).SelectedLine(Contains("dir")).
			Press(config.Keybinding{"<right>"}).Content(Contains("a.txt").Contains("b.txt")).
			Press(config.Keybinding{"<right>"}).SelectedLine(Contains("a.txt")).
			Press(config.Keybinding{"<left>"}).SelectedLine(Contains("dir")).
			Press(config.Keybinding{"<left>"}).Press(config.Keybinding{"<left>"})
		// An inapplicable key consumes both cases without opening a dialog or
		// falling through to the former Files action. (d has Files routes since
		// item 17, so it is no longer inapplicable here.)
		files.Press(config.Keybinding{"r"}).Press(config.Keybinding{"R"}).IsFocused()
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		files.Press(config.Keybinding{"<space>"}).PressEnter()
		panel := t.ExpectPopup().CommitMessagePanel().Type("keyboard commit")
		t.Views().CommitMessage().Press(config.Keybinding{"<ctrl+left>"})
		panel.Type("new ")
		t.Views().CommitMessage().Press(config.Keybinding{"<ctrl+right>"})
		panel.Type(" done").Content(Equals("keyboard new commit done")).Confirm()
		files.IsFocused().IsEmpty()
		files.Press(config.Keybinding{"L"})
		t.Views().Branches().IsFocused().Press(config.Keybinding{"<tab>"})
		t.Views().Remotes().IsFocused().Press(config.Keybinding{"<backtab>"})
		t.Views().Branches().IsFocused().Press(config.Keybinding{"H"})
		files.IsFocused().Press(config.Keybinding{"<f5>"}).IsFocused()
		t.Views().Confirmation().IsInvisible()
	},
})
