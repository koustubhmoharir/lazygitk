package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var Delete = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Try all combination of local and remote branch deletions",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "recency"
		config.GetUserConfig().Git.RemoteBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		shell.
			CloneIntoRemote("origin").
			EmptyCommit("blah").
			NewBranch("branch-one").
			EmptyCommit("on branch-one 01").
			PushBranchAndSetUpstream("origin", "branch-one").
			EmptyCommit("on branch-one 02").
			Checkout("master").
			Merge("branch-one"). // branch-one is contained in master, so no delete confirmation
			NewBranch("branch-two").
			EmptyCommit("on branch-two 01").
			PushBranchAndSetUpstream("origin", "branch-two"). // branch-two is contained in its own upstream, so no delete confirmation either
			NewBranchFrom("branch-three", "master").
			EmptyCommit("on branch-three 01").
			NewBranch("current-head"). // branch-three is contained in the current head, so no delete confirmation
			EmptyCommit("on current-head").
			NewBranchFrom("branch-four", "master").
			EmptyCommit("on branch-four 01").
			PushBranchAndSetUpstream("origin", "branch-four").
			EmptyCommit("on branch-four 02"). // branch-four is not contained in any of these, so we get a delete confirmation
			NewBranchFrom("branch-five", "master").
			EmptyCommit("on branch-five 01").
			PushBranchAndSetUpstream("origin", "branch-five"). // branch-five is contained in its own upstream
			NewBranchFrom("branch-six", "master").
			EmptyCommit("on branch-six 01").
			PushBranchAndSetUpstream("origin", "branch-six").
			EmptyCommit("on branch-six 02"). // branch-six is not contained in any of these, so we get a delete confirmation
			Checkout("current-head")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		currentHeadOID := t.Git().GetCommitHash("refs/heads/current-head")
		branchFourOID := t.Git().GetCommitHash("refs/heads/branch-four")
		branchFiveOID := t.Git().GetCommitHash("refs/heads/branch-five")
		branchSixOID := t.Git().GetCommitHash("refs/heads/branch-six")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("current-head").IsSelected(),
				Contains("branch-six ↑1"),
				Contains("branch-five ✓"),
				Contains("branch-four ↑1"),
				Contains("branch-three"),
				Contains("branch-two ✓"),
				Contains("master"),
				Contains("branch-one ↑1"),
			).

			// The checked-out branch is reserved by the main worktree.
			Press(keys.Universal.Remove).
			Tap(func() {
				t.Common().ReviewBranchDeleteMenu("Delete branch 'current-head'?", "refs/heads/current-head", currentHeadOID, "refs/heads/current-head", currentHeadOID, true).Cancel()
			}).

			// Delete branch-four. This is the only branch that is not fully merged, so we get
			// a confirmation popup.
			NavigateToLine(Contains("branch-four")).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'branch-four'?")).
					Select(MatchesRegexp(`^[[:space:]]*c Delete local branch`)).
					Confirm()
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Force delete branch")).
					Content(Contains("refs/heads/branch-four").Contains(branchFourOID).Contains("not preserved by the remaining reviewed HEAD, upstream or main refs")).
					Confirm()
			}).
			EventuallyContent(Contains("branch-three").DoesNotContain("branch-four")).Lines(
			Contains("current-head"),
			Contains("branch-six ↑1"),
			Contains("branch-five ✓"),
			Contains("branch-three").IsSelected(),
			Contains("branch-two ✓"),
			Contains("master"),
			Contains("branch-one ↑1"),
		).

			// Delete branch-three. This branch is contained in the current head, so this just works
			// without any confirmation.
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'branch-three'?")).
					Select(MatchesRegexp(`^[[:space:]]*c Delete local branch`)).
					Confirm()
			}).
			EventuallyContent(Contains("branch-two ✓").DoesNotContain("branch-three")).Lines(
			Contains("current-head"),
			Contains("branch-six ↑1"),
			Contains("branch-five ✓"),
			Contains("branch-two ✓").IsSelected(),
			Contains("master"),
			Contains("branch-one ↑1"),
		).

			// Delete branch-two. This branch is contained in its own upstream, so this just works
			// without any confirmation.
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'branch-two'?")).
					Select(MatchesRegexp(`^[[:space:]]*c Delete local branch`)).
					Confirm()
			}).
			EventuallyContent(Contains("master").DoesNotContain("branch-two")).Lines(
			Contains("current-head"),
			Contains("branch-six ↑1"),
			Contains("branch-five ✓"),
			Contains("master").IsSelected(),
			Contains("branch-one ↑1"),
		).

			// Delete remote branch of branch-one. We only get the normal remote branch confirmation for this one.
			SelectNextItem().
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'branch-one'?")).
					Select(MatchesRegexp(`^[[:space:]]*r Delete remote branch`)).
					Confirm()
			}).
			Tap(func() {
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Delete remote branch")).
					Content(Contains("refs/heads/branch-one").Contains(t.Git().GetCommitHash("refs/heads/branch-one")).Contains("Current HEAD:").Contains("Cached commit IDs may differ from the server's current tips")).
					Confirm()
			}).
			Tap(func() {
				checkRemoteBranches(t, keys, "origin", []string{
					"branch-five",
					"branch-four",
					"branch-six",
					"branch-two",
				})
			}).
			EventuallyContent(Contains("branch-one (upstream gone)")).Lines(
			Contains("current-head"),
			Contains("branch-six ↑1"),
			Contains("branch-five ✓"),
			Contains("master"),
			Contains("branch-one (upstream gone)").IsSelected(),
		).

			// Delete local branch of branch-one. Even though its upstream is gone, we don't get a confirmation
			// because it is contained in master.
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'branch-one'?")).
					Select(MatchesRegexp(`^[[:space:]]*c Delete local branch`)).
					Confirm()
			}).
			EventuallyContent(Contains("master").DoesNotContain("branch-one")).Lines(
			Contains("current-head"),
			Contains("branch-six ↑1"),
			Contains("branch-five ✓"),
			Contains("master").IsSelected(),
		).

			// Delete both local and remote branch of branch-six. We get the force-delete warning because it is not fully merged.
			NavigateToLine(Contains("branch-six")).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'branch-six'?")).
					Select(MatchesRegexp(`^[[:space:]]*b Delete local and remote branch`)).
					Confirm()
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Delete local and remote branch")).
					Content(Contains("refs/heads/branch-six").Contains(branchSixOID).Contains("origin -> refs/heads/branch-six").Contains("not preserved by the remaining reviewed HEAD, upstream or main refs").Contains("Cached commit IDs may differ from the server's current tips")).
					Confirm()
			}).
			EventuallyContent(Contains("branch-five ✓").DoesNotContain("branch-six")).Lines(
			Contains("current-head"),
			Contains("branch-five ✓").IsSelected(),
			Contains("master"),
		).

			// Delete both local and remote branch of branch-five. The upstream cache is removed too, so
			// its history is not preserved by any remaining reviewed ref.
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'branch-five'?")).
					Select(MatchesRegexp(`^[[:space:]]*b Delete local and remote branch`)).
					Confirm()
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Delete local and remote branch")).
					Content(Contains("refs/heads/branch-five").Contains(branchFiveOID).Contains("origin -> refs/heads/branch-five").Contains("not preserved by the remaining reviewed HEAD, upstream or main refs").Contains("Cached commit IDs may differ from the server's current tips")).
					Confirm()
			}).
			EventuallyContent(Contains("master").DoesNotContain("branch-five")).Lines(
			Contains("current-head"),
			Contains("master").IsSelected(),
		)
	},
})
