package interactive_rebase

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var AmendToIgnoredFileReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Amending staged files to a middle commit reviews the ignored files its rebase would overwrite before creating the fixup; Cancel creates no commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("collision", "base contents\n").
			Commit("base").
			CreateFileAndAdd("middle-file", "middle\n").
			Commit("middle").
			DeleteFileAndAdd("collision").
			CreateFileAndAdd(".gitignore", "/collision\n").
			Commit("head").
			CreateFile("collision", "valuable ignored contents\n").
			UpdateFileAndAdd("middle-file", "amended\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		original := t.Git().GetCommitHash("HEAD")
		commits := t.Views().Commits()
		amendTo := func() {
			commits.Focus().NavigateToLine(Contains("middle")).Press(keys.Commits.AmendToCommit)
			t.ExpectPopup().Confirmation().
				Title(Equals("Amend commit")).
				Content(Contains("Are you sure you want to amend this commit with your staged files?")).
				Confirm()
		}

		amendTo()
		risk := t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		t.Views().Menu().SelectedLine(Contains("Approve ignored-file overwrite risk and rebase")).
			Content(Contains("Cancel").Contains("\"collision\""))
		risk.Select(MatchesRegexp(`^[[:space:]]*Cancel[[:space:]]*$`)).Confirm()

		// Cancel is refused before the fixup exists: no commit is made and the ignored bytes stay.
		if got := t.Git().GetCommitHash("HEAD"); got != original {
			t.Fail("a cancelled amend created a commit: " + got)
		}
		t.FileSystem().FileContent("collision", Equals("valuable ignored contents\n"))
		commits.Focus().Lines(Contains("head"), Contains("middle"), Contains("base"))

		amendTo()
		t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten")).
			Select(Contains("Approve ignored-file overwrite risk and rebase")).
			Confirm()
		commits.Focus().Lines(Contains("head"), Contains("middle"), Contains("base"))
		if got := t.Git().GetCommitHash("HEAD"); got == original {
			t.Fail("approved amend did not rewrite HEAD")
		}
	},
})
