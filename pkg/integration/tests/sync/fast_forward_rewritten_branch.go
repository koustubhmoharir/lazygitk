package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardRewrittenBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse to fast-forward a diverged branch that is not checked out and leave every ref unchanged",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		createBranchRewrittenOnTheRemote(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		localOID := t.Git().GetCommitHash("refs/heads/feature")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-status && cp .git/index .git/ff-index && cp .git/config .git/ff-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-refs && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-worktree")

		t.Views().Branches().
			Focus().
			Lines(Contains("master").IsSelected(), Contains("feature ↓2↑2")).
			SelectNextItem().
			Press(keys.Branches.FastForward)
		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).
			Content(Contains("feature: ").Contains("has diverged")).
			Confirm()

		if got := t.Git().GetCommitHash("refs/heads/feature"); got != localOID {
			t.Fail("Refusing rewritten history moved the selected branch")
		}
		if got := t.Git().GetCommitHash("refs/heads/master"); got != masterOID || t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Refusing rewritten history changed master or HEAD")
		}
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/ff-status-after && cmp .git/ff-status .git/ff-status-after && cmp .git/index .git/ff-index && cmp .git/config .git/ff-config && git for-each-ref refs/heads refs/tags refs/stash --format='%(refname) %(objectname)' > .git/ff-refs-after && cmp .git/ff-refs .git/ff-refs-after && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/ff-worktree-after && cmp .git/ff-worktree .git/ff-worktree-after")
		t.Views().Branches().Lines(Contains("master"), Contains("feature ↓2↑2").IsSelected())
	},
})
