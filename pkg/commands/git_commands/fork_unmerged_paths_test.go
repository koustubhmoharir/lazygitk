package git_commands

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmergedPathsReportsConflictsUntilStaged(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "-q", "-b", "main")
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	// The spaced name checks that paths are split on NUL, not on whitespace.
	for _, name := range []string{"a file", "f"} {
		writeTestFile(t, filepath.Join(root, name), []byte("base\n"), 0o600)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-qm", "base")
	gitTest(t, root, "checkout", "-q", "-b", "topic")
	for _, name := range []string{"a file", "f"} {
		writeTestFile(t, filepath.Join(root, name), []byte("topic\n"), 0o600)
	}
	gitTest(t, root, "commit", "-qam", "topic")
	gitTest(t, root, "checkout", "-q", "main")
	for _, name := range []string{"a file", "f"} {
		writeTestFile(t, filepath.Join(root, name), []byte("main\n"), 0o600)
	}
	gitTest(t, root, "commit", "-qam", "main")
	gitStopsOnConflict(t, root, "merge", "--no-edit", "topic")

	status := pausedSequenceStatus(t, root)
	paths, err := status.UnmergedPaths()
	assert.NoError(t, err)
	assert.Equal(t, []string{"a file", "f"}, paths)

	// Removing the markers does not resolve a path until it is staged.
	writeTestFile(t, filepath.Join(root, "f"), []byte("resolved\n"), 0o600)
	paths, err = status.UnmergedPaths()
	assert.NoError(t, err)
	assert.Equal(t, []string{"a file", "f"}, paths)

	gitTest(t, root, "add", "f")
	paths, err = status.UnmergedPaths()
	assert.NoError(t, err)
	assert.Equal(t, []string{"a file"}, paths)

	gitTest(t, root, "add", "a file")
	paths, err = status.UnmergedPaths()
	assert.NoError(t, err)
	assert.Empty(t, paths)
}
