package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DeleteMultiple = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Try some combinations of local and remote branch deletions with a range selection of branches",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
		config.GetUserConfig().Git.RemoteBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		shell.
			CloneIntoRemote("origin").
			CloneIntoRemote("other-remote").
			EmptyCommit("blah").
			NewBranch("branch-01").
			EmptyCommit("on branch-01 01").
			PushBranchAndSetUpstream("origin", "branch-01").
			EmptyCommit("on branch-01 02").
			NewBranch("branch-02").
			EmptyCommit("on branch-02 01").
			PushBranchAndSetUpstream("origin", "branch-02").
			NewBranchFrom("branch-03", "master").
			EmptyCommit("on branch-03 01").
			NewBranch("current-head").
			EmptyCommit("on current-head").
			NewBranchFrom("branch-04", "master").
			EmptyCommit("on branch-04 01").
			PushBranchAndSetUpstream("other-remote", "branch-04").
			EmptyCommit("on branch-04 02").
			NewBranchFrom("branch-05", "master").
			EmptyCommit("on branch-05 01").
			PushBranchAndSetUpstream("origin", "branch-05").
			NewBranchFrom("branch-06", "master").
			EmptyCommit("on branch-06 01").
			PushBranch("origin", "branch-06").
			PushBranchAndSetUpstream("other-remote", "branch-06").
			EmptyCommit("on branch-06 02").
			Checkout("current-head")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branch04OID := t.Git().GetCommitHash("refs/heads/branch-04")
		branch05OID := t.Git().GetCommitHash("refs/heads/branch-05")
		branch06OID := t.Git().GetCommitHash("refs/heads/branch-06")
		currentHeadOID := t.Git().GetCommitHash("refs/heads/current-head")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("current-head").IsSelected(),
				Contains("branch-01 ↑1"),
				Contains("branch-02 ✓"),
				Contains("branch-03"),
				Contains("branch-04 ↑1"),
				Contains("branch-05 ✓"),
				Contains("branch-06 ↑1"),
				Contains("master"),
			).
			Press(keys.Universal.RangeSelectDown).

			// A selected range containing the checked-out branch cannot delete local refs.
			Press(keys.Universal.Remove).
			Tap(func() {
				t.Common().ReviewBranchDeleteMenu("Delete selected branches?", "refs/heads/current-head", currentHeadOID, "refs/heads/current-head", currentHeadOID, true).Cancel()
			}).

			// Delete branch-03 and branch-04. 04 is not fully merged, so we get
			// a confirmation popup.
			NavigateToLine(Contains("branch-03")).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete selected branches?")).
					Select(MatchesRegexp(`^[[:space:]]*c Delete local branches`)).
					Confirm()
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Force delete branch")).
					Content(Contains("not preserved by the remaining reviewed HEAD, upstream or main refs").Contains("refs/heads/branch-04").Contains(branch04OID)).
					Confirm()
			}).
			EventuallyContent(Contains("branch-05 ✓").DoesNotContain("branch-03").DoesNotContain("branch-04")).Lines(
			Contains("current-head"),
			Contains("branch-01 ↑1"),
			Contains("branch-02 ✓"),
			Contains("branch-05 ✓").IsSelected(),
			Contains("branch-06 ↑1"),
			Contains("master"),
		).

			// Delete remote branches of branch-05 and branch-06. They are on different remotes.
			NavigateToLine(Contains("branch-05")).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete selected branches?")).
					Select(MatchesRegexp(`^[[:space:]]*r Delete remote branches`)).
					Confirm()
			}).
			Tap(func() {
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Delete selected branches?")).
					Content(Contains("refs/heads/branch-05").Contains("origin -> refs/heads/branch-05").Contains("refs/heads/branch-06").Contains("other-remote -> refs/heads/branch-06").Contains("Cached commit IDs may differ from the server's current tips")).
					Confirm()
			}).
			Tap(func() {
				checkRemoteBranches(t, keys, "origin", []string{
					"branch-01",
					"branch-02",
					"branch-06",
				})
				checkRemoteBranches(t, keys, "other-remote", []string{
					"branch-04",
				})
			}).
			EventuallyContent(Contains("branch-05 (upstream gone)")).Lines(
			Contains("current-head"),
			Contains("branch-01 ↑1"),
			Contains("branch-02 ✓"),
			Contains("branch-05 (upstream gone)").IsSelected(),
			Contains("branch-06 (upstream gone)").IsSelected(),
			Contains("master"),
		).

			// The upstream refs can still be reviewed after the cached branch-05 ref was removed.
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete selected branches?")).
					Select(MatchesRegexp(`^[[:space:]]*b Delete local and remote branches`)).
					Confirm()
				confirmation := t.ExpectPopup().Confirmation()
				confirmation.Title(Equals("Delete selected branches?"))
				confirmation.Content(Contains("refs/heads/branch-05").Contains(branch05OID).
					Contains("origin -> refs/heads/branch-05").
					Contains("refs/heads/branch-06").Contains(branch06OID).
					Contains("other-remote -> refs/heads/branch-06").
					Contains("Cached commit IDs may differ from the server's current tips").
					Contains("not preserved by the remaining reviewed HEAD, upstream or main refs"))
				confirmation.Cancel()
			}).

			// Delete both local and remote branches of branch-01 and branch-02. We get
			// the force-delete warning because branch-01 it is not fully merged.
			NavigateToLine(Contains("branch-01")).
			Press(keys.Universal.RangeSelectDown).
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete selected branches?")).
					Select(MatchesRegexp(`^[[:space:]]*b Delete local and remote branches`)).
					Confirm()
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Delete selected branches?")).
					Content(Contains("refs/heads/branch-01").Contains("origin -> refs/heads/branch-01").Contains("refs/heads/branch-02").Contains("origin -> refs/heads/branch-02").Contains("Cached commit IDs may differ from the server's current tips").Contains("not preserved by the remaining reviewed HEAD, upstream or main refs")).
					Confirm()
			}).
			EventuallyContent(Contains("branch-05 (upstream gone)")).Lines(
			Contains("current-head"),
			Contains("branch-05 (upstream gone)").IsSelected(),
			Contains("branch-06 (upstream gone)"),
			Contains("master"),
		).
			Tap(func() {
				checkRemoteBranches(t, keys, "origin", []string{
					"branch-06",
				})
				checkRemoteBranches(t, keys, "other-remote", []string{
					"branch-04",
				})
			})
	},
})
