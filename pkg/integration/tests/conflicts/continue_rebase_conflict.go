package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ContinueRebaseConflict = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "c c continues a conflicted rebase that was started outside lazygitk, with the status line naming rebase",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("file", "base\n").
			Commit("base").
			NewBranch("topic").
			UpdateFileAndAdd("file", "topic\n").
			Commit("topic change").
			Checkout("master").
			UpdateFileAndAdd("file", "main\n").
			Commit("main change").
			Checkout("topic").
			RunCommandExpectError([]string{"git", "rebase", "master"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Lines(
				Contains("UU file").IsSelected(),
			)
		t.Views().Status().Content(Contains("rebasing"))

		t.Shell().UpdateFile("file", "resolved\n")
		t.Shell().GitAdd("file")

		t.Views().Options().Content(Contains("Continue rebase"))

		t.Views().Files().
			Press(config.Keybinding{"c"})

		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"c"})

		t.Views().Status().Content(DoesNotContain("rebasing"))
		t.Views().Commits().
			Lines(
				Contains("topic change"),
				Contains("main change"),
				Contains("base"),
			)
	},
})
