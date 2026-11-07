package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RebaseOntoTrackingBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Rebase the checked-out branch onto its remote tracking branch, from the checked-out branch only",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.
			EmptyCommit("initial").
			CloneIntoRemote("origin").
			NewBranch("feature").
			EmptyCommit("remote-commit").
			PushBranchAndSetUpstream("origin", "feature").
			HardReset("HEAD~1").
			EmptyCommit("local-commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().
			Focus().
			Lines(
				Contains("feature").IsSelected(),
				Contains("master"),
			).
			SelectNextItem().
			Lines(
				Contains("feature"),
				Contains("master").IsSelected(),
			).
			Press(keys.Branches.RebaseBranch)

		t.ExpectPopup().Menu().
			Title(Equals("Rebase 'feature'")).
			Select(Contains("Rebase onto a tracking branch…")).
			Tooltip(Contains("Disabled: Check out 'master' first")).
			Confirm().
			Tap(func() {
				t.ExpectToast(Equals("Disabled: Check out 'master' first"))
			}).
			Cancel()

		t.Views().Branches().
			Focus().
			SelectPreviousItem().
			Lines(
				Contains("feature").IsSelected(),
				Contains("master"),
			).
			Press(keys.Branches.RebaseBranch)

		t.ExpectPopup().Menu().
			Title(Equals("Rebase 'feature'")).
			Select(Contains("Rebase onto tracking branch 'origin/feature'")).
			Confirm()

		t.Views().Commits().Lines(
			Contains("local-commit"),
			Contains("remote-commit"),
			Contains("initial"),
		)
	},
})
