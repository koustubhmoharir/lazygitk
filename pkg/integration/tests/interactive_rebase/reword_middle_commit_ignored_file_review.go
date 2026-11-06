package interactive_rebase

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RewordMiddleCommitIgnoredFileReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Rewording a middle commit reviews the ignored files its rebase would overwrite; Cancel changes nothing, and approval runs the reword",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		// The rebase rewinds to base, which tracks collision. HEAD deletes it and
		// ignores it, so the user's bytes are ignored and not tracked.
		shell.CreateFileAndAdd("collision", "base contents\n").
			Commit("base").
			UpdateFileAndAdd("collision", "historical contents\n").
			Commit("middle").
			DeleteFileAndAdd("collision").
			CreateFileAndAdd(".gitignore", "/collision\n").
			Commit("head").
			CreateFile("collision", "valuable ignored contents\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		original := t.Git().GetCommitHash("HEAD")
		commits := t.Views().Commits()
		reword := func(message string) {
			commits.Focus().NavigateToLine(Contains("middle")).Press(keys.Commits.RenameCommit)
			t.ExpectPopup().CommitMessagePanel().
				Title(Equals("Reword commit")).
				InitialText(Equals("middle")).
				Clear().
				Type(message).
				Confirm()
		}
		assertUnchanged := func() {
			if got := t.Git().GetCommitHash("HEAD"); got != original {
				t.Fail("a cancelled reword changed HEAD: " + got)
			}
			t.FileSystem().FileContent("collision", Equals("valuable ignored contents\n"))
			commits.Focus().Lines(Contains("head"), Contains("middle"), Contains("base"))
		}

		reword("renamed middle")
		risk := t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		// The approval is selected, and the review names the file that would be overwritten.
		t.Views().Menu().SelectedLine(Contains("Approve ignored-file overwrite risk and rebase")).
			Content(Contains("Cancel").Contains("\"collision\""))
		risk.Select(MatchesRegexp(`^[[:space:]]*Cancel[[:space:]]*$`)).Confirm()
		assertUnchanged()

		reword("renamed middle")
		t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten")).
			Select(Contains("Approve ignored-file overwrite risk and rebase")).
			Confirm()
		commits.Focus().Lines(Contains("head"), Contains("renamed middle"), Contains("base"))
		if got := t.Git().GetCommitHash("HEAD"); got == original {
			t.Fail("approved reword did not rewrite HEAD")
		}
	},
})
