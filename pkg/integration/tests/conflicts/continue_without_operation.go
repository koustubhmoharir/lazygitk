package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ContinueWithoutOperation = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "c c does nothing from Commits when no cherry-pick, rebase, merge or revert is in progress; c in Files commits instead",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "base\n").Commit("base")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// c in Files commits when nothing is in progress, so c c is pressed from Commits.
		t.Views().Commits().Focus().Press(config.Keybinding{"c"})
		t.ExpectPopup().Menu().Title(Equals("c"))
		// Continue is not offered without an operation, so its key does nothing.
		t.Views().Menu().Press(config.Keybinding{"c"})
		t.ExpectPopup().Menu().Title(Equals("c")).Cancel()
		t.Views().Commits().IsFocused()
		t.Views().Options().Content(DoesNotContain("Continue"))
	},
})
