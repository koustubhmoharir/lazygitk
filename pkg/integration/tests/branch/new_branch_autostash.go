package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var NewBranchAutostash = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse a branch checkout that would overwrite dirty worktree changes without stashing",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "a\n\nb")
		shell.Commit("add file")
		shell.UpdateFileAndAdd("file", "a\n\nc")
		shell.Commit("edit last line")

		shell.Checkout("HEAD^")
		shell.UpdateFile("file", "b\n\nb")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")

		t.Views().Files().
			Lines(
				Contains("file"),
			)

		t.Views().Branches().
			Focus().
			Lines(
				MatchesRegexp(`\*.*HEAD`).IsSelected(),
				Contains("master"),
			).
			NavigateToLine(Contains("master")).
			Press(keys.Universal.New)

		t.ExpectPopup().Prompt().
			Title(Contains("New branch name (branch is off of 'master')")).
			Type("new-branch").
			Confirm()

		t.Common().ReviewBranchCreateMenu("refs/heads/master", t.Git().GetCommitHash("refs/heads/master"), "HEAD", t.Git().GetCommitHash("HEAD")).
			Select(MatchesRegexp(`^[[:space:]]*Do not set up upstream tracking`)).Confirm()

		t.ExpectPopup().Alert().Cancel()

		t.Common().AssertBranchRefAbsent("new-branch").AssertNoStashes().AssertRefOID("HEAD", headOID)
		t.Git().CurrentBranchName("HEAD")
		t.FileSystem().FileContent("file", Equals("b\n\nb"))

		t.Views().Branches().
			Lines(
				Contains("new-branch").IsSelected(),
				Contains("master"),
			)

		t.Git().CurrentBranchName("new-branch")

		t.Views().Files().
			Lines(
				Contains("file"),
			)
	},
})
