package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ForkDefaultConflicts = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default v inspection, t n c/t p c conflict navigation, g h p/g h b hunk picks and u u undo in the merge conflicts view",
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
		shared.CreateMergeConflictFileMultiple(shell)
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		// Opens the real default prefix menu for the first key, then picks the
		// continuation keys from the visible menus.
		chord := func(view *ViewDriver, keys ...string) {
			path := ""
			for i, key := range keys {
				if i == 0 {
					view.Press(config.Keybinding{key})
				} else {
					t.Views().Menu().Press(config.Keybinding{key})
				}
				if path == "" {
					path = key
				} else {
					path += " " + key
				}
				if i < len(keys)-1 {
					t.ExpectPopup().Menu().Title(Equals(path))
				}
			}
		}

		// 1. Files: "v" is the default inspection key; Enter commits in Files.
		t.Views().Files().
			IsFocused().
			Lines(
				Contains("UU file").IsSelected(),
			)
		t.Views().Files().Press(config.Keybinding{"v"})

		conflicts := t.Views().MergeConflicts()
		conflicts.
			IsFocused().
			Content(Contains("<<<<<<< HEAD\nFirst Change")).
			SelectedLines(
				Contains("<<<<<<< HEAD"),
				Contains("First Change"),
				Contains("======="),
			)

		// 2. t n c moves to the second conflict; t p c moves back to the first.
		chord(conflicts, "t", "n", "c")
		conflicts.
			IsFocused().
			SelectedLines(
				Contains("<<<<<<< HEAD"),
				Contains("Other First Change"),
				Contains("======="),
			)
		chord(conflicts, "t", "p", "c")
		conflicts.
			IsFocused().
			SelectedLines(
				Contains("<<<<<<< HEAD"),
				Contains("First Change"),
				Contains("======="),
			)

		// 3. g h p picks the first hunk of the first conflict; u u undoes it.
		chord(conflicts, "g", "h", "p")
		conflicts.
			IsFocused().
			Content(DoesNotContain("<<<<<<< HEAD\nFirst Change"))
		chord(conflicts, "u", "u")
		conflicts.
			IsFocused().
			Content(Contains("<<<<<<< HEAD\nFirst Change"))

		// 4. g h b keeps both hunks of the selected second conflict.
		chord(conflicts, "t", "n", "c")
		conflicts.
			IsFocused().
			SelectedLines(
				Contains("<<<<<<< HEAD"),
				Contains("Other First Change"),
				Contains("======="),
			)
		chord(conflicts, "g", "h", "b")
		conflicts.
			IsFocused().
			Content(Contains("Other First Change\nOther Second Change")).
			Content(DoesNotContain("Other Second Change\n>>>>>>>"))

		// 5. Escape returns to Files without leaving a popup behind.
		conflicts.Press(config.Keybinding{"<esc>"})
		t.Views().Files().IsFocused()
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()
	},
})
