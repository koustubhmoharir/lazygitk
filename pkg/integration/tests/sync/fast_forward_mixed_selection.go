package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardMixedSelection = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Fast-forward the behind branch of a mixed selection and list only the diverged and upstream-less branches as not fast-forwarded",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")

		shell.NewBranch("f-behind")
		shell.EmptyCommit("behind-two")
		shell.Checkout("master")

		shell.NewBranch("f-diverged")
		shell.EmptyCommit("diverged-two")
		shell.Checkout("master")

		shell.NewBranch("f-up-to-date")
		shell.Checkout("master")

		shell.NewBranch("f-no-upstream")
		shell.Checkout("master")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("f-behind", "origin/f-behind")
		shell.SetBranchUpstream("f-diverged", "origin/f-diverged")
		shell.SetBranchUpstream("f-up-to-date", "origin/f-up-to-date")

		// f-behind is behind its tracking ref
		shell.RunCommand([]string{"git", "branch", "-f", "f-behind", "f-behind~1"})

		// f-diverged has its own commit and lacks the tracking ref's commit
		shell.Checkout("f-diverged")
		shell.HardReset("HEAD~1")
		shell.EmptyCommit("diverged-local")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		behindTarget := t.Git().GetCommitHash("refs/remotes/origin/f-behind")
		divergedOID := t.Git().GetCommitHash("refs/heads/f-diverged")
		upToDateOID := t.Git().GetCommitHash("refs/heads/f-up-to-date")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		headOID := t.Git().GetCommitHash("HEAD")

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("f-behind ↓1"),
				Contains("f-diverged ↓1↑1"),
				Contains("f-no-upstream"),
				Contains("f-up-to-date ✓"),
			).
			NavigateToLine(Contains("f-behind")).
			Press(keys.Universal.ToggleRangeSelect).
			SelectNextItem().
			SelectNextItem().
			SelectNextItem().
			Lines(
				Contains("master"),
				Contains("f-behind ↓1").IsSelected(),
				Contains("f-diverged ↓1↑1").IsSelected(),
				Contains("f-no-upstream").IsSelected(),
				Contains("f-up-to-date ✓").IsSelected(),
			).
			Press(keys.Branches.FastForward)

		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).
			Content(Contains("f-diverged: ").Contains("has diverged").
				Contains("f-no-upstream: ").Contains("no upstream").
				DoesNotContain("f-behind").DoesNotContain("f-up-to-date")).
			Confirm()

		if got := t.Git().GetCommitHash("refs/heads/f-behind"); got != behindTarget {
			t.Fail("behind branch was not fast-forwarded to its upstream: " + got)
		}
		if t.Git().GetCommitHash("refs/heads/f-diverged") != divergedOID || t.Git().GetCommitHash("refs/heads/f-up-to-date") != upToDateOID {
			t.Fail("mixed selection moved the diverged or up-to-date branch")
		}
		if t.Git().GetCommitHash("refs/heads/master") != masterOID || t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("mixed selection changed master or HEAD")
		}
		t.Views().Branches().
			Lines(
				Contains("master"),
				Contains("f-behind ✓"),
				Contains("f-diverged ↓1↑1"),
				Contains("f-no-upstream"),
				Contains("f-up-to-date ✓"),
			)
	},
})
