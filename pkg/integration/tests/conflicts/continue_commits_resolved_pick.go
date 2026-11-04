package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ContinueCommitsResolvedPick = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "c c commits a conflict resolved and staged in a cherry-pick, with its original message, then applies the remaining picks",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("file", "base\n").
			Commit("base").
			NewBranch("topic").
			UpdateFileAndAdd("file", "one\n").
			Commit("one").
			CreateFileAndAdd("two", "two\n").
			Commit("two").
			CreateFileAndAdd("three", "three\n").
			Commit("three").
			Checkout("master").
			UpdateFileAndAdd("file", "main\n").
			Commit("main change")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		oneOID := t.Git().GetCommitHash("topic^^")
		twoOID := t.Git().GetCommitHash("topic^")
		threeOID := t.Git().GetCommitHash("topic")
		masterOID := t.Git().GetCommitHash("master")

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("topic")).
			Press(config.Keybinding{"v"})

		t.Views().SubCommits().
			IsFocused().
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Universal.RangeSelectDown).
			Press(config.Keybinding{"c"})

		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"p"})

		t.Common().ConfirmCherryPickList("refs/heads/master", masterOID,
			CherryPickReviewRow(1, oneOID, "one"),
			CherryPickReviewRow(2, twoOID, "two"),
			CherryPickReviewRow(3, threeOID, "three"),
		)

		t.Common().AcknowledgeConflicts()

		// Resolve and stage the conflict, but commit nothing.
		t.Shell().UpdateFile("file", "one\n")
		t.Shell().GitAdd("file")

		t.Views().Files().
			Focus().
			Press(config.Keybinding{"c"})

		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"c"})

		t.Views().Status().Content(DoesNotContain("cherry-picking"))
		// The branch's own commits are shown too, so check only the newest rows.
		t.Views().Commits().
			TopLines(
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)
	},
})
