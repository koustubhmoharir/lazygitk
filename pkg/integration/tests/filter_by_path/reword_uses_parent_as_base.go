package filter_by_path

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RewordUsesParentAsBase = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In path filter mode, reword rebases onto the target's parent, not onto the next filtered commit on the other side of a merge",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		// Fixed dates keep the log order the same in every run. The side and target
		// commits edit different lines of filterFile, so the merge is clean and a
		// rebase recreates it without conflicts. The merge's first parent is
		// "side work", so the path-filtered list is: target, side work, base work.
		// The row after the target is "side work", which is not an ancestor of it.
		shell.CreateFileAndAdd("filterFile", "top\nmiddle\nbottom\n")
		commitAtFixedDate(shell, "base work", "2001-01-01 12:00:00 +0000")
		shell.NewBranch("side")
		shell.UpdateFileAndAdd("filterFile", "top side\nmiddle\nbottom\n")
		commitAtFixedDate(shell, "side work", "2001-01-02 12:00:00 +0000")
		shell.Checkout("master")
		shell.UpdateFileAndAdd("filterFile", "top\nmiddle\nbottom target\n")
		commitAtFixedDate(shell, "target", "2001-01-03 12:00:00 +0000")
		shell.Checkout("side")
		shell.RunCommandWithEnv([]string{"git", "merge", "--no-ff", "-m", "merge target", "master"},
			fixedDateEnv("2001-01-04 12:00:00 +0000"))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// HEAD is the merge: its first parent is "side work", and "base work" is the
		// target's parent. The target is HEAD's second parent.
		sideOID := t.Git().GetCommitHash("HEAD^1")
		baseOID := t.Git().GetCommitHash("HEAD^1^")

		t.Views().Commits().
			Focus().
			Press(keys.Universal.FilteringMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Filtering")).
			Select(Contains("Enter path to filter by")).
			Confirm()
		t.ExpectPopup().Prompt().
			Title(Equals("Enter path:")).
			Type("filterF").
			SuggestionLines(Equals("filterFile")).
			ConfirmFirstSuggestion()

		t.Views().Commits().
			IsFocused().
			NavigateToLine(Contains("target")).
			Press(keys.Commits.RenameCommit)

		t.ExpectPopup().CommitMessagePanel().
			Clear().
			Type("new message").
			Confirm()

		t.Views().Commits().
			Lines(
				Contains(`new message`).IsSelected(),
				Contains(`side work`),
				Contains(`base work`),
			).
			Press(keys.Universal.Return)

		// The rebase kept the history: the target keeps its parent, the merge is
		// still a merge, and only the target's message changed.
		t.Views().Commits().
			Lines(
				Contains(`merge target`),
				Contains(`new message`),
				Contains(`side work`),
				Contains(`base work`),
			)
		t.Git().SameCommitAs("HEAD^2^", baseOID)
		t.Shell().RunCommandExpectError([]string{"git", "merge-base", "--is-ancestor", sideOID, "HEAD^2"})
		// The rebase must start from the target's parent. The next filtered row,
		// "side work", is not that parent, so a base on it would be wrong even though
		// --rebase-merges can keep this history's shape.
		t.Views().Extras().Content(Contains("git rebase --interactive")).Content(Contains(baseOID))
	},
})

// commitAtFixedDate commits the staged changes with fixed author and committer dates.
func commitAtFixedDate(shell *Shell, message string, date string) *Shell {
	return shell.RunCommandWithEnv([]string{"git", "commit", "-m", message}, fixedDateEnv(date))
}

func fixedDateEnv(date string) []string {
	return []string{
		"GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_DATE=" + date,
	}
}
