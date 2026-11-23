package main_view

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var StageUnderUnsupportedDiffRenderer = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Focusing a diff renderer that says nothing about its rows keeps its output, shows no selection, and explains why Space does nothing; cycling to git's own diff makes the lines stageable",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.UseHunkModeInDiffView = false
		// `cat -n` numbers every line, which pushes the +/- column off the start of it:
		// the diff can't be read back from the text, and cat says nothing about what it
		// is rendering, so there is no way to act on what it produces.
		// Git's own diff follows it, so that cycling away from cat leaves it.
		cfg.GetUserConfig().Git.DiffRenderers = []config.DiffRendererConfig{
			{Command: "cat -n"},
			{Type: "rawGit"},
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\n")
		shell.Commit("one")

		shell.UpdateFile("file1", "one\ntwo\nTHREE\nfour\nfive\nsix\nseven\neight\nNINE\nten\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// While browsing, the view shows the renderer's output.
		t.Views().Files().
			IsFocused().
			Lines(
				Contains("file1").IsSelected(),
			)
		t.Views().Main().Content(Contains("1  diff --git a/file1 b/file1"))

		// Focusing it keeps the renderer's output. Its rows can't be placed in the file,
		// so there is no selection, and Space says why nothing is staged.
		t.Views().Files().Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			Content(Contains("1  diff --git")).
			SelectionIsHidden().
			PressPrimaryAction()

		t.ExpectToast(Equals("Disabled: This diff renderer doesn't say which line each row is; switch renderer with t d n to act on lines"))

		t.Views().Files().Lines(
			Contains("M file1"),
		)

		// Cycling to git's own diff, which describes its rows, keeps the main view
		// focused and makes the lines stageable. The selection keeps its line index,
		// which is the file header here, so the next hunk is taken first.
		t.Views().Main().
			Press(keys.Universal.CycleDiffRenderers).
			IsFocused().
			Content(DoesNotContain("1  diff --git")).
			Tap(func() {
				t.ExpectToast(Equals("Diff renderer: (default) (2 of 2)"))
			}).
			Press(keys.Main.NextHunk).
			SelectedLines(
				Contains("-three"),
			).
			Press(keys.Main.ToggleSelectHunk).
			SelectedLines(
				Contains("-three"),
				Contains("+THREE"),
			).
			PressPrimaryAction()

		t.Views().Secondary().ContainsLines(
			Contains("-three"),
			Contains("+THREE"),
		)
		// The re-render after staging stays with git's diff, so the next hunk can be
		// staged as well.
		t.Views().Main().
			IsFocused().
			SelectedLines(
				Contains("-nine"),
				Contains("+NINE"),
			).
			PressPrimaryAction()

		t.Views().Files().Lines(
			Contains("M  file1"),
		)
	},
})
