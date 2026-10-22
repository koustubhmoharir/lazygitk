package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardRewrittenBranchWithoutReflogs = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse rewritten upstream history without reflogs after explicitly reviewing the fetch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		// Keep this rewritten-history fixture independent of reflog evidence.
		shell.SetConfig("core.logAllRefUpdates", "false")

		createBranchRewrittenOnTheRemote(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		localOID := t.Git().GetCommitHash("refs/heads/feature")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-no-reflogs-status && cp .git/index .git/ff-no-reflogs-index && cp .git/config .git/ff-no-reflogs-config && git for-each-ref refs/heads refs/tags refs/stash refs/remotes/origin/master --format='%(refname) %(objectname)' > .git/ff-no-reflogs-refs && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-no-reflogs-worktree")
		t.Views().Branches().
			Focus().
			Lines(Contains("master").IsSelected(), Contains("feature ↓2↑2")).
			SelectNextItem().
			Press(keys.Branches.FastForward)
		options := t.ExpectPopup().Menu().Title(Equals("Review selected branch fast-forward"))
		t.Views().Menu().Content(Contains("Local branch \"feature\""))
		t.Views().Menu().Content(Contains("Fast-forward only"))
		t.Views().Menu().Content(Contains("Fetch upstream for review"))
		options.Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("feature", "origin", "feature")
		diverged := t.ExpectPopup().Menu().Title(Equals("Review branch fast-forward"))
		t.Views().Menu().Content(Contains("local branch \"feature\" has diverged"))
		t.Views().Menu().Content(MatchesRegexp(`No\s+local\s+branch\s+was\s+moved`))
		t.Views().Menu().Content(Contains("will not reset"))
		t.Views().Menu().Content(Contains("Cancel"))
		diverged.Cancel()

		t.Views().Branches().
			Lines(
				Contains("master"),
				Contains("feature ↓2↑2").IsSelected(),
			)
		if t.Git().GetCommitHash("refs/heads/feature") != localOID || t.Git().GetCommitHash("HEAD") != headOID || t.Git().GetCommitHash("refs/heads/master") != masterOID {
			t.Fail("Refusing rewritten history moved the selected branch, master or HEAD")
		}
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-no-reflogs-status-after && cmp .git/ff-no-reflogs-status .git/ff-no-reflogs-status-after && cmp .git/index .git/ff-no-reflogs-index && cmp .git/config .git/ff-no-reflogs-config && git for-each-ref refs/heads refs/tags refs/stash refs/remotes/origin/master --format='%(refname) %(objectname)' > .git/ff-no-reflogs-refs-after && cmp .git/ff-no-reflogs-refs .git/ff-no-reflogs-refs-after && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-no-reflogs-worktree-after && cmp .git/ff-no-reflogs-worktree .git/ff-no-reflogs-worktree-after")
	},
})
