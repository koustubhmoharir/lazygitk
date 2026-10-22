package git_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func pullPolicyRequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func initPullPolicyRepo(t *testing.T) string {
	t.Helper()
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("clean\n"), 0o600)
	gitTest(t, root, "add", "tracked")
	gitTest(t, root, "commit", "-qm", "initial")
	return root
}

func TestPullRebasePolicyRealGitConfigPrecedence(t *testing.T) {
	root := initPullPolicyRepo(t)
	osCommand := oscommands.NewDummyOSCommand()
	commands := NewSyncCommands(NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil))

	// Use an isolated global config so this test is independent of the host.
	globalConfig := filepath.Join(t.TempDir(), "global-config")
	pullPolicyRequireNoError(t, os.WriteFile(globalConfig, []byte("[pull]\n\trebase = true\n"), 0o600))
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
	branchKey := "branch." + branch + ".rebase"
	gitTest(t, root, "config", branchKey, "false")
	policy, err := commands.effectivePullRebasePolicy(PullOptions{})
	pullPolicyRequireNoError(t, err)
	assert.False(t, policy.rebase, "branch false must override global pull true")

	for _, mode := range []string{"true", "merges", "m", "interactive", "i", "preserve", "p"} {
		gitTest(t, root, "config", branchKey, mode)
		policy, err = commands.effectivePullRebasePolicy(PullOptions{})
		pullPolicyRequireNoError(t, err)
		assert.True(t, policy.rebase, "mode %q", mode)
	}
	gitTest(t, root, "config", "--unset", branchKey)
	policy, err = commands.effectivePullRebasePolicy(PullOptions{})
	pullPolicyRequireNoError(t, err)
	assert.True(t, policy.rebase, "missing branch key falls back to pull.rebase")

	gitTest(t, root, "checkout", "--detach")
	policy, err = commands.effectivePullRebasePolicy(PullOptions{})
	pullPolicyRequireNoError(t, err)
	assert.True(t, policy.rebase, "detached HEAD uses pull.rebase")
}

func TestPullRebasePolicyRejectsDirtyBeforeFetchAndIgnoresMergeDirt(t *testing.T) {
	root := initPullPolicyRepo(t)
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, root, "init", "--bare", bare)
	gitTest(t, root, "remote", "add", "origin", bare)
	gitTest(t, root, "push", "-u", "origin", "HEAD")
	branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
	osCommand := oscommands.NewDummyOSCommand()
	commands := NewSyncCommands(NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil))
	gitTest(t, root, "config", "pull.rebase", "true")
	gitTest(t, root, "config", "rebase.autoStash", "true")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o600)
	err := commands.Pull(nil, PullOptions{RemoteName: "origin", BranchName: branch})
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	contents, readErr := os.ReadFile(filepath.Join(root, "tracked"))
	pullPolicyRequireNoError(t, readErr)
	assert.Equal(t, "dirty\n", string(contents))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", "rebase.autoStash"))
	assert.Empty(t, gitTest(t, root, "stash", "list"))
	_, fetchErr := os.Stat(filepath.Join(root, ".git", "FETCH_HEAD"))
	assert.True(t, os.IsNotExist(fetchErr), "dirty rejection must happen before fetching")

	gitTest(t, root, "checkout", "--", "tracked")
	writeTestFile(t, filepath.Join(root, "untracked"), []byte("keep me\n"), 0o600)
	assert.NoError(t, commands.Pull(nil, PullOptions{RemoteName: "origin", BranchName: branch}),
		"untracked files are harmless for pull's rebase start check")
	assert.FileExists(t, filepath.Join(root, "untracked"))
	gitTest(t, root, "config", "pull.rebase", "false")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("merge dirt\n"), 0o600)
	// A divergent remote isn't needed to verify that the configured merge path
	// does not acquire the rebase clean-tree policy.
	assert.NoError(t, commands.Pull(nil, PullOptions{RemoteName: "origin", BranchName: branch}))
}

