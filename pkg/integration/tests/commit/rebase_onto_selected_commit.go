package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RebaseOntoSelectedCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Rebase onto other-branch history with explicit ignored-file approval and fresh review",
	SetupConfig: func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd(".gitignore", "/ignored-target\n/unrelated-output\n").Commit("base").
			NewBranch("target").CreateFile("ignored-target", "target bytes\n").
			RunCommand([]string{"git", "add", "-f", "ignored-target"}).Commit("target commit").
			Checkout("master").CreateFileAndAdd("local-one", "one\n").Commit("local one").
			RunCommand([]string{"git", "branch", "interior", "HEAD"}).
			CreateFileAndAdd("local-two", "two\n").Commit("local two").
			CreateFile("ignored-target", "ignored user bytes\n").
			CreateFile("unrelated-output", "unrelated ignored bytes\n").
			CreateFile("untracked", "untracked user bytes\n").
			SetConfig("rebase.autoStash", "true").SetConfig("rebase.updateRefs", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		original := t.Git().GetCommitHash("HEAD")
		interior := t.Git().GetCommitHash("interior")
		target := t.Git().GetCommitHash("target")
		commits := t.Views().Commits()
		commits.Focus().NavigateToLine(Contains("local one")).Press(keys.Commits.MarkCommitAsBaseForRebase)
		t.Views().Information().Content(Contains("Marked a base commit for rebase"))
		t.Views().Branches().Focus().NavigateToLine(Contains("target")).PressEnter()
		history := t.Views().SubCommits()
		history.IsFocused().NavigateToLine(Contains("target commit"))
		open := func() {
			history.IsFocused().Press(keys.Universal.OptionMenu)
			t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Rebase current branch onto this commit").Confirm()
			t.ExpectPopup().Menu().Title(Contains("Rebase current branch onto this commit")).
				Select(Contains("Ordinary rebase")).Confirm()
		}
		confirmHistory := func() {
			t.ExpectPopup().Confirmation().Title(Equals("Rebase current branch?")).
				Content(Contains("master").Contains(target).Contains("marked base is not used")).Confirm()
		}
		assertUnchanged := func(contents string) {
			if got := t.Git().GetCommitHash("HEAD"); got != original {
				t.Fail("Rebase ran without current approval")
			}
			t.Git().CurrentBranchName("master")
			t.FileSystem().FileContent("ignored-target", Equals(contents))
			t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/stash"})
		}
		open()
		confirmHistory()
		risk := t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		t.Views().Menu().SelectedLine(Contains("Cancel")).
			Content(Contains("Approve ignored-file overwrite risk and rebase").Contains("ignored-target").DoesNotContain("unrelated-output"))
		assertUnchanged("ignored user bytes\n")
		// Enter on the default risk selection cancels; it must never approve.
		risk.Confirm()
		history.IsFocused()
		assertUnchanged("ignored user bytes\n")
		open()
		confirmHistory()
		risk = t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		t.Shell().UpdateFile("ignored-target", "new ignored bytes\n")
		risk.Select(Contains("Approve ignored-file overwrite risk and rebase")).Confirm()
		t.ExpectPopup().Confirmation().Title(Equals("Rebase changed; review again")).
			Content(Contains("previous approvals no longer apply").Contains(target)).Confirm()
		risk = t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		assertUnchanged("new ignored bytes\n")
		risk.Select(Contains("Approve ignored-file overwrite risk and rebase")).Confirm()
		commits.Focus().TopLines(Contains("local two"), Contains("local one"), Contains("target commit"))
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("interior"); got != interior {
			t.Fail("Rebase moved an additional branch")
		}
		if got := t.Git().GetCommitHash("target"); got != target {
			t.Fail("Rebase moved its target branch")
		}
		t.FileSystem().FileContent("ignored-target", Equals("target bytes\n")).
			FileContent("untracked", Equals("untracked user bytes\n")).
			FileContent("unrelated-output", Equals("unrelated ignored bytes\n"))
		t.Views().Information().Content(Contains("Marked a base commit for rebase"))
		t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/stash"})
	},
})
