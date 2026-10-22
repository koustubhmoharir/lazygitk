package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardRewrittenStackOfBranches = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refuse the first rewritten branch in a selected stack after explicitly reviewing its fetch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.LocalBranchSortOrder = "alphabetical"
	},
	SetupRepo: func(shell *Shell) {
		createStackRewrittenOnTheRemote(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		branch1OID := t.Git().GetCommitHash("refs/heads/branch1")
		branch2OID := t.Git().GetCommitHash("refs/heads/branch2")
		branch3OID := t.Git().GetCommitHash("refs/heads/branch3")
		masterOID := t.Git().GetCommitHash("refs/heads/master")
		headOID := t.Git().GetCommitHash("HEAD")
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/fast-forward-rewritten-stack-status && cp .git/index .git/fast-forward-rewritten-stack-index && cp .git/config .git/fast-forward-rewritten-stack-config && git for-each-ref refs/heads refs/tags refs/stash refs/remotes/origin/master --format='%(refname) %(objectname)' > .git/fast-forward-rewritten-stack-refs && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/fast-forward-rewritten-stack-worktree")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("branch1 ↓1↑1"),
				Contains("branch2 ↓2↑2"),
				Contains("branch3 ↓3↑3"),
			).
			SelectNextItem().
			Press(keys.Universal.ToggleRangeSelect).
			SelectNextItem().
			SelectNextItem().
			Lines(
				Contains("master"),
				Contains("branch1 ↓1↑1").IsSelected(),
				Contains("branch2 ↓2↑2").IsSelected(),
				Contains("branch3 ↓3↑3").IsSelected(),
			).
			Press(keys.Branches.FastForward)

		options := t.ExpectPopup().Menu().Title(Equals("Review selected branch fast-forward"))
		t.Views().Menu().Content(Contains("Local branch \"branch1\""))
		t.Views().Menu().Content(Contains("Fast-forward only"))
		t.Views().Menu().Content(Contains("Fetch upstream for review"))
		options.Select(Contains("Fetch upstream for review")).Confirm()
		t.ConfirmFastForwardFetch("branch1", "origin", "branch1")
		diverged := t.ExpectPopup().Menu().Title(Equals("Review branch fast-forward"))
		t.Views().Menu().Content(Contains("local branch \"branch1\" has diverged"))
		t.Views().Menu().Content(MatchesRegexp(`No\s+local\s+branch\s+was\s+moved`))
		t.Views().Menu().Content(Contains("will not reset"))
		t.Views().Menu().Content(Contains("Cancel"))
		diverged.Cancel()

		if t.Git().GetCommitHash("refs/heads/branch1") != branch1OID || t.Git().GetCommitHash("refs/heads/branch2") != branch2OID || t.Git().GetCommitHash("refs/heads/branch3") != branch3OID || t.Git().GetCommitHash("refs/heads/master") != masterOID || t.Git().GetCommitHash("HEAD") != headOID {
			t.Fail("Refusing the first divergent stack member moved a branch or HEAD")
		}
		t.Shell().RunShellCommand("GIT_OPTIONAL_LOCKS=0 git status --porcelain=v1 -z > .git/fast-forward-rewritten-stack-status-after && cmp .git/fast-forward-rewritten-stack-status .git/fast-forward-rewritten-stack-status-after && cmp .git/index .git/fast-forward-rewritten-stack-index && cmp .git/config .git/fast-forward-rewritten-stack-config && git for-each-ref refs/heads refs/tags refs/stash refs/remotes/origin/master --format='%(refname) %(objectname)' > .git/fast-forward-rewritten-stack-refs-after && cmp .git/fast-forward-rewritten-stack-refs .git/fast-forward-rewritten-stack-refs-after && git ls-files | while IFS= read -r f; do printf '%s ' \"$f\"; git hash-object -- \"$f\"; done > .git/fast-forward-rewritten-stack-worktree-after && cmp .git/fast-forward-rewritten-stack-worktree .git/fast-forward-rewritten-stack-worktree-after")
	},
})
