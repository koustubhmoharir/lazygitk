package interactive_rebase

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SquashFixupsAboveIgnoredFileReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Squashing fixups above a commit reviews the ignored files its rebase would overwrite; Cancel changes nothing, and approval squashes",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		// The rebase rewinds to base, which tracks collision. HEAD deletes it and
		// ignores it, so the user's bytes are ignored and not tracked.
		shell.CreateFileAndAdd("collision", "base contents\n").
			Commit("base").
			CreateFileAndAdd("middle-file", "middle\n").
			Commit("middle").
			DeleteFileAndAdd("collision").
			CreateFileAndAdd(".gitignore", "/collision\n").
			Commit("head").
			CreateFile("collision", "valuable ignored contents\n").
			CreateFileAndAdd("fixup-file", "fixup content\n").
			Commit("fixup! middle")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		original := t.Git().GetCommitHash("HEAD")
		commits := t.Views().Commits()
		squashAbove := func() {
			commits.Focus().NavigateToLine(MatchesRegexp(`[^!] middle$`)).Press(keys.Commits.SquashAboveCommits)
			t.ExpectPopup().Menu().
				Title(Equals("Apply fixup commits")).
				Select(Contains("Above the selected commit")).
				Confirm()
		}

		squashAbove()
		risk := t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		t.Views().Menu().SelectedLine(Contains("Approve ignored-file overwrite risk and rebase")).
			Content(Contains("Cancel").Contains("\"collision\""))
		risk.Select(MatchesRegexp(`^[[:space:]]*Cancel[[:space:]]*$`)).Confirm()

		if got := t.Git().GetCommitHash("HEAD"); got != original {
			t.Fail("a cancelled squash moved HEAD: " + got)
		}
		t.FileSystem().FileContent("collision", Equals("valuable ignored contents\n"))
		commits.Focus().Lines(Contains("fixup! middle"), Contains("head"), Contains("middle"), Contains("base"))

		squashAbove()
		t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten")).
			Select(Contains("Approve ignored-file overwrite risk and rebase")).
			Confirm()
		commits.Focus().Lines(Contains("head"), Contains("middle"), Contains("base"))
		if got := t.Git().GetCommitHash("HEAD"); got == original {
			t.Fail("approved squash did not rewrite HEAD")
		}
	},
})
