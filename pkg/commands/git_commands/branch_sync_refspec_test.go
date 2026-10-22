package git_commands

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestBranchSyncRefspecConfiguredAndOrderedMappings(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/repo.git")
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/heads/topic:refs/custom/exact")
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/custom/*")
	gitTest(t, root, "config", "branch.selected.remote", "origin")
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/topic")
	gitTest(t, root, "branch", "selected")
	gitTest(t, root, "update-ref", "refs/remotes/origin/topic", oid)
	info, err := branchSyncTestCommands(t, root).InspectBranchSync("selected")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "refs/remotes/origin/topic", info.UpstreamTrackingRef, "Git's native canonical mapping uses the first configured refspec")

	mapping, err := branchSyncTestCommands(t, root).branchSyncTrackingRefs([]string{
		"+refs/heads/*:refs/remotes/origin/*",
		"refs/heads/topic:refs/custom/exact",
		"refs/heads/*:refs/custom/*",
	}, "refs/heads/topic")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"refs/remotes/origin/topic", "refs/custom/exact", "refs/custom/topic"}, mapping.refs)
	assert.Equal(t, "refs/remotes/origin/topic", mapping.preferred)
}

func TestBranchSyncRefspecAbbreviatedSourceNativeBehavior(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, root, "init", "--bare", "-q", remote)
	gitTest(t, root, "push", remote, oid+":refs/heads/topic")
	gitTest(t, root, "remote", "add", "origin", remote)
	gitTest(t, root, "config", "--unset-all", "remote.origin.fetch")
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "topic:refs/cache/topic")

	fetch := exec.Command("git", "-C", root, "fetch", "origin")
	output, err := fetch.CombinedOutput()
	if !assert.NoError(t, err, string(output)) {
		return
	}
	assert.Equal(t, "refs/cache/topic", gitTest(t, root, "for-each-ref", "--format=%(refname)", "refs/cache"))
	gitTest(t, root, "update-ref", "-d", "refs/cache/topic", oid)
	advanced := branchPlacementTestCommit(t, root, "advance", "advance\n", "advance")
	gitTest(t, root, "push", "origin", advanced+":refs/heads/topic")
	assert.Equal(t, advanced, gitTest(t, remote, "rev-parse", "refs/heads/topic"))
	assert.Empty(t, gitTest(t, root, "for-each-ref", "--format=%(refname)", "refs/cache"), "explicit native push must not create a local tracking ref")

	mapping, err := branchSyncTestCommands(t, root).branchSyncTrackingRefs([]string{"topic:refs/cache/topic"}, "refs/heads/topic")
	if !assert.NoError(t, err) {
		return
	}
	assert.Empty(t, mapping.refs, "tracking lookup uses exact source equality for nonpattern refspecs")
	assert.Empty(t, mapping.preferred)
}

func TestBranchSyncRefspecWildcardsAndNegatives(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	for _, test := range []struct {
		name string
		spec string
		ref  string
		want string
	}{
		{"middle wildcard spans slashes", "refs/remotes/*/x:refs/cache/*/x", "refs/remotes/a/b/x", "refs/cache/a/b/x"},
		{"empty capture", "refs/heads/pre*post:refs/cache/pre*post", "refs/heads/prepost", "refs/cache/prepost"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mapping, err := commands.branchSyncTrackingRefs([]string{test.spec}, test.ref)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, []string{test.want}, mapping.refs)
		})
	}
	for _, test := range []struct {
		specs []string
		ref   string
		want  string
	}{
		{[]string{"^refs/heads/private", "refs/heads/*:refs/remotes/origin/*"}, "refs/heads/private", "refs/remotes/origin/private"},
		{[]string{"refs/heads/*:refs/remotes/origin/*", "^refs/heads/private"}, "refs/heads/private", "refs/remotes/origin/private"},
		{[]string{"refs/heads/*:refs/remotes/origin/*", "^refs/heads/private/*"}, "refs/heads/private/topic", "refs/remotes/origin/private/topic"},
	} {
		mapping, err := commands.branchSyncTrackingRefs(test.specs, test.ref)
		if !assert.NoError(t, err) {
			continue
		}
		assert.True(t, mapping.excluded)
		assert.Empty(t, mapping.preferred)
		assert.Equal(t, []string{test.want}, mapping.refs)
	}
}

func TestBranchSyncRefspecEmptyAndOmittedDestinations(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	mapping, err := commands.branchSyncTrackingRefs([]string{
		"refs/heads/topic",
		"refs/heads/topic:",
		"refs/heads/*:refs/remotes/origin/*",
	}, "refs/heads/topic")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"refs/remotes/origin/topic"}, mapping.refs)
	assert.Empty(t, mapping.preferred, "an explicit empty RHS is the first applicable mapping and suppresses a later preferred destination")

	mapping, err = commands.branchSyncTrackingRefs([]string{"refs/heads/topic", "refs/heads/topic:refs/cache/topic"}, "refs/heads/topic")
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"refs/cache/topic"}, mapping.refs)
		assert.Equal(t, "refs/cache/topic", mapping.preferred)
	}
	mapping, err = commands.branchSyncTrackingRefs([]string{"", "@:", "refs/heads/*:refs/cache/*"}, "refs/heads/topic")
	if assert.NoError(t, err) {
		assert.Equal(t, "refs/cache/topic", mapping.preferred)
	}
}

func TestBranchSyncRefspecRejectsMalformedSpecsAndExpansion(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	for _, spec := range []string{
		"+^refs/heads/*", "^refs/heads/*:refs/cache/*", "refs/heads/*", "refs/heads/**:refs/cache/**",
		"refs/heads/*:refs/cache", "refs/heads/:refs/cache/topic", "refs/heads/topic:refs/cache//topic", "refs/heads/topic\x00:refs/cache/topic",
		"refs/heads/topic:refs/cache/topic:extra", "0123456789012345678901234567890123456789",
		"^0123456789012345678901234567890123456789012345678901234567890123",
	} {
		_, err := commands.branchSyncTrackingRefs([]string{spec}, "refs/heads/topic")
		assert.Error(t, err, "spec %q", spec)
	}
	for _, remoteRef := range []string{"topic", "refs/heads/../topic", "refs/heads/topic\x00suffix", "refs/heads//topic"} {
		_, err := commands.branchSyncTrackingRefs([]string{"refs/heads/*:refs/cache/*"}, remoteRef)
		assert.Error(t, err, "remote ref %q", remoteRef)
	}
	_, err := commands.branchSyncTrackingRefs([]string{"refs/heads/*x:refs/cache/*/y"}, "refs/heads/x")
	assert.ErrorContains(t, err, "expanded fetch destination")
}

func TestBranchSyncRefspecQueryFailure(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).ExpectGitArgs([]string{"check-ref-format", "--allow-onelevel", "refs/heads/topic"}, "", exec.ErrNotFound)
	commands := NewSyncCommands(buildGitCommon(commonDeps{runner: runner}))
	_, err := commands.branchSyncTrackingRefs(nil, "refs/heads/topic")
	assert.ErrorIs(t, err, exec.ErrNotFound)
	runner.CheckForMissingCalls()
}
