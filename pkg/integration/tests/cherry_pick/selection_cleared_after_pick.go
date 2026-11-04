package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var SelectionClearedAfterPick = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "c p in Commits picks two non-adjacent selected commits of another branch, and the successful pick clears the selection",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("base.txt", "base\n").Commit("base").
			NewBranch("source").
			CreateFileAndAdd("s1.txt", "s1\n").Commit("s1").
			CreateFileAndAdd("s2.txt", "s2\n").Commit("s2").
			CreateFileAndAdd("s3.txt", "s3\n").Commit("s3").
			Checkout("master").
			CreateFileAndAdd("m1.txt", "m1\n").Commit("m1")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		masterOID := t.Git().GetCommitHash("master")
		s1OID := t.Git().GetCommitHash("source~2")
		s3OID := t.Git().GetCommitHash("source")

		// Commits shows the whole graph, so source's commits are listed too. s1 and s3
		// are not next to each other, so each is added to the selection with Space.
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

		// The pick succeeded, so no selection remains. Escape has nothing to clear.
		commits.ExplicitSelection()
		commits.Press(config.Keybinding{"<esc>"})
		commits.ExplicitSelection()
		commits.NoMarkedLines()

		commits.TopLines(
			Contains("s3"),
			Contains("s1"),
			Contains("m1"),
		)
	},
})
