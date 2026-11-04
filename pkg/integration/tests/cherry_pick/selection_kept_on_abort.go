package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var SelectionKeptOnAbort = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Aborting a pick that stopped on conflicts keeps the selection, so the pick can be retried",
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

		// Abort the pick from the options menu. The selection stays.
		// g o opens the merge or rebase options, from which the operation is aborted.
		t.Views().Commits().
			Focus().
			Press(config.Keybinding{"g"})
		t.ExpectPopup().Menu().Title(Equals("g"))
		t.Views().Menu().Press(config.Keybinding{"o"})
		t.ExpectPopup().Menu().
			Title(Equals("Cherry-pick options")).
			Select(Contains("abort")).
			Confirm()

		t.Views().Status().
			Content(DoesNotContain("cherry-picking"))
		commits.ExplicitSelection(Contains("s3"), Contains("s1"))
	},
})
