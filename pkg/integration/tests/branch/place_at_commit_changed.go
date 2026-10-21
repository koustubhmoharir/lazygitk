package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PlaceAtCommitChanged = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Reconfirm branch placement after the branch changes while the warning is open",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("oldest")
		shell.EmptyCommit("middle")
		shell.EmptyCommit("newest")
		shell.RunCommand([]string{"git", "branch", "topic", "HEAD~1"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		oldest := t.Git().GetCommitHash("HEAD~2")
		middle := t.Git().GetCommitHash("HEAD~1")
		head := t.Git().GetCommitHash("HEAD")
		openPlacement := func() {
			t.Views().Commits().Focus().NavigateToLine(Contains("oldest"))
			t.Views().Commits().Press(keys.Universal.OptionMenu)
			t.ExpectPopup().Menu().Title(Equals("Keybindings")).Filter("Create/move branch here").Confirm()
			t.ExpectPopup().Prompt().Title(Contains("Local branch name")).Type("topic").Confirm()
		}

		openPlacement()
		initialMoveWarning := t.ExpectPopup().Confirmation().
			Title(Equals("Move existing branch?")).
			Content(Contains("topic").Contains(middle[:7]).Contains(oldest[:7]))
		// The branch changes after the warning was rendered. Accepting its stale
		// plan must produce a fresh review using the current ref value.
		t.Shell().RunCommand([]string{"git", "branch", "-f", "topic", "HEAD"})
		initialMoveWarning.Confirm()
		updatedMoveWarning := t.ExpectPopup().Confirmation().
			Title(Equals("Branch changed; review again")).
			Content(Contains("topic").Contains(head[:7]).Contains(oldest[:7]))
		updatedMoveWarning.Cancel()
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("refs/heads/topic"); got != head {
			t.Fail("Canceling the refreshed branch move changed the externally updated branch")
		}

		openPlacement()
		moveWarning := t.ExpectPopup().Confirmation().
			Title(Equals("Move existing branch?")).
			Content(Contains("topic").Contains(head[:7]).Contains(oldest[:7]))
		t.Shell().RunCommand([]string{"git", "branch", "-D", "topic"})
		moveWarning.Confirm()
		recreationWarning := t.ExpectPopup().Confirmation().
			Title(Equals("Branch changed; review again")).
			Content(Contains("no longer exists").Contains(oldest[:7]))
		t.Shell().RunCommandExpectError([]string{"git", "show-ref", "--verify", "--quiet", "refs/heads/topic"})
		if t.Git().GetCommitHash("HEAD") != head {
			t.Fail("Branch placement changed HEAD before the fresh creation was confirmed")
		}
		recreationWarning.Confirm()
		t.Views().Commits().IsFocused()
		if got := t.Git().GetCommitHash("refs/heads/topic"); got != oldest {
			t.Fail("Confirmed fresh branch placement did not create topic at the selected commit")
		}
		t.Git().CurrentBranchName("master")
		if got := t.Git().GetCommitHash("HEAD"); got != head {
			t.Fail("Branch placement changed HEAD")
		}
		t.Views().Commits().IsFocused().SelectedLine(Contains("oldest"))
	},
})
