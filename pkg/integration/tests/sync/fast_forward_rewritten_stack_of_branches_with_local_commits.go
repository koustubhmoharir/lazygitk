package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardRewrittenStackOfBranchesWithLocalCommits = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse every diverged branch in a selected stack with local commits and move none of them",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createStackRewrittenOnTheRemote(shell)
		shell.Checkout("branch2")
		shell.EmptyCommit("mine")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		branch3OID := t.Git().GetCommitHash("refs/heads/branch3")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/fast-forward-rewritten-stack-local-status && cp .git/index .git/fast-forward-rewritten-stack-local-index && cp .git/config .git/fast-forward-rewritten-stack-local-config && git for-each-ref refs/heads refs/tags refs/stash refs/remotes/origin/master --format='%(refname) %(objectname)' > .git/fast-forward-rewritten-stack-local-refs && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/fast-forward-rewritten-stack-local-worktree")

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("branch1 ↓1↑1"),
				Contains("branch2 ↓2↑3"),
				Contains("branch3 ↓3↑3"),
			).
			SelectNextItem().
			Press(keys.Universal.ToggleRangeSelect).
			SelectNextItem().
			SelectNextItem().
			Press(keys.Branches.FastForward)

		t.ExpectPopup().Alert().Title(Equals("Not fast-forwarded")).Content(Contains("branch1: ").Contains("branch2: ").Contains("branch3: ").Contains("has diverged")).Confirm()

		// Every selected branch has diverged, so none of the refs moves.
		t.Views().Branches().Lines(
			Contains("master"),
			Contains("branch1 ↓1↑1").IsSelected(),
			Contains("branch2 ↓2↑3").IsSelected(),
			Contains("branch3 ↓3↑3").IsSelected(),
		)
		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID || t.Git().GetCommitHash("refs/heads/branch2") != branch2OID || t.Git().GetCommitHash("refs/heads/branch3") != branch3OID || t.Git().GetCommitHash("refs/heads/master") != masterOID || t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Refusing diverged stack members moved a branch or HEAD")
		}
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/fast-forward-rewritten-stack-local-status-after && cmp .git/fast-forward-rewritten-stack-local-status .git/fast-forward-rewritten-stack-local-status-after && cmp .git/index .git/fast-forward-rewritten-stack-local-index && cmp .git/config .git/fast-forward-rewritten-stack-local-config && git for-each-ref refs/heads refs/tags refs/stash refs/remotes/origin/master --format='%(refname) %(objectname)' > .git/fast-forward-rewritten-stack-local-refs-after && cmp .git/fast-forward-rewritten-stack-local-refs .git/fast-forward-rewritten-stack-local-refs-after && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/fast-forward-rewritten-stack-local-worktree-after && cmp .git/fast-forward-rewritten-stack-local-worktree .git/fast-forward-rewritten-stack-local-worktree-after")
	},
})
