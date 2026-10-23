package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RenameBranchAndPull = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Rename a branch to no longer match its upstream, then pull from the upstream",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")

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

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master"),
			).
			Press(keys.Branches.RenameBranch).
			Tap(func() {
				t.ExpectPopup().Confirmation().
					Title(Equals("Rename branch")).
					Content(Equals("This branch is tracking a remote. This action will only rename the local branch name, not the name of the remote branch. Continue?")).
					Confirm()

				t.ExpectPopup().Prompt().
					Title(Contains("Enter new branch name")).
					InitialText(Equals("master")).
					Type("-local").
					Confirm()
				t.ExpectPopup().Confirmation().Title(Equals("Rename branch")).
					Content(Contains(`"master"`).Contains(`"master-local"`).Contains(sourceOID)).Confirm()
			}).
			Lines(Contains("master-local").IsSelected()).
			Press(keys.Universal.Pull)
		// A selected branch that is checked out uses native current integration.
		t.ReviewCurrentBranchPull("master-local", "origin", "master", sourceOID, targetOID).Confirm()

		t.Views().Commits().
			Lines(
				Contains("two"),
				Contains("one"),
			)
	},
})
