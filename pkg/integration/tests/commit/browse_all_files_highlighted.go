package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesHighlighted = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In the full listing of a commit's files, file content is highlighted and numbered by default, and t d n cycles it to plain text and back",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("main.go", "package main\n\nfunc main() {}\n")
		shell.CreateFileAndAdd("notes.unknownext", "alpha\nbeta\n")
		shell.CreateFileAndAdd("notes.txt", "plain\n")
		shell.Commit("first")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"f"})

		// A Go file is highlighted, with its lines numbered.
		t.Views().CommitFiles().
			IsFocused().
			NavigateToLine(Equals("  A main.go"))
		t.Views().Main().
			Title(Equals("File content")).
			Content(Contains("1 package main")).
			Content(Contains("3 func main() {}"))

		// Cycling from a file shows the file renderers: the next one is plain text.
		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderers)
		t.ExpectToast(Equals("File renderer: Plain text (2 of 2)"))
		t.Views().Main().
			Content(Contains("func main() {}")).
			Content(DoesNotContain("3 func main() {}"))

		// Cycling again returns to the built-in highlighter, and backwards from it is plain text.
		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderers)
		t.ExpectToast(Equals("File renderer: Built-in highlighting (1 of 2)"))
		t.Views().Main().
			Content(Contains("3 func main() {}"))

		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderersReverse)
		t.ExpectToast(Equals("File renderer: Plain text (2 of 2)"))
		t.Views().Main().
			Content(DoesNotContain("3 func main() {}"))
		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderers)
		t.ExpectToast(Equals("File renderer: Built-in highlighting (1 of 2)"))

		// A file of an unknown type is numbered plain text.
		t.Views().CommitFiles().
			NavigateToLine(Equals("  A notes.unknownext"))
		t.Views().Main().
			Content(Equals("1 alpha\n2 beta"))
	},
})
