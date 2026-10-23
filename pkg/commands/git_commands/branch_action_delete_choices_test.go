package git_commands

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchActionDeleteChoicesContextRetainsSourcesAndConfiguration(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", base)
	gitTest(t, root, "remote", "add", "peer=one", filepath.Join(t.TempDir(), "remote.git"))
	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/heads/selected", base)
	if !assert.NoError(t, err) {
		return
	}
	context, err := branches.PrepareBranchActionDeleteContext([]BranchActionTarget{target})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"peer=one"}, context.Remotes())
	before := mergePreviewSnapshot(t, root)
	sources := context.Targets()
	sources[0] = BranchActionTarget{}
	remotes := context.Remotes()
	remotes[0] = "changed"
	assert.NoError(t, branches.CheckBranchActionDeleteContext(context))
	assert.Equal(t, before, mergePreviewSnapshot(t, root))
	assert.Error(t, transactionCommands(t, root).CheckBranchActionDeleteContext(context))
	gitTest(t, root, "config", "remote.peer=one.url", filepath.Join(t.TempDir(), "different.git"))
	assert.ErrorIs(t, branches.CheckBranchActionDeleteContext(context), ErrBranchActionChanged)
}

func TestBranchActionDeleteChoicesLocalRetainsEveryMergeRefReadOnly(t *testing.T) {
	root, oid, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", oid)
	gitTest(t, root, "remote", "add", "origin", filepath.Join(root, "missing-bare.git"))
	gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/custom/cache/*")
	gitTest(t, root, "config", "branch.selected.remote", "origin")
	gitTest(t, root, "config", "--add", "branch.selected.merge", "refs/heads/review")
	gitTest(t, root, "config", "--add", "branch.selected.merge", "refs/heads/other-review")
	gitTest(t, root, "config", "--add", "branch.selected.merge", "refs/heads/review")
	gitTest(t, root, "update-ref", "refs/custom/cache/review", oid)
	gitTest(t, root, "update-ref", "refs/custom/cache/other-review", oid)
	gitTest(t, root, "update-ref", "refs/heads/selected", oid)
	gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/master")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "untracked"), []byte("untracked\n"), 0o644)

	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/heads/selected", oid)
	if !assert.NoError(t, err) {
		t.Fatal("target preparation failed")
	}
	commands := NewSyncCommands(branches.GitCommon)
	indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
	headBefore := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	statusBefore := gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all")
	choices, err := commands.InspectBranchActionLocalDeleteUpstreams(target)
	if !assert.NoError(t, err) {
		t.Fatal("upstream inspection failed")
	}
	assert.Equal(t, "origin", choices.Remote())
	assert.Equal(t, []string{"refs/heads/review", "refs/heads/other-review"}, choices.Refs())
	refs := choices.Refs()
	refs[0] = "refs/heads/changed"
	assert.Equal(t, []string{"refs/heads/review", "refs/heads/other-review"}, choices.Refs(), "getter must clone")
	assert.NoError(t, commands.CheckBranchActionRemoteDeleteChoices(choices))
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, statusBefore, gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all"))
}

func TestBranchActionDeleteChoicesLocalDotAndInvalidMergeRefs(t *testing.T) {
	root, oid, _ := transactionHistory(t)
	gitTest(t, root, "branch", "selected", oid)
	gitTest(t, root, "config", "branch.selected.remote", ".")
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/master")
	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/heads/selected", oid)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	choices, err := NewSyncCommands(branches.GitCommon).InspectBranchActionLocalDeleteUpstreams(target)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, ".", choices.Remote())
	assert.Equal(t, []string{"refs/heads/master"}, choices.Refs())

	for _, merge := range []string{"master", "refs/heads/../bad"} {
		gitTest(t, root, "config", "--unset-all", "branch.selected.merge")
		gitTest(t, root, "config", "branch.selected.merge", merge)
		_, err := NewSyncCommands(branches.GitCommon).InspectBranchActionLocalDeleteUpstreams(target)
		assert.Error(t, err, "must reject malformed or abbreviated merge %q", merge)
	}
}

func TestBranchActionDeleteChoicesNoUpstreamAndStaleLocalConfig(t *testing.T) {
	root, oid, next := transactionHistory(t)
	gitTest(t, root, "branch", "selected", oid)
	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/heads/selected", oid)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	commands := NewSyncCommands(branches.GitCommon)
	_, err = commands.InspectBranchActionLocalDeleteUpstreams(target)
	assert.ErrorIs(t, err, ErrBranchDeleteNoUpstream)

	gitTest(t, root, "remote", "add", "origin", filepath.Join(root, "unused.git"))
	gitTest(t, root, "config", "branch.selected.remote", "origin")
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/review")
	choices, err := commands.InspectBranchActionLocalDeleteUpstreams(target)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/changed")
	assert.Error(t, commands.CheckBranchActionRemoteDeleteChoices(choices))
	gitTest(t, root, "update-ref", "refs/heads/selected", next, oid)
	assert.Error(t, commands.CheckBranchActionRemoteDeleteChoices(choices))
}

func TestBranchActionDeleteChoicesRemoteUsesExactInverseMapping(t *testing.T) {
	root, oid, _ := transactionHistory(t)
	gitTest(t, root, "remote", "add", "origin", filepath.Join(root, "unused.git"))
	gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/review/*:refs/custom/cache/*")
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/tags/review/*:refs/custom/cache/*")
	gitTest(t, root, "update-ref", "refs/custom/cache/topic", oid)
	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/custom/cache/topic", oid)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	choices, err := NewSyncCommands(branches.GitCommon).InspectBranchActionRemoteDeleteChoices(target, "origin")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"refs/heads/review/topic", "refs/tags/review/topic"}, choices.Refs())
	assert.NoError(t, NewSyncCommands(branches.GitCommon).CheckBranchActionRemoteDeleteChoices(choices))
}

func TestBranchActionDeleteChoicesRemoteStaleForeignAndReadOnly(t *testing.T) {
	root, oid, next := transactionHistory(t)
	gitTest(t, root, "remote", "add", "origin", filepath.Join(root, "unused.git"))
	gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/cache/*")
	gitTest(t, root, "update-ref", "refs/cache/topic", oid)
	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/cache/topic", oid)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	commands := NewSyncCommands(branches.GitCommon)
	choices, err := commands.InspectBranchActionRemoteDeleteChoices(target, "origin")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	headBefore := mustRead(t, filepath.Join(root, ".git", "HEAD"))
	indexBefore := mustRead(t, filepath.Join(root, ".git", "index"))
	statusBefore := gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all")
	gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/topic:refs/cache/topic")
	assert.Error(t, commands.CheckBranchActionRemoteDeleteChoices(choices))
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
	assert.Equal(t, statusBefore, gitTest(t, root, "status", "--porcelain=v2", "--untracked-files=all"))

	gitTest(t, root, "update-ref", "refs/cache/topic", next, oid)
	assert.Error(t, commands.CheckBranchActionRemoteDeleteChoices(choices))
	other := transactionCommands(t, root)
	assert.Error(t, NewSyncCommands(other.GitCommon).CheckBranchActionRemoteDeleteChoices(choices))
}
