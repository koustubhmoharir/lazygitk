package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesCommandRenderer = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In the full listing of a commit's files, a stdinFilter file renderer shows the file's content through its command, with the file name quoted; a failing command shows its error",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Git.FileRenderers = []config.FileRendererConfig{
			{Type: "stdinFilter", Name: "prefix", Command: "awk -v f={{filename}} '{print f \": \" $0}'"},
			{Type: "plain"},
			{Type: "stdinFilter", Name: "fail", Command: "sh -c 'echo boom >&2; exit 3'"},
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("my notes.txt", "first line\nsecond line\n")
		shell.Commit("first")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"f"})

		// The command gets the file's content on stdin, and its output is shown. The file
		// name has a space, which the command sees as one argument.
		t.Views().CommitFiles().
			IsFocused().
			NavigateToLine(Contains("A my notes.txt"))
		t.Views().Main().
			Title(Equals("File content")).
			Content(Contains("my notes.txt: first line")).
			Content(Contains("my notes.txt: second line"))

		// Cycling to plain text shows the content as it is.
		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderers)
		t.ExpectToast(Equals("File renderer: Plain text (2 of 3)"))
		t.Views().Main().
			Content(Contains("first line")).
			Content(DoesNotContain("my notes.txt: "))

		// A command that fails shows its error in the main view.
		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderers)
		t.ExpectToast(Equals("File renderer: fail (3 of 3)"))
		t.Views().Main().
			Content(Contains("boom")).
			Content(DoesNotContain("first line"))

		// Cycling on from the failing command returns to the first renderer.
		t.Views().CommitFiles().
			Press(keys.Universal.CycleDiffRenderers)
		t.ExpectToast(Equals("File renderer: prefix (1 of 3)"))
		t.Views().Main().
			Content(Contains("my notes.txt: first line"))
	},
})
