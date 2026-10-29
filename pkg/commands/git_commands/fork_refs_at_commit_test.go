package git_commands

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

const forkRefsAtCommitHash = "78976bc0f0b5d6c2d2e6a7a9b6c1a2b3c4d5e6f7"

func TestRefsAtCommit(t *testing.T) {
	forEachRefArgs := []string{"for-each-ref", "--points-at=" + forkRefsAtCommitHash, "--format=%(symref)%00%(refname)", "refs/heads", "refs/remotes", "refs/tags"}

	type scenario struct {
		testName    string
		runner      *oscommands.FakeCmdObjRunner
		expected    []string
		expectError bool
	}
	scenarios := []scenario{
		{
			testName: "Lists local branches, remote branches and tags in for-each-ref order",
			runner: oscommands.NewFakeRunner(t).ExpectGitArgs(forEachRefArgs,
				"\x00refs/heads/main\n"+
					"\x00refs/heads/side\n"+
					"\x00refs/remotes/origin/feat\n"+
					"\x00refs/tags/v1.0\n", nil),
			expected: []string{"refs/heads/main", "refs/heads/side", "refs/remotes/origin/feat", "refs/tags/v1.0"},
		},
		{
			testName: "Skips symbolic refs such as origin/HEAD",
			runner: oscommands.NewFakeRunner(t).ExpectGitArgs(forEachRefArgs,
				"refs/remotes/origin/feat\x00refs/remotes/origin/HEAD\n"+
					"\x00refs/remotes/origin/feat\n", nil),
			expected: []string{"refs/remotes/origin/feat"},
		},
		{
			testName: "No ref points at the commit",
			runner: oscommands.NewFakeRunner(t).ExpectGitArgs(forEachRefArgs,
				"", nil),
			expected: []string{},
		},
		{
			testName: "Git fails",
			runner: oscommands.NewFakeRunner(t).ExpectGitArgs(forEachRefArgs,
				"", errors.New("git failed")),
			expectError: true,
		},
	}
	for _, s := range scenarios {
		t.Run(s.testName, func(t *testing.T) {
			instance := buildCommitCommands(commonDeps{runner: s.runner})

			refs, err := instance.RefsAtCommit(forkRefsAtCommitHash)
			if s.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, s.expected, refs)
			}
			s.runner.CheckForMissingCalls()
		})
	}
}

func TestRefsAtCommitRejectsInvalidHash(t *testing.T) {
	runner := oscommands.NewFakeRunner(t)
	instance := buildCommitCommands(commonDeps{runner: runner})

	_, err := instance.RefsAtCommit("not-a-hash")
	assert.Error(t, err)
	runner.CheckForMissingCalls()
}
