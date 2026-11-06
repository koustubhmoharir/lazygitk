package cherry_pick

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var SelectionHiddenByFilterIsPicked = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A selected commit that a path filter hides stays selected and counted; clearing the filter shows it selected again, and c p picks both commits",
	SetupConfig: shared.UseDefaultKeymap,
	SetupRepo: func(shell *Shell) {
		shell.
			CreateFileAndAdd("base.txt", "base\n").Commit("base").
			NewBranch("source").
			CreateFileAndAdd("s1.txt", "s1\n").Commit("s1").
			CreateFileAndAdd("s2.txt", "s2\n").Commit("s2").
			Checkout("master").
			CreateFileAndAdd("m1.txt", "m1\n").Commit("m1")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		masterOID := t.Git().GetCommitHash("master")
		s1OID := t.Git().GetCommitHash("source~1")
		s2OID := t.Git().GetCommitHash("source")

		commits := t.Views().Commits().Focus()
		space := config.Keybinding{"<space>"}
		commits.NavigateToLine(Contains("s1")).Press(space)
		commits.NavigateToLine(Contains("s2")).Press(space)
		commits.ExplicitSelection(Contains("s2"), Contains("s1"))
		commits.Subtitle(Equals("2 selected"))

		// A path filter on s1.txt hides s2. The set keeps s2, and the subtitle counts it.
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"p"})
		t.ExpectPopup().Prompt().Title(Equals("Enter path:")).Type("s1.txt").Confirm()
		commits.Lines(Contains("s1"))
		commits.ExplicitSelection(Contains("s1"))
		commits.Subtitle(Equals("2 selected, 1 not shown"))

		// Clearing the filter shows s2 again, still selected, and the count is back to two.
		commits.Press(config.Keybinding{"w"})
		t.ExpectPopup().Menu().Title(Equals("w"))
		t.Views().Menu().Press(config.Keybinding{"c"})
		commits.ExplicitSelection(Contains("s2"), Contains("s1"))
		commits.Subtitle(Equals("2 selected"))

		// c p picks both commits, including the one the filter hid.
		t.Common().CherryPickSelectionWithDefaultKeys(commits)
		t.Common().ConfirmCherryPickList("refs/heads/master", masterOID,
			CherryPickReviewRow(1, s1OID, "s1"),
			CherryPickReviewRow(2, s2OID, "s2"),
		)

		commits.ExplicitSelection()
		commits.TopLines(
			Contains("s2"),
			Contains("s1"),
			Contains("m1"),
		)
	},
})
