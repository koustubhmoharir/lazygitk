package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullAndSetUpstream = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Explicitly set the branch upstream, then review a fetched commit before pulling",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")

		shell.CloneIntoRemote("origin")

		// remove the 'two' commit so that we have something to pull from the remote
		shell.HardReset("HEAD^")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("HEAD")
		targetOID := t.Git().GetCommitHash("refs/remotes/origin/master")
		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Equals("repo → master"))

		// Tracking setup is a separate, explicit metadata action, never a side
		// effect of fetching or integrating a branch with no saved upstream.
		t.Views().Branches().Focus().Press(keys.Branches.SetUpstream)
		t.ExpectPopup().Menu().Title(Equals("Upstream options")).
			Select(Contains(" Set upstream of selected branch")).Confirm()
		t.ExpectPopup().Prompt().Title(Contains("Full upstream tracking ref")).
			Type("refs/remotes/origin/master").Confirm()
		t.ExpectPopup().Confirmation().Title(Equals("Review selected branch upstream")).
			Content(Contains(sourceOID).Contains(targetOID).Contains("refs/remotes/origin/master")).Confirm()
		t.Views().Status().Content(Equals("↓1 repo → master"))
		if t.Git().GetCommitHash("HEAD") != sourceOID {
			t.Fail("Setting upstream moved HEAD")
		}
		t.Views().Files().Focus().Press(keys.Universal.Pull)
		review := t.ReviewCurrentBranchPull("master", "origin", "master", sourceOID, targetOID)
		if t.Git().GetCommitHash("HEAD") != sourceOID {
			t.Fail("Fetching/reviewing moved HEAD before integration approval")
		}
		review.Confirm()

		t.Views().Commits().
			Lines(
				Contains("two"),
				Contains("one"),
			)

		t.Views().Status().Content(Equals("✓ repo → master"))
	},
})
