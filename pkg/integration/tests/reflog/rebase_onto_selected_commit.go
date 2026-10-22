package reflog

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RebaseOntoSelectedCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Rebase onto a reflog commit after its branch has been deleted",
	SetupConfig: func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("base-file", "base contents\n").Commit("base").
			NewBranch("gone").
			CreateFileAndAdd("orphan-file", "orphan target contents\n").Commit("orphan target").
			Checkout("master").
			RunCommand([]string{"git", "branch", "-D", "gone"}).
			CreateFileAndAdd("local-file", "local change contents\n").Commit("local change").
			CreateFile("untracked", "preserve this file\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		target := t.Git().GetCommitHash("HEAD@{2}")
		if t.Git().GetCommitHash("HEAD^") == target {
			t.Fail("reflog target unexpectedly matches the local commit's parent")
		}

		reflog := t.Views().ReflogCommits()
		reflog.Focus().NavigateToLine(Contains("commit: orphan target"))
		reflog.Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Rebase current branch onto this commit").Confirm()
		t.ExpectPopup().Menu().Title(Contains("Rebase current branch onto this commit")).
			Select(Contains("Ordinary rebase")).Confirm()
		t.ExpectPopup().Confirmation().
			Title(Equals("Rebase current branch?")).
			Content(Contains("master").Contains(target)).
			Confirm()

		t.Views().Commits().Focus().TopLines(
			Contains("local change"),
			Contains("orphan target"),
			Contains("base"),
		)
		t.Git().CurrentBranchName("master")
		t.FileSystem().
			FileContent("orphan-file", Equals("orphan target contents\n")).
			FileContent("local-file", Equals("local change contents\n")).
			FileContent("untracked", Equals("preserve this file\n"))
		t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/heads/gone"})
		t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/stash"})
	},
})
