package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RebaseOntoDirectoryRenameIgnoredFile = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Rebasing onto a commit that renames a directory reviews the ignored file that Git's replay would create under the new name; Cancel keeps it",
	SetupConfig: func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd(".gitignore", "/new/b\n").
			CreateFileAndAdd("old/a", "base\n").
			Commit("base").
			NewBranch("topic").
			CreateFileAndAdd("old/b", "topic\n").
			Commit("addition").
			Checkout("master").
			RunCommand([]string{"git", "mv", "old", "new"}).
			Commit("rename").
			Checkout("topic").
			CreateFile("new/b", "precious ignored bytes\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		original := t.Git().GetCommitHash("HEAD")
		t.Views().Branches().Focus().NavigateToLine(Contains("master")).PressEnter()
		history := t.Views().SubCommits()
		history.IsFocused().NavigateToLine(Contains("rename"))
		history.IsFocused().Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Rebase current branch onto this commit").Confirm()
		t.ExpectPopup().Menu().Title(Contains("Rebase current branch onto this commit")).
			Select(Contains("Ordinary rebase")).Confirm()
		t.ExpectPopup().Confirmation().Title(Equals("Rebase current branch?")).
			Content(Contains("topic").Contains("marked base is not used")).Confirm()

		// Git's replay of old/b creates new/b under the renamed directory, which holds the ignored file.
		risk := t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		t.Views().Menu().SelectedLine(Contains("Approve ignored-file overwrite risk and rebase")).
			Content(Contains("Cancel").Contains("new/b"))
		risk.Select(MatchesRegexp(`^[[:space:]]*Cancel[[:space:]]*$`)).Confirm()

		if got := t.Git().GetCommitHash("HEAD"); got != original {
			t.Fail("a cancelled rebase moved HEAD: " + got)
		}
		t.Git().CurrentBranchName("topic")
		t.FileSystem().FileContent("new/b", Equals("precious ignored bytes\n"))
	},
})
