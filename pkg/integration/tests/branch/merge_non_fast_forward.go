package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// Fork: the Merge menu offers the configured merge and the explicit
// fast-forward and no-fast-forward choices, and runs the chosen one directly.
// The assertions check the parents of each merge commit rather than the graph
// glyphs, so a wrong parent or a silent fast-forward fails the test.
var MergeNonFastForward = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Merge selected branches with configured and no-fast-forward modes",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		shell.NewBranch("original-branch").
			EmptyCommit("one").
			NewBranch("branch1").
			EmptyCommit("branch1").
			Checkout("original-branch").
			NewBranchFrom("branch2", "original-branch").
			EmptyCommit("branch2").
			Checkout("original-branch")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		oneOID := t.Git().GetCommitHash("refs/heads/original-branch")
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")

		t.Views().Branches().Focus().
			Lines(Contains("original-branch").IsSelected(), Contains("branch1"), Contains("branch2")).
			NavigateToLine(Contains("branch2")).Press(keys.Branches.MergeIntoCurrentBranch)

		t.ExpectPopup().Menu().Title(Equals("Merge")).
			ContainsLines(
				Contains("Merge using configured options"),
				Contains("Merge (allow fast-forward)"),
				Contains("Merge (no fast-forward)"),
				Contains("Squash merge and leave uncommitted"),
				Contains("Squash merge and commit"),
			).
			Select(MatchesRegexp(`^[[:space:]]*n → Merge \(no fast-forward\)`)).Confirm()

		t.Views().Commits().EventuallyContent(Contains(`Merge branch "branch2" into "original-branch"`)).ContainsLines(
			Contains(`Merge branch "branch2" into "original-branch"`).IsSelected(),
			Contains("branch2"),
		)
		// branch2 was based on the current tip, so only the no-fast-forward choice
		// makes a merge commit: HEAD has the old tip and branch2 as its two parents.
		assertMergeCommitParents(t, oneOID, branch2OID)
		mergeBranch2OID := t.Git().GetCommitHash("HEAD")

		t.Views().Branches().Focus().NavigateToLine(Contains("branch1")).Press(keys.Branches.MergeIntoCurrentBranch)
		// m runs the configured ordinary merge straight from the menu: the merge
		// commit appears without a second confirmation.
		t.ExpectPopup().Menu().Title(Equals("Merge"))
		t.Views().Menu().Press(config.Keybinding{"m"})

		t.Views().Commits().EventuallyContent(Contains(`Merge branch "branch1" into "original-branch"`)).ContainsLines(
			Contains(`Merge branch "branch1" into "original-branch"`).IsSelected(),
			Contains("branch1"),
			Contains(`Merge branch "branch2" into "original-branch"`),
		)
		// original-branch has moved past branch1's base, so the configured merge
		// cannot fast-forward either; HEAD merges branch1 into the first merge commit.
		assertMergeCommitParents(t, mergeBranch2OID, branch1OID)
		// The whole graph, as subjects in order. Glyphs are not asserted.
		t.Views().Commits().Lines(
			Contains(`Merge branch "branch1" into "original-branch"`).IsSelected(),
			Contains("branch1"),
			Contains(`Merge branch "branch2" into "original-branch"`),
			Contains("branch2"),
			Contains("one"),
		)
		t.Views().Confirmation().IsInvisible()
	},
})

// assertMergeCommitParents checks that HEAD is a merge commit with exactly two
// parents: firstParent, then secondParent. A fast-forward has no merge commit,
// so it fails the HEAD^2 check.
func assertMergeCommitParents(t *TestDriver, firstParent, secondParent string) {
	t.Shell().RunCommand([]string{"git", "rev-parse", "--verify", "--quiet", "HEAD^2"})
	t.Shell().RunCommandExpectError([]string{"git", "rev-parse", "--verify", "--quiet", "HEAD^3"})
	if got := t.Git().GetCommitHash("HEAD^1"); got != firstParent {
		t.Fail("Expected the first parent of HEAD to be " + firstParent + ", but got " + got)
	}
	if got := t.Git().GetCommitHash("HEAD^2"); got != secondParent {
		t.Fail("Expected the second parent of HEAD to be " + secondParent + ", but got " + got)
	}
}
