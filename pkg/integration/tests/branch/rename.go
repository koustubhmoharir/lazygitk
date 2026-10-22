package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var Rename = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Review a literal branch rename without changing its commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master"),
			).
			Press(keys.Branches.RenameBranch).
			Tap(func() {
				t.ExpectPopup().Prompt().
					Title(Contains("Enter new branch name")).
					InitialText(Equals("master")).
					Clear().
					Type("new-branch-name").
					Confirm()
				t.ExpectPopup().Confirmation().
					Title(Equals("Rename branch")).
					Content(Contains("master").Contains("new-branch-name").Contains(headOID)).
					Confirm()
			}).
			Lines(
				Contains("new-branch-name"),
			)
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("Renaming a branch changed the checked-out commit")
		}
	},
})
