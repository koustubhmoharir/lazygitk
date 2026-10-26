package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardRewrittenBranchCheckedOut = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to fast-forward the checked out branch when its upstream was rewritten",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		createBranchRewrittenOnTheRemote(shell)

		shell.Checkout("feature")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		localOID := t.Git().GetCommitHash("refs/heads/feature")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-current-status && cp .git/index .git/ff-current-index && cp .git/config .git/ff-current-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-current-refs && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-current-worktree")
		t.Views().Commits().
			Lines(
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)

		t.Views().Branches().
			Focus().
			Lines(
				Contains("feature ↓2↑2").IsSelected(),
				Contains("master"),
			).
			Press(keys.Branches.FastForward)

		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).Content(Contains("feature: ").Contains("has diverged")).Confirm()

		if t.Git().GetCommitHash("refs/heads/feature") != localOID || t.Git().GetCommitHash("HEAD") != headOID || t.Git().GetCommitHash("refs/heads/master") != masterOID {
			t.Fail("Refusing divergent current-branch fast-forward changed branch refs or HEAD")
		}
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-current-status-after && cmp .git/ff-current-status .git/ff-current-status-after && cmp .git/index .git/ff-current-index && cmp .git/config .git/ff-current-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-current-refs-after && cmp .git/ff-current-refs .git/ff-current-refs-after && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-current-worktree-after && cmp .git/ff-current-worktree .git/ff-current-worktree-after")

		t.Views().Commits().Lines(Contains("three"), Contains("two"), Contains("one"))
	},
})
