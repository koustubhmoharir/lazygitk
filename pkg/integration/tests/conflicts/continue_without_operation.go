package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ContinueWithoutOperation = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "c c does nothing when no cherry-pick, rebase, merge or revert is in progress",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "base\n").Commit("base")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().IsFocused()

		t.Views().Files().Press(config.Keybinding{"c"})
		t.Views().Menu().IsInvisible()
		t.Views().Files().IsFocused()
		t.Views().Options().Content(DoesNotContain("Continue"))
	},
})
