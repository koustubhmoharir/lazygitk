package git_commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchUpstreamSelectedBranchMetadata(t *testing.T) {
	h := branchPushTestHistory(t)
	commands := branchSyncTestCommands(t, h.root)
	gitTest(t, h.root, "config", "branch.master.remote", ".")
	gitTest(t, h.root, "config", "branch.master.merge", "refs/heads/selected")
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("staged contents\n"), 0o644)
	gitTest(t, h.root, "add", "tracked")
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("unstaged contents\n"), 0o644)
	writeTestFile(t, filepath.Join(h.root, "binary-untracked"), []byte{0, 1, 2, 255}, 0o600)
	before := branchUpstreamWorktreeSnapshot(t, h.root)
	plan, err := commands.PrepareBranchUpstream("selected", h.selected, BranchUpstreamOptions{TrackingRef: "refs/remotes/origin/selected"})
	branchPushRequireNoError(t, err)
	assert.Equal(t, "selected", plan.Name())
	assert.Equal(t, h.selected, plan.SourceOID())
	assert.Equal(t, "refs/remotes/origin/selected", plan.TrackingRef())
	assert.Equal(t, h.base, plan.TargetOID())
	assert.False(t, plan.Unset())
	assert.Empty(t, plan.PreviousRemote())
	previousMerges := plan.PreviousMergeRefs()
	assert.Empty(t, previousMerges)
	previousMerges = append(previousMerges, "mutation")
	assert.Equal(t, []string{"mutation"}, previousMerges)
	assert.Empty(t, plan.PreviousMergeRefs())
	cmd, err := commands.BranchUpstreamCommand(plan)
	branchPushRequireNoError(t, err)
	assert.Equal(t, []string{"git", "branch", "--set-upstream-to=refs/remotes/origin/selected", "--", "selected"}, cmd.Args())
	assert.True(t, cmd.LockRetriesDisabled())
	output, err := cmd.GetCmd().CombinedOutput()
	branchPushRequireNoError(t, err, string(output))
	assert.Equal(t, before, branchUpstreamWorktreeSnapshot(t, h.root))
	assert.Equal(t, "origin", gitTest(t, h.root, "config", "branch.selected.remote"))
	assert.Equal(t, "refs/heads/selected", gitTest(t, h.root, "config", "branch.selected.merge"))
	assert.Equal(t, ".", gitTest(t, h.root, "config", "branch.master.remote"))
	assert.Equal(t, "refs/heads/selected", gitTest(t, h.root, "config", "branch.master.merge"))

	unset, err := commands.PrepareBranchUpstream("selected", h.selected, BranchUpstreamOptions{Unset: true})
	branchPushRequireNoError(t, err)
	assert.True(t, unset.Unset())
	assert.Equal(t, "origin", unset.PreviousRemote())
	assert.Equal(t, []string{"refs/heads/selected"}, unset.PreviousMergeRefs())
	cmd, err = commands.BranchUpstreamCommand(unset)
	branchPushRequireNoError(t, err)
	output, err = cmd.GetCmd().CombinedOutput()
	branchPushRequireNoError(t, err, string(output))
	assert.Equal(t, before, branchUpstreamWorktreeSnapshot(t, h.root))
	assert.Empty(t, branchPushConfig(t, h.root, "branch.selected.remote"))
	assert.Empty(t, branchPushConfig(t, h.root, "branch.selected.merge"))
	assert.Equal(t, ".", gitTest(t, h.root, "config", "branch.master.remote"))
	assert.Equal(t, "refs/heads/selected", gitTest(t, h.root, "config", "branch.master.merge"))
	assert.Equal(t, h.selected, gitTest(t, h.root, "rev-parse", "refs/heads/selected"))
}

func TestBranchUpstreamRejectsStaleAndUnsafePlans(t *testing.T) {
	h := branchPushTestHistory(t)
	commands := branchSyncTestCommands(t, h.root)
	opts := BranchUpstreamOptions{TrackingRef: "refs/remotes/origin/selected"}
	plan, err := commands.PrepareBranchUpstream("selected", h.selected, opts)
	branchPushRequireNoError(t, err)
	configBefore := branchUpstreamConfigBytes(t, h.root)
	gitTest(t, h.root, "update-ref", "refs/remotes/origin/selected", h.mastertip, h.base)
	_, err = commands.BranchUpstreamCommand(plan)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	assert.Empty(t, branchPushConfig(t, h.root, "branch.selected.remote"))
	assert.Equal(t, configBefore, branchUpstreamConfigBytes(t, h.root))

	for _, invalid := range []BranchUpstreamOptions{{Unset: true, TrackingRef: "refs/heads/master"}, {TrackingRef: "refs/heads/selected"}, {TrackingRef: "refs/remotes/origin/alias"}} {
		if invalid.TrackingRef == "refs/remotes/origin/alias" {
			gitTest(t, h.root, "symbolic-ref", invalid.TrackingRef, "refs/remotes/origin/selected")
		}
		_, err := commands.PrepareBranchUpstream("selected", h.selected, invalid)
		assert.Error(t, err)
		assert.Equal(t, configBefore, branchUpstreamConfigBytes(t, h.root))
	}
	for _, invalid := range []BranchUpstreamOptions{{TrackingRef: "refs/heads/bad?name"}, {TrackingRef: "refs/cache/missing"}} {
		_, err := commands.PrepareBranchUpstream("selected", h.selected, invalid)
		assert.Error(t, err)
		assert.Equal(t, configBefore, branchUpstreamConfigBytes(t, h.root))
	}
	_, err = commands.PrepareBranchUpstream("selected", h.selected, BranchUpstreamOptions{Unset: true})
	assert.Error(t, err, "no upstream is a precondition failure")
	assert.False(t, errors.Is(err, ErrBranchSyncChanged))
	assert.Equal(t, configBefore, branchUpstreamConfigBytes(t, h.root))
}

