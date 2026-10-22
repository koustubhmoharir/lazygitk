package git_commands

import (
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func currentPullFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root, _, _ := transactionHistory(t)
	gitTest(t, root, "checkout", "-q", "master")
	local := gitTest(t, root, "rev-parse", "HEAD")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gitTest(t, filepath.Dir(bare), "init", "--bare", "-q", bare)
	gitTest(t, root, "push", "-q", bare, "refs/heads/master:refs/heads/master")
	gitTest(t, root, "remote", "add", "origin", bare)
	gitTest(t, root, "config", "core.editor", "true")
	return root, bare, local
}

func currentPullOptions() BranchFetchOptions {
	return BranchFetchOptions{Remote: "origin", RemoteRef: "refs/heads/master", TrackingRef: "refs/remotes/custom/master"}
}

func advanceCurrentPullRemote(t *testing.T, bare, filename, contents string) string {
	t.Helper()
	server := filepath.Join(t.TempDir(), "server")
	gitTest(t, filepath.Dir(server), "clone", "-q", bare, server)
	gitTest(t, server, "config", "user.name", "Test")
	gitTest(t, server, "config", "user.email", "test@example.com")
	writeTestFile(t, filepath.Join(server, filename), []byte(contents), 0o644)
	gitTest(t, server, "add", filename)
	gitTest(t, server, "commit", "-qm", "remote advance")
	gitTest(t, server, "push", "-q", "origin", "HEAD:refs/heads/master")
	return gitTest(t, server, "rev-parse", "HEAD")
}

func prepareCurrentPull(t *testing.T, root, oid string) (*SyncCommands, CurrentBranchPullPlan) {
	t.Helper()
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareCurrentBranchPull("master", oid, currentPullOptions())
	if err != nil {
		t.Fatal(err)
	}
	return commands, plan
}

func executeCurrentPull(t *testing.T, commands *SyncCommands, plan CurrentBranchPullPlan) ([]byte, error) {
	t.Helper()
	cmd, err := commands.CurrentBranchPullCommand(plan, nil)
	if err != nil {
		return nil, err
	}
	return cmd.GetCmd().CombinedOutput()
}

