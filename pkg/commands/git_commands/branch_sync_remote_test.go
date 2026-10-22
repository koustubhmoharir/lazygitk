package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestBranchSyncRemotePreservesAllMappings(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/fetch.git")
	gitTest(t, root, "config", "--add", "remote.origin.url", "https://example.invalid/fetch-two.git")
	gitTest(t, root, "config", "--add", "remote.origin.pushurl", "https://example.invalid/push-one.git")
	gitTest(t, root, "config", "--add", "remote.origin.pushurl", "https://example.invalid/push-two.git")
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/heads/topic:refs/cache/topic")
	gitTest(t, root, "config", "--add", "remote.origin.push", "refs/heads/topic:refs/heads/one")
	gitTest(t, root, "config", "--add", "remote.origin.push", "refs/heads/topic:refs/heads/two")
	gitTest(t, root, "config", "remote.origin.mirror", "yes")
	gitTest(t, root, "remote", "add", "origin.extra", "https://example.invalid/unrelated.git")
	commands := branchSyncTestCommands(t, root)
	beforeHead := gitTest(t, root, "rev-parse", "HEAD")
	beforeIndex := gitTest(t, root, "write-tree")
	info, err := commands.InspectBranchSyncRemote("origin")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"https://example.invalid/fetch.git", "https://example.invalid/fetch-two.git"}, info.URLs)
	assert.Equal(t, []string{"https://example.invalid/push-one.git", "https://example.invalid/push-two.git"}, info.PushURLs)
	assert.Equal(t, []string{"+refs/heads/*:refs/remotes/origin/*", "+refs/heads/topic:refs/cache/topic"}, info.FetchRefspecs)
	assert.Equal(t, []string{"refs/heads/topic:refs/heads/one", "refs/heads/topic:refs/heads/two"}, info.PushRefspecs)
	assert.True(t, info.Mirror)
	assert.Len(t, info.Signature, 64)
	info.URLs[0] = "mutated inspection result"
	fresh, err := commands.InspectBranchSyncRemote("origin")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.invalid/fetch.git", fresh.URLs[0])
	assert.Equal(t, info.Signature, fresh.Signature)
	assert.Equal(t, beforeHead, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, beforeIndex, gitTest(t, root, "write-tree"))
}

func TestBranchSyncRemoteFreshScopedSignature(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/one.git")
	gitTest(t, root, "remote", "add", "origin.extra", "https://example.invalid/other.git")
	commands := branchSyncTestCommands(t, root)
	inspect := func() BranchSyncRemoteInfo {
		t.Helper()
		info, err := commands.InspectBranchSyncRemote("origin")
		if err != nil {
			t.Fatal(err)
		}
		return info
	}
	first := inspect()
	assert.False(t, first.Mirror)
	gitTest(t, root, "config", "remote.origin.extra.url", "https://example.invalid/still-unrelated.git")
	gitTest(t, root, "config", "color.ui", "never")
	assert.Equal(t, first.Signature, inspect().Signature, "unrelated settings and dotted remote names do not invalidate this remote")
	gitTest(t, root, "config", "remote.origin.url", "https://example.invalid/two.git")
	second := inspect()
	assert.NotEqual(t, first.Signature, second.Signature)
	gitTest(t, root, "config", "url.https://replacement.invalid/.insteadOf", "https://example.invalid/")
	third := inspect()
	assert.NotEqual(t, second.Signature, third.Signature, "URL rewriting can change the actual endpoint")
	gitTest(t, root, "config", "remote.origin.mirror", "true")
	fourth := inspect()
	assert.True(t, fourth.Mirror)
	assert.NotEqual(t, third.Signature, fourth.Signature)
}

func TestBranchSyncRemoteValuelessBooleanAndPushOnly(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	// A native valueless boolean differs from an explicitly empty value.
	path := filepath.Join(root, ".git", "config")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\n[remote \"publish\"]\n\tpushurl = https://example.invalid/push.git\n\tmirror\n")...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	commands := branchSyncTestCommands(t, root)
	info, err := commands.InspectBranchSyncRemote("publish")
	if !assert.NoError(t, err) {
		return
	}
	assert.Empty(t, info.URLs)
	assert.Equal(t, []string{"https://example.invalid/push.git"}, info.PushURLs)
	assert.True(t, info.Mirror)
	gitTest(t, root, "config", "remote.publish.mirror", "")
	fresh, err := commands.InspectBranchSyncRemote("publish")
	assert.NoError(t, err)
	assert.False(t, fresh.Mirror)
	assert.NotEqual(t, info.Signature, fresh.Signature)
}

