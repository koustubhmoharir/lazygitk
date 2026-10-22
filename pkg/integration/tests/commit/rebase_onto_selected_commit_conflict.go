package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RebaseOntoSelectedCommitConflict = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Selected-commit rebase refuses tracked dirt and supports normal conflict abort and continuation",
	SetupConfig: func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("shared", "base\n").Commit("base").
			NewBranch("target").UpdateFileAndAdd("shared", "target\n").Commit("target conflict").
			Checkout("master").UpdateFileAndAdd("shared", "local\n").Commit("local conflict").
			SetConfig("rebase.autoStash", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		original := t.Git().GetCommitHash("HEAD")
		t.Views().Branches().Focus().NavigateToLine(Contains("target")).PressEnter()
		history := t.Views().SubCommits()
		history.IsFocused().NavigateToLine(Contains("target conflict"))
		open := func() {
			history.IsFocused().Press(keys.Universal.OptionMenu)
			t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Rebase current branch onto this commit").Confirm()
			t.ExpectPopup().Menu().Title(Contains("Rebase current branch onto this commit")).
				Select(Contains("Ordinary rebase")).Confirm()
		}
		t.Shell().UpdateFile("shared", "dirty tracked bytes\n")
		open()
		t.ExpectPopup().Alert().Title(Equals("Error")).Content(Contains("commit your changes first")).Confirm()
		t.FileSystem().FileContent("shared", Equals("dirty tracked bytes\n"))
		t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/stash"})
		t.Shell().RunCommand([]string{"git", "restore", "--worktree", "shared"})
		open()
		t.ExpectPopup().Confirmation().Title(Equals("Rebase current branch?")).Content(Contains("master")).Confirm()
		t.ExpectPopup().Menu().Title(Equals("Conflicts!")).Select(Contains("Abort")).Confirm()
		if got := t.Git().GetCommitHash("HEAD"); got != original {
			t.Fail("Abort failed to restore the source")
		}
		history.IsFocused()
		open()
		t.ExpectPopup().Confirmation().Title(Equals("Rebase current branch?")).Content(Contains("master")).Confirm()
		t.Common().AcknowledgeConflicts()
		// Resolve and stage explicitly: start-only guards must not reject this
		// staged state when continuing the operation through its normal menu.
		t.Shell().UpdateFileAndAdd("shared", "resolved\n")
		t.Views().Files().Focus().Press(keys.Universal.CreateRebaseOptionsMenu)
		t.ExpectPopup().Menu().Title(Contains("Rebase")).Select(Contains("continue")).Confirm()
		t.Views().Information().Content(DoesNotContain("Rebasing"))
		t.Views().Commits().Focus().TopLines(Contains("local conflict"), Contains("target conflict"))
		t.FileSystem().FileContent("shared", Equals("resolved\n"))
	},
})
