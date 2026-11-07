package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var CherryPickAddAddConflict = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A pick that stops on an add/add conflict offers the conflict flow, not a plain error",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		// Both branches add added.txt with different contents. The file is not
		// called "file", so only Git's "CONFLICT (add/add):" line identifies the
		// conflict.
		shell.
			CreateFileAndAdd("base.txt", "base\n").Commit("base").
			NewBranch("source").
			CreateFileAndAdd("added.txt", "source\n").Commit("source adds").
			Checkout("master").
			CreateFileAndAdd("added.txt", "master\n").Commit("master adds")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		masterOID := t.Git().GetCommitHash("master")
		sourceOID := t.Git().GetCommitHash("source")

		commits := t.Views().Commits().Focus()
		commits.NavigateToLine(Contains("source adds")).Press(config.Keybinding{"<space>"})
		commits.ExplicitSelection(Contains("source adds"))

		t.Common().CherryPickSelectionWithDefaultKeys(commits)
		t.Common().ConfirmCherryPickList("refs/heads/master", masterOID,
			CherryPickReviewRow(1, sourceOID, "source adds"),
		)

		t.Common().AcknowledgeConflicts()

		t.Views().Files().
			IsFocused().
			SelectedLine(Contains("added.txt"))
	},
})
