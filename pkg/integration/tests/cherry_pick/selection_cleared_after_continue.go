package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var SelectionClearedAfterContinue = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A pick that stopped on conflicts keeps its selection, and c c that continues the operation to its end clears it",
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

		// s1 conflicts with master's change to the same file; s3 applies cleanly after it.
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
		commits.ExplicitSelection(Contains("s3"), Contains("s1"))

		// Resolve the conflict by taking the picked side, then stage the file.
		// With the default keys, v opens a conflicted file (Enter commits in Files).
		t.Views().Files().
			IsFocused().
			SelectedLine(Contains("file")).
			Press(config.Keybinding{"v"})

		t.Views().MergeConflicts().
			IsFocused().
			SelectNextItem().
			PressPrimaryAction()

		t.Views().Files().
			Focus().
			SelectedLine(Contains("file")).
			PressPrimaryAction()

		// c c continues the pick of s1, then picks s3, and the operation ends.
		t.Views().Files().
			IsFocused().
			Press(config.Keybinding{"c"})
		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"c"})

		t.Views().Status().
			Content(DoesNotContain("cherry-picking"))

		// Finishing the operation by continue cleared the selection.
		commits.Focus().ExplicitSelection()
	},
})