func TestCurrentBranchPullNativeMergeFastForwardAndScope(t *testing.T) {
	root, bare, local := currentPullFixture(t)
	remoteTip := advanceCurrentPullRemote(t, bare, "incoming", "incoming\n")
	gitTest(t, root, "branch", "other-local", local)
	gitTest(t, root, "config", "branch.master.remote", "origin")
	gitTest(t, root, "config", "branch.master.merge", "refs/heads/master")
	gitTest(t, root, "config", "fetch.prune", "true")
	gitTest(t, root, "config", "fetch.pruneTags", "true")
	gitTest(t, root, "config", "remote.origin.pruneTags", "true")
	gitTest(t, root, "config", "remote.origin.tagOpt", "--tags")
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/remotes/extra/*")
	gitTest(t, root, "update-ref", "refs/remotes/extra/master", local)
	gitTest(t, root, "tag", "preserve", local)
	gitTest(t, bare, "tag", "remote-only", remoteTip)
	gitTest(t, root, "config", "pull.rebase", "false")
	gitTest(t, root, "config", "pull.ff", "only")
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareCurrentBranchPull("master", local, currentPullOptions())
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := commands.CurrentBranchPullCommand(plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "-c", "remote.origin.pruneTags=false", "-c", "fetch.pruneTags=false", "-c", "merge.autostash=false", "-c", "rebase.updateRefs=false", "pull", "--no-edit", "--no-autostash", "--refmap=", "--no-tags", "--no-prune", "--no-recurse-submodules", "--", "origin", "+refs/heads/master:refs/remotes/custom/master"}, cmd.Args())
	assert.True(t, cmd.LockRetriesDisabled())
	assert.Equal(t, oscommands.PROMPT, cmd.GetCredentialStrategy())
	assert.Contains(t, cmd.GetEnvVars(), "GIT_SEQUENCE_EDITOR=:")
	output, err := cmd.GetCmd().CombinedOutput()
	if err != nil {
		t.Fatalf("pull: %v\n%s", err, output)
	}
	assert.Equal(t, remoteTip, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, local, gitTest(t, root, "rev-parse", "refs/heads/other-local"))
	assert.Equal(t, local, gitTest(t, root, "rev-parse", "refs/remotes/extra/master"))
	assert.Equal(t, local, gitTest(t, root, "rev-parse", "refs/tags/preserve"))
	assert.Empty(t, gitTest(t, root, "for-each-ref", "--format=%(refname)", "refs/tags/remote-only"))
}

func TestCurrentBranchPullRebasePolicyCleanlinessAndNativePrecedence(t *testing.T) {
	root, bare, _ := currentPullFixture(t)
	gitTest(t, root, "config", "pull.rebase", "true")
	gitTest(t, root, "config", "rebase.updateRefs", "true")
	remoteTip := advanceCurrentPullRemote(t, bare, "remote", "remote\n")
	gitTest(t, root, "config", "branch.master.remote", "origin")
	gitTest(t, root, "config", "branch.master.merge", "refs/heads/master")
	localTip := branchPlacementTestCommit(t, root, "local-only", "local\n", "local-only commit")
	writeTestFile(t, filepath.Join(root, "harmless-untracked"), []byte("leave me\n"), 0o600)
	gitTest(t, root, "branch", "other-local", localTip)
	commands, plan := prepareCurrentPull(t, root, localTip)
	assert.True(t, plan.UsesRebase())
	output, err := executeCurrentPull(t, commands, plan)
	if err != nil {
		t.Fatalf("pull: %v\n%s", err, output)
	}
	assert.Equal(t, localTip, gitTest(t, root, "rev-parse", "refs/heads/other-local"))
	assert.NotEqual(t, localTip, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Empty(t, gitTest(t, root, "merge-base", "--is-ancestor", remoteTip, "HEAD"))
	assert.Equal(t, "local\n", string(mustRead(t, filepath.Join(root, "local-only"))))
	assert.Equal(t, "leave me\n", string(mustRead(t, filepath.Join(root, "harmless-untracked"))))

	// Tracked dirt is refused before native pull can contact/update the remote.
	gitTest(t, root, "config", "pull.ff", "only")
	gitTest(t, root, "config", "pull.rebase", "true")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
	_, err = commands.PrepareCurrentBranchPull("master", gitTest(t, root, "rev-parse", "HEAD"), currentPullOptions())
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
}

func TestCurrentBranchPullMergeAllowsUnrelatedDirtWithoutAutostash(t *testing.T) {
	root, bare, local := currentPullFixture(t)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("stash me\n"), 0o644)
	gitTest(t, root, "stash", "push", "-qm", "preserved pre-existing stash")
	stashRef := gitTest(t, root, "rev-parse", "-q", "--verify", "refs/stash")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("harmless dirty edit\n"), 0o644)
	advanceCurrentPullRemote(t, bare, "incoming-unrelated", "incoming\n")
	gitTest(t, root, "config", "branch.master.remote", "origin")
	gitTest(t, root, "config", "branch.master.merge", "refs/heads/master")
	gitTest(t, root, "config", "pull.rebase", "false")
	gitTest(t, root, "config", "merge.autostash", "true")
	commands, plan := prepareCurrentPull(t, root, local)
	output, err := executeCurrentPull(t, commands, plan)
	if err != nil {
		t.Fatalf("merge pull with unrelated dirt: %v\n%s", err, output)
	}
	assert.Equal(t, "harmless dirty edit\n", string(mustRead(t, filepath.Join(root, "tracked"))))
	assert.Equal(t, stashRef, gitTest(t, root, "rev-parse", "-q", "--verify", "refs/stash"))
	assert.FileExists(t, filepath.Join(root, "incoming-unrelated"))
}

func TestCurrentBranchPullRebaseRefusesTrackedDirtBeforeFetch(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unstaged", true: "staged"}[staged], func(t *testing.T) {
			root, bare, local := currentPullFixture(t)
			advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
			gitTest(t, root, "config", "pull.rebase", "true")
			gitTest(t, root, "config", "branch.master.remote", "origin")
			gitTest(t, root, "config", "branch.master.merge", "refs/heads/master")
			gitTest(t, root, "update-ref", "refs/remotes/custom/master", local)
			writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
			if staged {
				gitTest(t, root, "add", "tracked")
				writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged plus unstaged\n"), 0o644)
			}
			commands := branchSyncTestCommands(t, root)
			_, err := commands.PrepareCurrentBranchPull("master", local, currentPullOptions())
			assert.ErrorIs(t, err, ErrRebaseRequiresClean)
			assert.Equal(t, local, gitTest(t, root, "rev-parse", "refs/remotes/custom/master"))
			assert.Equal(t, local, gitTest(t, root, "rev-parse", "HEAD"))
		})
	}
}

func TestCurrentBranchPullRebaseRefusesDirtAddedAfterPlan(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unstaged", true: "staged"}[staged], func(t *testing.T) {
			root, bare, local := currentPullFixture(t)
			advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
			gitTest(t, root, "config", "pull.rebase", "true")
			gitTest(t, root, "config", "branch.master.remote", "origin")
			gitTest(t, root, "config", "branch.master.merge", "refs/heads/master")
			gitTest(t, root, "update-ref", "refs/remotes/custom/master", local)
			commands, plan := prepareCurrentPull(t, root, local)
			writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty after prepare\n"), 0o644)
			if staged {
				gitTest(t, root, "add", "tracked")
			}
			_, err := commands.CurrentBranchPullCommand(plan, nil)
			assert.ErrorIs(t, err, ErrRebaseRequiresClean)
			assert.Equal(t, local, gitTest(t, root, "rev-parse", "refs/remotes/custom/master"))
			assert.Equal(t, local, gitTest(t, root, "rev-parse", "HEAD"))
		})
	}
}

func TestCurrentBranchPullFFOnlyWinsOverConfiguredRebase(t *testing.T) {
	root, bare, local := currentPullFixture(t)
	gitTest(t, root, "config", "pull.rebase", "true")
	gitTest(t, root, "config", "pull.ff", "only")
	gitTest(t, root, "config", "branch.master.remote", "origin")
	gitTest(t, root, "config", "branch.master.merge", "refs/heads/master")
	gitTest(t, root, "update-ref", "refs/remotes/custom/master", local)
	branchPlacementTestCommit(t, root, "local-only", "local\n", "local divergence")
	advanceCurrentPullRemote(t, bare, "remote-only", "remote\n")
	localTip := gitTest(t, root, "rev-parse", "HEAD")
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareCurrentBranchPull("master", localTip, currentPullOptions())
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, plan.UsesRebase())
	output, err := executeCurrentPull(t, commands, plan)
	assert.Error(t, err, "native pull.ff=only must reject divergence instead of forcibly rebasing")
	assert.NotContains(t, string(output), "Successfully rebased")
	assert.Equal(t, localTip, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestCurrentBranchPullRejectsStaleSourceRemoteAndPolicy(t *testing.T) {
	root, bare, local := currentPullFixture(t)
	commands, plan := prepareCurrentPull(t, root, local)
	gitTest(t, root, "config", "branch.Master.rebase", "true")
	_, err := commands.CurrentBranchPullCommand(plan, nil)
	assert.NoError(t, err, "branch subsections are case-sensitive")
	gitTest(t, root, "config", "branch.master.mergeOptions", "--no-ff")
	_, err = commands.CurrentBranchPullCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	plan, err = commands.PrepareCurrentBranchPull("master", local, currentPullOptions())
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "config", "pull.ff", "only")
	_, err = commands.CurrentBranchPullCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	plan, err = commands.PrepareCurrentBranchPull("master", local, currentPullOptions())
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/changed/*")
	_, err = commands.CurrentBranchPullCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	plan, err = commands.PrepareCurrentBranchPull("master", local, currentPullOptions())
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "config", "branch.master.rebase", "true")
	_, err = commands.CurrentBranchPullCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	plan, err = commands.PrepareCurrentBranchPull("master", local, currentPullOptions())
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "remote", "set-url", "origin", bare+"-changed")
	_, err = commands.CurrentBranchPullCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
}

func TestCurrentBranchPullRejectsChangedSourceHeadAndBackend(t *testing.T) {
	root, _, local := currentPullFixture(t)
	commands, plan := prepareCurrentPull(t, root, local)
	newTip := branchPlacementTestCommit(t, root, "source-change", "changed\n", "source changed")
	_, err := commands.CurrentBranchPullCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	assert.Equal(t, newTip, gitTest(t, root, "rev-parse", "HEAD"))

	root2, _, local2 := currentPullFixture(t)
	_, foreignPlan := prepareCurrentPull(t, root2, local2)
	_, err = commands.CurrentBranchPullCommand(foreignPlan, nil)
	assert.Error(t, err, "a plan is bound to its original backend instance")

	root3, _, local3 := currentPullFixture(t)
	commands3, headPlan := prepareCurrentPull(t, root3, local3)
	gitTest(t, root3, "branch", "elsewhere")
	gitTest(t, root3, "checkout", "-q", "elsewhere")
	_, err = commands3.CurrentBranchPullCommand(headPlan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
}
