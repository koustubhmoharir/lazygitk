package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RebaseOntoTrackingBranchChoice = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Rebase the checked-out branch onto a same-named branch on another remote, chosen from the tracking branch menu",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.
			EmptyCommit("initial").
			CloneIntoRemote("origin").
			NewBranch("feature").
			EmptyCommit("fork-commit").
			CloneIntoRemote("fork").
			HardReset("HEAD~1").
			EmptyCommit("origin-commit").
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
			Press(keys.Branches.RebaseBranch)

		t.ExpectPopup().Menu().
			Title(Equals("Rebase 'feature'")).
			Select(Contains("Rebase onto a tracking branch…")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Rebase 'feature' onto")).
			Lines(
				Contains("origin/feature (upstream)"),
				Contains("fork/feature"),
				Contains("Cancel"),
			).
			Select(Contains("fork/feature")).
			Confirm()

		t.Views().Commits().Lines(
			Contains("local-commit"),
			Contains("fork-commit"),
			Contains("initial"),
		)
	},
})
