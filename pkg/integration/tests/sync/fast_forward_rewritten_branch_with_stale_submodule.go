package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardRewrittenBranchWithStaleSubmodule = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to fast-forward a diverged checked out branch while preserving a submodule checked out at a different commit than recorded",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial")
		shell.CloneIntoSubmodule("sub", "sub")
		createBranchRewrittenOnTheRemote(shell)

		shell.Checkout("feature")
		shell.RunCommand([]string{"git", "-C", "sub", "commit", "--allow-empty", "-m", "newer"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		localOID := t.Git().GetCommitHash("refs/heads/feature")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-submodule-status && cp .git/index .git/ff-submodule-index && cp .git/config .git/ff-submodule-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-submodule-refs && git ls-files | while IFS= read -r f; do if [ \"$f\" != sub ]; then printf '%s ' \"$f\"; git hash-object -- \"$f\"; fi; done > .git/ff-submodule-worktree")
		t.Shell().RunShellCommand("git -C sub rev-parse HEAD > .git/fast-forward-rewritten-submodule-tip")
		t.Views().Files().
			Lines(
				Contains("sub"),
			)

		t.Views().Branches().
			Focus().
			Lines(
				Contains("feature ↓2↑2").IsSelected(),
				Contains("master"),
			).
			Press(keys.Branches.FastForward)

		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).Content(Contains("feature: ").Contains("has diverged")).Confirm()
		if t.Git().GetCommitHash("refs/heads/feature") != localOID || t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Refusing divergent current-branch fast-forward changed the branch or HEAD")
		}
		t.Shell().RunShellCommand("git -C sub rev-parse HEAD > .git/fast-forward-rewritten-submodule-tip-after && cmp .git/fast-forward-rewritten-submodule-tip .git/fast-forward-rewritten-submodule-tip-after")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-submodule-status-after && cmp .git/ff-submodule-status .git/ff-submodule-status-after && cmp .git/index .git/ff-submodule-index && cmp .git/config .git/ff-submodule-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-submodule-refs-after && cmp .git/ff-submodule-refs .git/ff-submodule-refs-after && git ls-files | while IFS= read -r f; do if [ \"$f\" != sub ]; then printf '%s ' \"$f\"; git hash-object -- \"$f\"; fi; done > .git/ff-submodule-worktree-after && cmp .git/ff-submodule-worktree .git/ff-submodule-worktree-after")

		// The submodule was left alone
		t.Views().Files().
			Lines(
				Contains("sub"),
			)
	},
})
