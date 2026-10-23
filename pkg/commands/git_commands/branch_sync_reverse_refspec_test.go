package git_commands

import (
	"os/exec"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestBranchSyncReverseRefspecMappings(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)

	tests := []struct {
		name string
		spec string
		ref  string
		want []string
	}{
		{"conventional wildcard", "refs/heads/*:refs/remotes/origin/*", "refs/remotes/origin/topic", []string{"refs/heads/topic"}},
		{"custom namespaces", "refs/remotes/*/x:refs/cache/*/x", "refs/cache/a/b/x", []string{"refs/remotes/a/b/x"}},
		{"prefix and suffix wildcard", "refs/heads/pre*post:refs/cache/pre*post", "refs/cache/preab/post", []string{"refs/heads/preab/post"}},
		{"exact mapping", "refs/tags/release:refs/cache/current", "refs/cache/current", []string{"refs/tags/release"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := commands.branchSyncRemoteRefsForTracking([]string{test.spec}, test.ref)
			if assert.NoError(t, err) {
				assert.Equal(t, test.want, got)
			}
		})
	}
}

func TestBranchSyncReverseRefspecDeduplicatesAndReturnsAllInOrder(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	got, err := commands.branchSyncRemoteRefsForTracking([]string{
		"refs/heads/topic:refs/cache/topic",
		"refs/heads/*:refs/cache/*",
		"refs/tags/topic:refs/cache/topic",
		"refs/heads/topic:refs/cache/topic",
	}, "refs/cache/topic")
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"refs/heads/topic", "refs/tags/topic"}, got)
	}
}

func TestBranchSyncReverseRefspecNegativesDoNotEraseCandidates(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	for _, specs := range [][]string{
		{"^refs/heads/private", "refs/heads/*:refs/cache/*"},
		{"refs/heads/*:refs/cache/*", "^refs/heads/private"},
	} {
		got, err := commands.branchSyncRemoteRefsForTracking(specs, "refs/cache/private")
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"refs/heads/private"}, got)
		}
	}
}

func TestBranchSyncReverseRefspecIgnoresMissingDestinations(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	got, err := commands.branchSyncRemoteRefsForTracking([]string{
		"refs/heads/topic",
		"refs/heads/topic:",
		"refs/heads/other:refs/cache/other",
	}, "refs/cache/topic")
	if assert.NoError(t, err) {
		assert.Empty(t, got)
	}
}

func TestBranchSyncReverseRefspecAbbreviatedSource(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	got, err := commands.branchSyncRemoteRefsForTracking([]string{"topic:refs/cache/topic"}, "refs/cache/elsewhere")
	if assert.NoError(t, err) {
		assert.Empty(t, got)
	}

	_, err = commands.branchSyncRemoteRefsForTracking([]string{"topic*:refs/cache/*"}, "refs/cache/topic")
	assert.ErrorContains(t, err, "matching fetch source")
}

func TestBranchSyncReverseRefspecMalformedConfigurationDoesNotEchoCredentials(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	_, err := commands.branchSyncRemoteRefsForTracking([]string{"https://user:private-token@example.invalid/repo"}, "refs/cache/topic")
	if assert.Error(t, err) {
		assert.NotContains(t, err.Error(), "private-token")
		assert.NotContains(t, err.Error(), "https://")
	}
}

func TestBranchSyncReverseRefspecRejectsMalformedSpecsAndRefs(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	for _, spec := range []string{
		"refs/heads/topic\x00:refs/cache/topic",
		"refs/heads/**:refs/cache/**",
		"+^refs/heads/topic",
		"^refs/heads/topic:refs/cache/topic",
		"^refs/heads/**",
		"^0123456789012345678901234567890123456789",
		"0123456789012345678901234567890123456789",
	} {
		_, err := commands.branchSyncRemoteRefsForTracking([]string{spec}, "refs/cache/topic")
		assert.Error(t, err, "spec %q", spec)
	}
	for _, ref := range []string{"cache/topic", "refs/cache/../topic", "refs/cache//topic", "refs/cache/topic\x00"} {
		_, err := commands.branchSyncRemoteRefsForTracking(nil, ref)
		assert.Error(t, err, "tracking ref %q", ref)
	}
}

func TestBranchSyncReverseRefspecQueryFailure(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).ExpectGitArgs([]string{"check-ref-format", "--allow-onelevel", "refs/cache/topic"}, "", exec.ErrNotFound)
	commands := NewSyncCommands(buildGitCommon(commonDeps{runner: runner}))
	_, err := commands.branchSyncRemoteRefsForTracking(nil, "refs/cache/topic")
	assert.ErrorIs(t, err, exec.ErrNotFound)
	runner.CheckForMissingCalls()
}