func TestBranchUpstreamTracksCustomNamespaceAndPreservesCheckedOutBranch(t *testing.T) {
	h := branchPushTestHistory(t)
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/cache/origin/*")
	gitTest(t, h.root, "update-ref", "refs/cache/origin/selected", h.base)
	commands := branchSyncTestCommands(t, h.root)
	plan, err := commands.PrepareBranchUpstream("selected", h.selected, BranchUpstreamOptions{TrackingRef: "refs/cache/origin/selected"})
	branchPushRequireNoError(t, err)
	cmd, err := commands.BranchUpstreamCommand(plan)
	branchPushRequireNoError(t, err)
	output, err := cmd.GetCmd().CombinedOutput()
	branchPushRequireNoError(t, err, string(output))
	assert.Equal(t, "refs/heads/selected", gitTest(t, h.root, "config", "branch.selected.merge"))
	assert.Equal(t, "origin", gitTest(t, h.root, "config", "branch.selected.remote"))
	assert.Equal(t, h.mastertip, gitTest(t, h.root, "rev-parse", "HEAD"))
	localPlan, err := commands.PrepareBranchUpstream("selected", h.selected, BranchUpstreamOptions{TrackingRef: "refs/heads/master"})
	branchPushRequireNoError(t, err)
	cmd, err = commands.BranchUpstreamCommand(localPlan)
	branchPushRequireNoError(t, err)
	output, err = cmd.GetCmd().CombinedOutput()
	branchPushRequireNoError(t, err, string(output))
	assert.Equal(t, ".", gitTest(t, h.root, "config", "branch.selected.remote"))
	assert.Equal(t, "refs/heads/master", gitTest(t, h.root, "config", "branch.selected.merge"))

	_, err = commands.PrepareBranchUpstream("selected", h.selected, BranchUpstreamOptions{TrackingRef: "refs/heads/selected"})
	assert.Error(t, err)
	assert.Equal(t, "refs/heads/master", gitTest(t, h.root, "symbolic-ref", "HEAD"))
}

func TestBranchUpstreamStaleSourceProfileAndForeignBackend(t *testing.T) {
	h := branchPushTestHistory(t)
	commands := branchSyncTestCommands(t, h.root)
	opts := BranchUpstreamOptions{TrackingRef: "refs/cache/origin/selected"}
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/cache/origin/*")
	gitTest(t, h.root, "update-ref", opts.TrackingRef, h.base)
	plan, err := commands.PrepareBranchUpstream("selected", h.selected, opts)
	branchPushRequireNoError(t, err)
	configBefore := branchUpstreamConfigBytes(t, h.root)
	gitTest(t, h.root, "config", "--add", "remote.backup.fetch", "+refs/heads/*:refs/cache/origin/*")
	changedConfig := branchUpstreamConfigBytes(t, h.root)
	_, err = commands.BranchUpstreamCommand(plan)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	assert.Equal(t, changedConfig, branchUpstreamConfigBytes(t, h.root))
	assert.NotEqual(t, configBefore, changedConfig)
	assert.Empty(t, branchPushConfig(t, h.root, "branch.selected.remote"))

	plan, err = commands.PrepareBranchUpstream("selected", h.selected, opts)
	branchPushRequireNoError(t, err)
	gitTest(t, h.root, "update-ref", "refs/heads/selected", h.base, h.selected)
	changedConfig = branchUpstreamConfigBytes(t, h.root)
	_, err = commands.BranchUpstreamCommand(plan)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	assert.Equal(t, changedConfig, branchUpstreamConfigBytes(t, h.root))
	assert.Empty(t, branchPushConfig(t, h.root, "branch.selected.remote"))

	other := branchPushTestHistory(t)
	foreign := branchSyncTestCommands(t, other.root)
	plan, err = commands.PrepareBranchUpstream("selected", h.base, BranchUpstreamOptions{TrackingRef: "refs/heads/master"})
	branchPushRequireNoError(t, err)
	configBefore = branchUpstreamConfigBytes(t, other.root)
	_, err = foreign.BranchUpstreamCommand(plan)
	assert.Error(t, err)
	assert.Equal(t, configBefore, branchUpstreamConfigBytes(t, other.root))
	assert.Empty(t, branchPushConfig(t, other.root, "branch.selected.remote"))
}

func TestBranchUpstreamRefusesAmbiguousAndInheritedConfigForSetAndUnset(t *testing.T) {
	for _, source := range []string{"multiple-remote", "multiple-merge", "include", "worktree", "global"} {
		for _, unset := range []bool{false, true} {
			t.Run(source+map[bool]string{false: "-set", true: "-unset"}[unset], func(t *testing.T) {
				h := branchPushTestHistory(t)
				target := "refs/remotes/origin/selected"
				opts := BranchUpstreamOptions{TrackingRef: target}
				var inheritedFile string
				var inheritedBefore []byte
				if source == "multiple-remote" || source == "multiple-merge" {
					key, first, second := "branch.selected.remote", "origin", "backup"
					if source == "multiple-merge" {
						key, first, second = "branch.selected.merge", "refs/heads/selected", "refs/heads/master"
					}
					gitTest(t, h.root, "config", "--add", key, first)
					gitTest(t, h.root, "config", "--add", key, second)
				} else if source == "include" {
					inheritedFile = filepath.Join(t.TempDir(), "included-config")
					if err := os.WriteFile(inheritedFile, []byte(`[branch "selected"]
 remote = origin
 merge = refs/heads/selected
`), 0o600); err != nil {
						t.Fatal(err)
					}
					gitTest(t, h.root, "config", "include.path", inheritedFile)
					inheritedBefore = mustRead(t, inheritedFile)
				} else if source == "worktree" {
					gitTest(t, h.root, "config", "extensions.worktreeConfig", "true")
					gitTest(t, h.root, "config", "--worktree", "branch.selected.remote", "origin")
					gitTest(t, h.root, "config", "--worktree", "branch.selected.merge", "refs/heads/selected")
				} else {
					inheritedFile = filepath.Join(t.TempDir(), "global-config")
					if err := os.WriteFile(inheritedFile, []byte(`[branch "selected"]
 remote = origin
 merge = refs/heads/selected
`), 0o600); err != nil {
						t.Fatal(err)
					}
					inheritedBefore = mustRead(t, inheritedFile)
					old, had := os.LookupEnv("GIT_CONFIG_GLOBAL")
					if err := os.Setenv("GIT_CONFIG_GLOBAL", inheritedFile); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						if had {
							_ = os.Setenv("GIT_CONFIG_GLOBAL", old)
						} else {
							_ = os.Unsetenv("GIT_CONFIG_GLOBAL")
						}
					})
				}
				if unset {
					opts = BranchUpstreamOptions{Unset: true}
				}
				configBefore := branchUpstreamConfigBytes(t, h.root)
				_, err := branchSyncTestCommands(t, h.root).PrepareBranchUpstream("selected", h.selected, opts)
				assert.ErrorIs(t, err, ErrBranchUpstreamAmbiguous)
				assert.Equal(t, configBefore, branchUpstreamConfigBytes(t, h.root))
				if inheritedFile != "" {
					assert.Equal(t, inheritedBefore, mustRead(t, inheritedFile))
				}
			})
		}
	}
}

func TestBranchUpstreamAllowsLinkedWorktreeOccupancy(t *testing.T) {
	h := branchPushTestHistory(t)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, h.root, "worktree", "add", "-q", linked, "selected")
	trackedPath := filepath.Join(linked, "tracked")
	writeTestFile(t, trackedPath, []byte("linked dirty file\n"), 0o644)
	gitTest(t, linked, "add", "tracked")
	mainBefore := branchUpstreamWorktreeSnapshot(t, h.root)
	linkedBefore := branchUpstreamWorktreeSnapshot(t, linked)
	commands := branchSyncTestCommands(t, h.root)
	plan, err := commands.PrepareBranchUpstream("selected", h.selected, BranchUpstreamOptions{TrackingRef: "refs/remotes/origin/selected"})
	branchPushRequireNoError(t, err)
	cmd, err := commands.BranchUpstreamCommand(plan)
	branchPushRequireNoError(t, err)
	output, err := cmd.GetCmd().CombinedOutput()
	branchPushRequireNoError(t, err, string(output))
	assert.Equal(t, mainBefore, branchUpstreamWorktreeSnapshot(t, h.root))
	assert.Equal(t, linkedBefore, branchUpstreamWorktreeSnapshot(t, linked))
}

func branchUpstreamConfigBytes(t *testing.T, root string) []byte {
	t.Helper()
	config := mustRead(t, filepath.Join(root, ".git", "config"))
	worktreeConfig, err := os.ReadFile(filepath.Join(root, ".git", "config.worktree"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return append(config, worktreeConfig...)
}

func branchUpstreamWorktreeSnapshot(t *testing.T, root string) string {
	t.Helper()
	gitDir := gitTest(t, root, "rev-parse", "--git-dir")
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	paths := []string{"HEAD", "index"}
	state := gitTest(t, root, "status", "--porcelain=v1", "-z")
	for _, name := range []string{"tracked", "binary-untracked"} {
		bytes, err := os.ReadFile(filepath.Join(root, name))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		state += string(bytes)
	}
	for _, name := range paths {
		bytes, err := os.ReadFile(filepath.Join(gitDir, name))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		state += string(bytes)
	}
	return state
}
