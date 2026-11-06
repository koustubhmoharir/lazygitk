package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// Fork: g m n refuses while tracked files have changes, and never stashes them.
var MoveCommitsRefusesTrackedChanges = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Moving commits to a new branch is refused while a tracked file has changes, and nothing is stashed",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.CloneIntoRemote("origin")
		shell.PushBranchAndSetUpstream("origin", "master")
		shell.NewBranch("feature")
		shell.EmptyCommit("feature branch commit")
		shell.PushBranchAndSetUpstream("origin", "feature")
		shell.CreateFileAndAdd("file1", "file1 content")
		shell.Commit("new commit 1")
		shell.EmptyCommit("new commit 2")
		shell.UpdateFile("file1", "file1 changed")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Lines(
				Contains("M file1"),
			)
		t.Views().Branches().
			Focus().
			Lines(
				Contains("feature ↑2").IsSelected(),
				Contains("master ✓"),
			).
			Press(keys.Branches.MoveCommitsToNewBranch)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Commit or discard the changes to tracked files first. Moving commits never stashes.")).
			Confirm()

		t.Views().Branches().
			Lines(
				Contains("feature ↑2").IsSelected(),
				Contains("master ✓"),
			)
		t.Views().Commits().
			Lines(
				Contains("new commit 2").IsSelected(),
				Contains("new commit 1"),
				Contains("feature branch commit"),
				Contains("initial commit"),
			)
		t.Views().Files().
			Lines(
				Contains("M file1"),
			)
		t.Views().Stash().IsEmpty()
	},
})
