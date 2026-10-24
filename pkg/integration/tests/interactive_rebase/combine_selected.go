package interactive_rebase

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CombineSelected = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Combine at the selected ancestor with editable full messages, cancellation and dirty-tree draft recovery",
	SetupConfig: func(config *config.AppConfig) { config.GetUserConfig().Git.Log.ShowGraph = "never" },
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("first", "first\n").Commit("first subject\n\nfirst full description").
			CreateFileAndAdd("second", "second\n").Commit("second subject\n\nsecond full description").
			CreateFileAndAdd("tail", "tail\n").Commit("unselected tail").
			RunCommand([]string{"git", "branch", "unchanged", "HEAD"}).SetConfig("rebase.updateRefs", "true").SetConfig("rebase.autoStash", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		original := t.Git().GetCommitHash("HEAD")
		keeper := t.Git().GetCommitHash("HEAD~2")
		second := t.Git().GetCommitHash("HEAD~1")
		commits := t.Views().Commits()
		commits.Focus().NavigateToLine(Contains("second subject")).Press(keys.Universal.RangeSelectDown)
		open := func() {
			commits.IsFocused().Press(keys.Universal.OptionMenu)
			t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Combine selected commits").Confirm()
		}
		review := func() *ConfirmationDriver {
			return t.ExpectPopup().Confirmation().Title(Equals("Review commit combination")).Content(Contains(keeper).Contains(second).Contains(original).Contains("Unselected commits remain separate"))
		}
		unchanged := func() {
			if got := t.Git().GetCommitHash("HEAD"); got != original {
				t.Fail("combine mutated HEAD before editor approval")
			}
		}
		open()
		review().Cancel()
		unchanged()
		open()
		review().Confirm()
		panel := t.ExpectPopup().CommitMessagePanel().InitialText(Equals("first subject"))
		panel.SwitchToDescription().Content(Contains("first subject").Contains("first full description").Contains("second subject").Contains("second full description")).SwitchToSummary()
		unchanged()
		panel.Clear().Type("approved combined subject")
		t.Shell().UpdateFile("tail", "dirty tail\n")
		panel.Confirm()
		t.ExpectPopup().Alert().Title(Equals("Error")).Content(Contains("commit your changes first")).Confirm()
		panel = t.ExpectPopup().CommitMessagePanel().InitialText(Equals("approved combined subject"))
		unchanged()
		t.Shell().RunCommand([]string{"git", "checkout", "--", "tail"})
		t.Shell().EmptyCommit("external after review")
		freshHead := t.Git().GetCommitHash("HEAD")
		panel.Confirm()
		stale := t.ExpectPopup().Menu().Title(Equals("Review commit combination"))
		t.Views().Menu().Content(Contains("captured history").Contains("changed"))
		// A failed fresh capture must retain the draft too, not only a failed
		// initial execution. No review authorizes an automatic retry.
		t.Shell().UpdateFile("tail", "dirty during fresh review\n")
		stale.Select(Contains("Capture and review current history again")).Confirm()
		t.ExpectPopup().Alert().Title(Equals("Error")).Content(Contains("commit your changes first")).Confirm()
		panel = t.ExpectPopup().CommitMessagePanel().InitialText(Equals("approved combined subject"))
		t.Shell().RunCommand([]string{"git", "checkout", "--", "tail"})
		panel.Confirm()
		t.ExpectPopup().Menu().Title(Equals("Review commit combination")).
			Select(Contains("Capture and review current history again")).Confirm()
		t.ExpectPopup().Confirmation().Title(Equals("Review commit combination")).
			Content(Contains(freshHead).Contains(keeper).Contains(second).Contains("fresh state")).Confirm()
		t.ExpectPopup().CommitMessagePanel().InitialText(Equals("approved combined subject")).Confirm()
		commits.Focus().Lines(Contains("external after review"), Contains("unselected tail"), Contains("approved combined subject"))
		if got := t.Git().GetCommitHash("unchanged"); got != original {
			t.Fail("combine moved an unrelated branch")
		}
		t.Shell().RunCommand([]string{"git", "log", "--reverse", "--format=%s", "--output=actual-log"})
		t.FileSystem().FileContent("actual-log", Equals("approved combined subject\nunselected tail\nexternal after review\n"))
		t.Shell().RunCommand([]string{"git", "show", "-s", "--format=%B", "--output=combined-message", "HEAD~2"})
		t.FileSystem().FileContent("combined-message", Contains("approved combined subject\n\nfirst subject\n\nfirst full description\n\n\nsecond subject\n\nsecond full description"))
		t.FileSystem().FileContent("first", Equals("first\n")).FileContent("second", Equals("second\n")).FileContent("tail", Equals("tail\n"))
		t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/stash"})
	},
})