func TestPullRebasePolicyUnknownAndQueryFailure(t *testing.T) {
	root := initPullPolicyRepo(t)
	branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
	branchKey := "branch." + branch + ".rebase"
	gitTest(t, root, "config", branchKey, "nonsense")
	osCommand := oscommands.NewDummyOSCommand()
	commands := NewSyncCommands(NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil))
	_, err := commands.effectivePullRebasePolicy(PullOptions{})
	assert.ErrorContains(t, err, "unsupported pull rebase mode")

	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"symbolic-ref", "--quiet", "--short", "HEAD"}, "main\n", nil).
		ExpectGitArgs([]string{"config", "--type=bool-or-str", "--get", "branch.main.rebase"}, "", assert.AnError)
	fakeCommands := buildSyncCommands(commonDeps{runner: runner})
	_, err = fakeCommands.effectivePullRebasePolicy(PullOptions{})
	assert.ErrorIs(t, err, assert.AnError)
	runner.CheckForMissingCalls()
}

func TestPullRebasePolicyLinkedWorktreeContext(t *testing.T) {
	root := initPullPolicyRepo(t)
	branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, root, "clone", "--bare", root, remote)
	gitTest(t, root, "remote", "add", "origin", remote)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-b", "linked-policy", linked)
	osCommand := oscommands.NewDummyOSCommand()
	commands := NewSyncCommands(NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil))
	gitTest(t, linked, "config", "branch.linked-policy.rebase", "true")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty root\n"), 0o600)
	policy, err := commands.effectivePullRebasePolicy(PullOptions{WorktreeGitDir: gitTest(t, linked, "rev-parse", "--absolute-git-dir"), WorktreePath: linked})
	pullPolicyRequireNoError(t, err)
	assert.True(t, policy.rebase)
	// A clean target worktree must not inherit the open worktree's dirt.
	pullPolicyRequireNoError(t, commands.Pull(nil, PullOptions{
		RemoteName: "origin", BranchName: branch,
		WorktreeGitDir: gitTest(t, linked, "rev-parse", "--absolute-git-dir"), WorktreePath: linked,
	}))
	writeTestFile(t, filepath.Join(linked, "tracked"), []byte("dirty linked\n"), 0o600)
	err = commands.Pull(nil, PullOptions{RemoteName: "origin", BranchName: branch, WorktreeGitDir: gitTest(t, linked, "rev-parse", "--absolute-git-dir"), WorktreePath: linked})
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	contents, readErr := os.ReadFile(filepath.Join(root, "tracked"))
	pullPolicyRequireNoError(t, readErr)
	assert.Equal(t, "dirty root\n", string(contents))
}

func TestPullRebasePolicyReplaysDivergentHistory(t *testing.T) {
	root := initPullPolicyRepo(t)
	branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, root, "clone", "--bare", root, remote)
	gitTest(t, root, "remote", "add", "origin", remote)
	gitTest(t, root, "push", "-u", "origin", "HEAD")
	other := filepath.Join(t.TempDir(), "other")
	gitTest(t, root, "clone", remote, other)
	gitTest(t, other, "config", "user.name", "Test")
	gitTest(t, other, "config", "user.email", "test@example.invalid")
	target := selectedCommitRebaseCommit(t, other, "remote-file", "remote\n", "remote")
	gitTest(t, other, "push", "origin", "HEAD")
	oldHead := selectedCommitRebaseCommit(t, root, "local-file", "local\n", "local")
	gitTest(t, root, "config", "pull.rebase", "true")
	gitTest(t, root, "config", "rebase.autoStash", "true")
	gitTest(t, root, "config", "pull.autoStash", "true")
	osCommand := oscommands.NewDummyOSCommand()
	commands := NewSyncCommands(NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil))
	pullPolicyRequireNoError(t, commands.Pull(nil, PullOptions{RemoteName: "origin", BranchName: branch}))
	assert.NotEqual(t, oldHead, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, target, gitTest(t, root, "merge-base", "HEAD", target))
	assert.FileExists(t, filepath.Join(root, "remote-file"))
	assert.FileExists(t, filepath.Join(root, "local-file"))
	assert.Empty(t, gitTest(t, root, "status", "--porcelain"))
	assert.Empty(t, gitTest(t, root, "stash", "list"))
	assert.Equal(t, "true", gitTest(t, root, "config", "rebase.autoStash"))
	assert.Equal(t, "true", gitTest(t, root, "config", "pull.autoStash"))
}
