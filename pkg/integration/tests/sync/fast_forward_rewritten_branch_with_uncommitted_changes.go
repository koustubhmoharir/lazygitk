package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardRewrittenBranchWithUncommittedChanges = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to fast-forward a diverged checked out branch and preserve uncommitted changes",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "content")
		createBranchRewrittenOnTheRemote(shell)

		shell.Checkout("feature")
		shell.UpdateFile("file", "changed")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		localOID := t.Git().GetCommitHash("refs/heads/feature")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-dirty-status && cp .git/index .git/ff-dirty-index && cp .git/config .git/ff-dirty-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-dirty-refs && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-dirty-worktree")
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
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-dirty-status-after && cmp .git/ff-dirty-status .git/ff-dirty-status-after && cmp .git/index .git/ff-dirty-index && cmp .git/config .git/ff-dirty-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-dirty-refs-after && cmp .git/ff-dirty-refs .git/ff-dirty-refs-after && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-dirty-worktree-after && cmp .git/ff-dirty-worktree .git/ff-dirty-worktree-after")

		t.Views().Branches().
			Lines(
				Contains("feature ↓2↑2").IsSelected(),
				Contains("master"),
			)

		t.Views().Files().
			Lines(
				Contains("file"),
			)
		t.FileSystem().FileContent("file", Equals("changed"))
	},
})
