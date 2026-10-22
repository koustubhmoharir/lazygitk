package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FindBaseCommitForFixup = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Finds the base commit to create a fixup for",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.NewBranch("mybranch").
			EmptyCommit("1st commit").
			CreateFileAndAdd("file1", "file1 content\n").
			Commit("2nd commit").
			CreateFileAndAdd("file2", "file2 content\n").
			Commit("3rd commit").
			UpdateFile("file1", "file1 changed content").
			UpdateFile("file2", "file2 changed content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Lines(
				Contains("3rd commit"),
				Contains("2nd commit"),
				Contains("1st commit"),
			)

		// Two changes from different commits: this fails
		t.Views().Files().
			Focus().
			Press(keys.Files.FindBaseCommitForFixup)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(
				MatchesRegexp("Multiple base commits found.*\n\n" +
					".*3rd commit\n" +
					".*2nd commit"),
			).
			Confirm()

		// Stage only one of the files: this succeeds
		t.Views().Files().
			IsFocused().
			NavigateToLine(Contains("file1")).
			PressPrimaryAction()
		t.Views().Files().Press(keys.Files.FindBaseCommitForFixup)

		t.Views().Commits().
			IsFocused().
			Lines(
				Contains("3rd commit"),
				Contains("2nd commit").IsSelected(),
				Contains("1st commit"),
			)
		stagedBlob := t.Git().GetCommitHash(":file1")
		head := t.Git().GetCommitHash("HEAD")
		t.Views().Commits().Press(keys.Commits.AmendToCommit)

		t.ExpectPopup().Confirmation().
			Title(Equals("Amend commit")).
			Content(Contains("Are you sure you want to amend this commit with your staged files?")).
			Confirm()

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Cannot amend to commit: commit tracked changes first")).
			Confirm()
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Historical amend changed HEAD despite remaining tracked changes")
		}
		if got := t.Git().GetCommitHash(":file1"); got != stagedBlob {
			t.Fail("Historical amend changed the staged file1 blob")
		}
		t.FileSystem().FileContent("file1", Equals("file1 changed content"))
		t.FileSystem().FileContent("file2", Equals("file2 changed content"))
		t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/stash"})
		t.Views().Files().Focus().Press(keys.Files.FindBaseCommitForFixup)
		t.Views().Commits().IsFocused().Lines(
			Contains("3rd commit"),
			Contains("2nd commit").IsSelected(),
			Contains("1st commit"),
		)
	},
})