func TestBranchSyncRemoteInvalidAndEmptyConfiguration(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	for _, name := range []string{"", ".", "missing", "origin\x00injected", "origin\nother"} {
		_, err := commands.InspectBranchSyncRemote(name)
		assert.Error(t, err, "remote %q", name)
	}
	for _, url := range []string{"https://user:secret@example.invalid/repo", "user:secret@example.invalid:repo"} {
		_, err := commands.InspectBranchSyncRemote(url)
		assert.Error(t, err)
		if err != nil {
			assert.NotContains(t, err.Error(), "secret")
		}
	}
	gitTest(t, root, "config", "remote.empty.url", "")
	_, err := commands.InspectBranchSyncRemote("empty")
	assert.ErrorContains(t, err, "no configured fetch or push URL")
	gitTest(t, root, "remote", "add", "broken", "https://example.invalid/repo.git")
	gitTest(t, root, "config", "remote.broken.mirror", "not-a-boolean")
	_, err = commands.InspectBranchSyncRemote("broken")
	assert.ErrorContains(t, err, "inspect remote mirroring")
}

func TestBranchSyncRemoteConfigFraming(t *testing.T) {
	entries, err := parseBranchSyncConfig("remote.origin.url\nfile:///line\nbreak\x00remote.origin.mirror\x00remote.origin.empty\n\x00")
	assert.NoError(t, err)
	assert.Equal(t, []branchSyncConfigEntry{
		{key: "remote.origin.url", value: "file:///line\nbreak", hasValue: true},
		{key: "remote.origin.mirror"},
		{key: "remote.origin.empty", hasValue: true},
	}, entries)
	for _, malformed := range []string{"truncated", "\x00", "\nmissing-key\x00"} {
		_, err := parseBranchSyncConfig(malformed)
		assert.Error(t, err)
	}
}

func TestBranchSyncRemoteQueryFailure(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).ExpectGitArgs([]string{"config", "--null", "--list", "--includes"}, "", exec.ErrNotFound)
	commands := NewSyncCommands(buildGitCommon(commonDeps{runner: runner}))
	_, err := commands.InspectBranchSyncRemote("origin")
	assert.ErrorIs(t, err, exec.ErrNotFound)
	runner.CheckForMissingCalls()
}

func TestBranchSyncRemoteIncludesAreFresh(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	path := filepath.Join(root, "included-config")
	writeTestFile(t, path, []byte("[remote \"origin\"]\nurl = https://example.invalid/one.git\n"), 0o600)
	gitTest(t, root, "config", "include.path", path)
	commands := branchSyncTestCommands(t, root)
	first, err := commands.InspectBranchSyncRemote("origin")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"https://example.invalid/one.git"}, first.URLs)
	writeTestFile(t, path, []byte("[remote \"origin\"]\nurl = https://example.invalid/two.git\n"), 0o600)
	second, err := commands.InspectBranchSyncRemote("origin")
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://example.invalid/two.git"}, second.URLs)
	assert.NotEqual(t, first.Signature, second.Signature)
}

func TestBranchSyncRemoteEmptyURLValuesResetEarlierLists(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/old-fetch.git")
	gitTest(t, root, "config", "--add", "remote.origin.url", "")
	gitTest(t, root, "config", "--add", "remote.origin.url", "https://example.invalid/new-fetch.git")
	gitTest(t, root, "config", "--add", "remote.origin.pushurl", "https://example.invalid/old-push.git")
	gitTest(t, root, "config", "--add", "remote.origin.pushurl", "")
	gitTest(t, root, "config", "--add", "remote.origin.pushurl", "https://example.invalid/new-push.git")
	info, err := branchSyncTestCommands(t, root).InspectBranchSyncRemote("origin")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"https://example.invalid/new-fetch.git"}, info.URLs)
	assert.Equal(t, []string{"https://example.invalid/new-push.git"}, info.PushURLs)
	assert.Equal(t, info.URLs[0], gitTest(t, root, "remote", "get-url", "--all", "origin"))
	assert.Equal(t, info.PushURLs[0], gitTest(t, root, "remote", "get-url", "--push", "--all", "origin"))
}
