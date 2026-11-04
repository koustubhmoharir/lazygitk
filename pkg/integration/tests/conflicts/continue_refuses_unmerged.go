package conflicts

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ContinueRefusesUnmerged = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "c c refuses to continue a cherry-pick while a file is unmerged, including after its markers are removed but before it is staged",
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
			Lines(
				Contains("three").IsSelected(),
				Contains("two"),
				Contains("one"),
				Contains("base"),
			).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Universal.RangeSelectDown).
			Lines(
				Contains("three").IsSelected(),
				Contains("two").IsSelected(),
				Contains("one").IsSelected(),
				Contains("base"),
			).
			Press(config.Keybinding{"c"})

		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"p"})

		t.Common().ConfirmCherryPickList("refs/heads/master", masterOID,
			CherryPickReviewRow(1, oneOID, "one"),
			CherryPickReviewRow(2, twoOID, "two"),
			CherryPickReviewRow(3, threeOID, "three"),
		)

		t.Common().AcknowledgeConflicts()

		t.Views().Files().
			IsFocused().
			Press(config.Keybinding{"c"})

		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"c"})

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Cannot continue yet. These files are still unmerged; resolve them and stage them first:\n\n- file")).
			Confirm()

		// Removing the markers is not enough: the file must be staged too.
		t.Shell().UpdateFile("file", "resolved\n")

		t.Views().Files().
			IsFocused().
			Press(config.Keybinding{"c"})

		t.ExpectPopup().Menu().Title(Equals("c"))
		t.Views().Menu().Press(config.Keybinding{"c"})

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Cannot continue yet. These files are still unmerged; resolve them and stage them first:\n\n- file")).
			Confirm()

		t.Views().Status().Content(Contains("(cherry-picking)"))
		t.Views().Options().Content(Contains("Continue cherry-pick"))
	},
})
