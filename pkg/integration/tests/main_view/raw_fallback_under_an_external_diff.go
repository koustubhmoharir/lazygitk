package main_view

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RawFallbackUnderAnExternalDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Focusing the main view under an external diff that says nothing about its rows keeps the renderer's output and scroll position, shows no selection, and the primary action explains why",
	ExtraCmdArgs: []string{},
	Skip:         false,
	Width:        120,
	Height:       30,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.UseHunkModeInDiffView = false
		// An external diff whose output has nothing to do with the diff it was given,
		// let alone anything to say about which line of which file each row shows. It
		// is long enough to be scrolled about in.
		cfg.GetUserConfig().Git.DiffRenderers = []config.DiffRendererConfig{
			{Name: "opaque", Type: "extDiff", Command: `sh -c 'seq -f "EXT-%g" 40'`},
		}
	},
	SetupRepo: func(shell *Shell) {
		lines := make([]string, 40)
		for i := range lines {
			lines[i] = fmt.Sprintf("line%02d", i+1)
		}
		shell.CreateFileAndAdd("file1", strings.Join(lines, "\n")+"\n")
		shell.Commit("one")

		for _, i := range []int{5, 15, 25, 35} {
			lines[i-1] = strings.ToUpper(lines[i-1])
		}
		shell.UpdateFile("file1", strings.Join(lines, "\n")+"\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Browsing shows what the renderer produced, whatever that is.
		t.Views().Main().
			Content(Contains("EXT-1")).
			Tap(func() {
				t.Views().Files().
					IsFocused().
					Press(keys.Universal.ScrollDownMain).
					Press(keys.Universal.ScrollDownMain).
					Press(keys.Universal.ScrollDownMain)
			}).
			OriginY(6).
			Tap(func() {
				t.Views().Files().Press(keys.Universal.FocusMainView)
			}).
			IsFocused().
			// Focusing the view to act on it keeps the renderer's output, which has no
			// rows that can be placed in the file, so there is no selection. The view
			// stays at the offset it was at, rather than going to the top.
			Content(Contains("EXT-1")).
			Content(DoesNotContain("+LINE05")).
			OriginY(6).
			SelectionIsHidden().
			PressPrimaryAction()

		t.ExpectToast(Equals("Disabled: This diff renderer doesn't say which line each row is; switch renderer with t d n to act on lines"))

		t.Views().Files().Lines(
			Contains("M file1"),
		)
	},
})
