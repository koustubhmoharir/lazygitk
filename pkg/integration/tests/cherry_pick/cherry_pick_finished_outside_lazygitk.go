package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var CherryPickFinishedOutsideLazygitk = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "When a c p that stopped on conflicts is finished outside lazygitk, the selection stays and a notice says so",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("file", "base\n").Commit("base").
			NewBranch("source").
			UpdateFileAndAdd("file", "source\n").Commit("s1").
			CreateFileAndAdd("other", "other\n").Commit("s2").
			CreateFileAndAdd("third", "third\n").Commit("s3").
			Checkout("master").
			UpdateFileAndAdd("file", "main\n").Commit("m1")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		masterOID := t.Git().GetCommitHash("master")
		s1OID := t.Git().GetCommitHash("source~2")
		s3OID := t.Git().GetCommitHash("source")

		// s1 conflicts with master's change to the same file; s3 is not next to s1.
		commits := t.Views().Commits().Focus()
		space := config.Keybinding{"<space>"}
		commits.NavigateToLine(Contains("s1")).Press(space)
		commits.NavigateToLine(Contains("s3")).Press(space)
		commits.ExplicitSelection(Contains("s3"), Contains("s1"))

		t.Common().CherryPickSelectionWithDefaultKeys(commits)
		t.Common().ConfirmCherryPickList("refs/heads/master", masterOID,
			CherryPickReviewRow(1, s1OID, "s1"),
			CherryPickReviewRow(2, s3OID, "s3"),
		)

		t.Common().AcknowledgeConflicts()

		// Resolve the conflict and finish the pick outside lazygitk.
		t.Shell().
			UpdateFileAndAdd("file", "resolved\n").
			RunCommand([]string{"git", "-c", "core.editor=true", "cherry-pick", "--continue"})

		// F5 refreshes everything in the fork's default keymap.
		t.GlobalPress(config.Keybinding{"<f5>"})

		t.ExpectToast(Equals("The cherry-pick ended outside lazygitk, so the picked commits stay selected."))

		// The selection is kept, since the operation did not end through lazygitk.
		commits.ExplicitSelection(Contains("s3"), Contains("s1"))
	},
})
