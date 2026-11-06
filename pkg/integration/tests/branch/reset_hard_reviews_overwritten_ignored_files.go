package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// Fork: a hard reset reviews the untracked and ignored files that the target
// commit tracks at the same paths. Cancel leaves them in place; approval resets.
var ResetHardReviewsOverwrittenIgnoredFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Hard reset lists the ignored file that the target would overwrite; Cancel keeps it and approval resets",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "file")
		shell.Commit("base")
		shell.NewBranch("other")
		shell.CreateFileAndAdd("generated.txt", "committed generated")
		shell.Commit("other commit")
		shell.Checkout("master")
		shell.CreateFileAndAdd(".gitignore", "generated.txt\n")
		shell.Commit("ignore generated.txt")
		shell.CreateFile("generated.txt", "local ignored")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		openReset := func() {
			t.Views().Branches().
				Focus().
				NavigateToLine(Contains("other")).
				Press(keys.Commits.ViewResetOptions)

			t.ExpectPopup().Menu().
				Title(Contains("Reset to other")).
				Select(Contains("Hard reset")).
				Confirm()
		}
		headOID := t.Git().GetCommitHash("HEAD")

		openReset()
		t.ExpectPopup().Menu().
			Title(Equals("Untracked or ignored files will be overwritten or removed")).
			Content(Contains(`"generated.txt"`)).
			Cancel()

		t.FileSystem().FileContent("generated.txt", Equals("local ignored"))
		if t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Cancelling the hard reset review moved HEAD")
		}

		openReset()
		t.ExpectPopup().Menu().
			Title(Equals("Untracked or ignored files will be overwritten or removed")).
			Content(Contains(`"generated.txt"`)).
			Select(Contains("Approve")).
			Confirm()

		t.Views().Commits().
			Lines(
				Contains("other commit"),
				Contains("base"),
			)
		t.FileSystem().FileContent("generated.txt", Equals("committed generated"))
	},
})
