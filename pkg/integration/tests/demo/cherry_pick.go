package demo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CherryPick = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cherry pick",
	ExtraCmdArgs: []string{},
	Skip:         false,
	IsDemo:       true,
	SetupConfig: func(config *config.AppConfig) {
		setDefaultDemoConfig(config)
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateNCommitsWithRandomMessages(50)

		shell.
			EmptyCommit("Fix bug in timezone conversion.").
			NewBranch("hotfix/fix-bug").
			NewBranch("feature/user-module").
			Checkout("hotfix/fix-bug").
			EmptyCommit("Integrate support for markdown in user posts").
			EmptyCommit("Remove unused code and libraries").
			Checkout("feature/user-module").
			EmptyCommit("Handle session timeout gracefully").
			EmptyCommit("Add Webpack for asset bundling").
			Checkout("hotfix/fix-bug")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		destinationOID := t.Git().GetCommitHash("hotfix/fix-bug")
		handleSessionOID := t.Git().GetCommitHash("feature/user-module^")
		addWebpackOID := t.Git().GetCommitHash("feature/user-module")
		t.SetCaptionPrefix("Cherry pick commits from another branch")
		t.Wait(1000)

		t.Views().Branches().
			Focus().
			Lines(
				Contains("hotfix/fix-bug"),
				Contains("feature/user-module"),
				Contains("master"),
			).
			SelectNextItem().
			Wait(300).
			PressEnter()

		subCommits := t.Views().SubCommits().
			IsFocused().
			TopLines(
				Contains("Add Webpack for asset bundling").IsSelected(),
				Contains("Handle session timeout gracefully"),
				Contains("Fix bug in timezone conversion."),
			).
			// select the two commits at the top
			Press(keys.Universal.RangeSelectDown).
			TopLines(
				Contains("Add Webpack for asset bundling").IsSelected(),
				Contains("Handle session timeout gracefully").IsSelected(),
				Contains("Fix bug in timezone conversion."),
			)

		t.Common().CherryPickSelection(subCommits)
		t.Wait(1000)
		t.Common().ConfirmCherryPickList("refs/heads/hotfix/fix-bug", destinationOID,
			CherryPickReviewRow(1, handleSessionOID, "Handle session timeout gracefully"),
			CherryPickReviewRow(2, addWebpackOID, "Add Webpack for asset bundling"),
		)

		t.Views().Commits().
			TopLines(
				Contains("Add Webpack for asset bundling"),
				Contains("Handle session timeout gracefully"),
				Contains("Remove unused code and libraries"),
				Contains("Integrate support for markdown in user posts"),
				Contains("Fix bug in timezone conversion."),
			)
	},
})
