package git_commands

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

const forkRefCommitHash = "78976bc0f0b5d6c2d2e6a7a9b6c1a2b3c4d5e6f7"

func TestResolveRefCommitHash(t *testing.T) {
	type scenario struct {
		testName    string
		runner      *oscommands.FakeCmdObjRunner
		expected    string
		expectError bool
	}
	scenarios := []scenario{
		{
			testName: "Peels the ref to its commit",
			runner: oscommands.NewFakeRunner(t).ExpectGitArgs(
				[]string{"rev-parse", "--verify", "--end-of-options", "refs/tags/v1^{commit}"},
				forkRefCommitHash+"\n", nil),
			expected: forkRefCommitHash,
		},
		{
			testName: "Git fails",
			runner: oscommands.NewFakeRunner(t).ExpectGitArgs(
				[]string{"rev-parse", "--verify", "--end-of-options", "refs/tags/v1^{commit}"},
				"", errors.New("git failed")),
			expectError: true,
		},
		{
			testName: "Git returns something that is not an object ID",
			runner: oscommands.NewFakeRunner(t).ExpectGitArgs(
				[]string{"rev-parse", "--verify", "--end-of-options", "refs/tags/v1^{commit}"},
				"not-a-hash\n", nil),
			expectError: true,
		},
	}
	for _, s := range scenarios {
		t.Run(s.testName, func(t *testing.T) {
			instance := buildCommitCommands(commonDeps{runner: s.runner})

			hash, err := instance.ResolveRefCommitHash("refs/tags/v1")
			if s.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, s.expected, hash)
			}
			s.runner.CheckForMissingCalls()
		})
	}
}

func TestTagsPointingAt(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).ExpectGitArgs(
		[]string{"tag", "--points-at", forkRefCommitHash},
		"v1\nv1.0-rc\n", nil)
	instance := buildCommitCommands(commonDeps{runner: runner})

	tags, err := instance.TagsPointingAt(forkRefCommitHash)
	assert.NoError(t, err)
	assert.Equal(t, []string{"v1", "v1.0-rc"}, tags)
	runner.CheckForMissingCalls()
}

func TestTagsPointingAtNoTags(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).ExpectGitArgs(
		[]string{"tag", "--points-at", forkRefCommitHash},
		"", nil)
	instance := buildCommitCommands(commonDeps{runner: runner})

	tags, err := instance.TagsPointingAt(forkRefCommitHash)
	assert.NoError(t, err)
	assert.Empty(t, tags)
	runner.CheckForMissingCalls()
}

func TestTagsPointingAtRejectsInvalidHash(t *testing.T) {
	// No git call is expected: the argument must be a full object ID.
	runner := oscommands.NewFakeRunner(t)
	instance := buildCommitCommands(commonDeps{runner: runner})

	_, err := instance.TagsPointingAt("--all")
	assert.Error(t, err)
	runner.CheckForMissingCalls()
}
