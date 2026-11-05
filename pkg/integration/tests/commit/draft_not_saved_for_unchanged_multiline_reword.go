package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var DraftNotSavedForUnchangedMultilineReword = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Escape in the reword box of a commit with a description, without changes, saves no draft",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.RunCommand([]string{"git", "commit", "--allow-empty", "-m", "subject", "-m", "body"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().Focus().NavigateToLine(Contains("subject")).Press(config.Keybinding{"<f2>"})
		t.ExpectPopup().CommitMessagePanel().
			InitialText(Equals("subject")).
			Cancel()

		t.FileSystem().PathNotPresent(".git/LAZYGITK_COMMIT_DRAFTS")
	},
})
