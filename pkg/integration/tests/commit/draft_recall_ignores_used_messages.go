package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftRecallIgnoresUsedMessages = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "With no draft, Up and Down leave the commit message box unchanged and never show an earlier commit's message, also in the combine box",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.EmptyCommit("commit 2")
		shell.EmptyCommit("commit 3")
		shell.CreateFileAndAdd("myfile", "myfile content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			PressEnter()

		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("")).
			SelectPreviousMessage().
			Content(Equals("")).
			Type("my commit message").
			SelectPreviousMessage().
			Content(Equals("my commit message")).
			SelectNextMessage().
			Content(Equals("my commit message")).
			Confirm()

		space := config.Keybinding{"<space>"}
		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("commit 2")).Press(space)
		commits.NavigateToLine(Contains("commit 3")).Press(space)
		commits.Press(config.Keybinding{"G"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"C"})
		t.ExpectPopup().Confirmation().Title(Equals("Review commit combination")).
			Content(Contains("commit 2").Contains("commit 3").Contains("Unselected commits remain separate")).
			Confirm()

		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("commit 2")).
			SelectPreviousMessage().
			Content(Equals("commit 2")).
			SelectNextMessage().
			Content(Equals("commit 2")).
			Cancel()
	},
})
