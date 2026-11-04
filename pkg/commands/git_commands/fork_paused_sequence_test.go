package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stefanhaller/git-todo-parser/todo"
	"github.com/stretchr/testify/assert"
)

// pausedSequenceRepo returns a repo whose topic branch has three commits, the
// first of which changes f. main then changes f too, so picking or reverting
// the first commit conflicts. It returns the topic commits, oldest first, and
// leaves main checked out.
func pausedSequenceRepo(t *testing.T) (root string, commits []string) {
	t.Helper()
	root = t.TempDir()
	gitTest(t, root, "init", "-q", "-b", "main")
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	writeTestFile(t, filepath.Join(root, "f"), []byte("base\n"), 0o600)
	gitTest(t, root, "add", "f")
	gitTest(t, root, "commit", "-qm", "base")
	gitTest(t, root, "checkout", "-q", "-b", "topic")
	writeTestFile(t, filepath.Join(root, "f"), []byte("one\n"), 0o600)
	gitTest(t, root, "commit", "-qam", "one")
	commits = append(commits, gitTest(t, root, "rev-parse", "HEAD"))
	for _, name := range []string{"two", "three"} {
		writeTestFile(t, filepath.Join(root, name), []byte(name+"\n"), 0o600)
		gitTest(t, root, "add", name)
		gitTest(t, root, "commit", "-qm", name)
		commits = append(commits, gitTest(t, root, "rev-parse", "HEAD"))
	}
	gitTest(t, root, "checkout", "-q", "main")
	writeTestFile(t, filepath.Join(root, "f"), []byte("later\n"), 0o600)
	gitTest(t, root, "commit", "-qam", "later")
	return root, commits
}

func pausedSequenceStatus(t *testing.T, root string) *StatusCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	common := NewGitCommon(nil, nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil)
	return NewStatusCommands(common)
}

// gitStopsOnConflict runs a git command that is expected to stop on a conflict.
func gitStopsOnConflict(t *testing.T, root string, args ...string) {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if assert.Error(t, err, "expected a conflict: %s", output) {
		assert.Contains(t, string(output), "CONFLICT")
	}
}

// resolveAndCommit resolves the conflict on f and commits, leaving any
// sequence in progress.
func resolveAndCommit(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, filepath.Join(root, "f"), []byte("resolved\n"), 0o600)
	gitTest(t, root, "add", "f")
	gitTest(t, root, "commit", "-qm", "resolved")
}

func TestPausedSequenceCherryPickAfterCommit(t *testing.T) {
	root, commits := pausedSequenceRepo(t)
	gitStopsOnConflict(t, root, "cherry-pick", commits[0], commits[1], commits[2])
	resolveAndCommit(t, root)

	assert.NoFileExists(t, filepath.Join(root, ".git", "CHERRY_PICK_HEAD"))
	assert.DirExists(t, filepath.Join(root, ".git", "sequencer"))

	state := pausedSequenceStatus(t, root).WorkingTreeState()
	assert.True(t, state.CherryPicking)
	assert.False(t, state.Reverting)
	assert.True(t, state.SequencePaused)
	assert.True(t, state.Any())
}

func TestPausedSequenceRevertAfterCommit(t *testing.T) {
	root, commits := pausedSequenceRepo(t)
	gitStopsOnConflict(t, root, "revert", "--no-edit", commits[0], commits[1], commits[2])
	resolveAndCommit(t, root)

	assert.NoFileExists(t, filepath.Join(root, ".git", "REVERT_HEAD"))
	assert.DirExists(t, filepath.Join(root, ".git", "sequencer"))

	state := pausedSequenceStatus(t, root).WorkingTreeState()
	assert.False(t, state.CherryPicking)
	assert.True(t, state.Reverting)
	assert.True(t, state.SequencePaused)
	assert.True(t, state.Any())
}

func TestPausedSequenceMidConflict(t *testing.T) {
	root, commits := pausedSequenceRepo(t)
	gitStopsOnConflict(t, root, "cherry-pick", commits[0], commits[1], commits[2])

	assert.FileExists(t, filepath.Join(root, ".git", "CHERRY_PICK_HEAD"))
	state := pausedSequenceStatus(t, root).WorkingTreeState()
	assert.True(t, state.CherryPicking)
	assert.False(t, state.SequencePaused)
}

func TestPausedSequenceSingleCommitPick(t *testing.T) {
	root, commits := pausedSequenceRepo(t)
	gitStopsOnConflict(t, root, "cherry-pick", commits[0])

	// A single-commit pick does not use the sequencer.
	assert.NoDirExists(t, filepath.Join(root, ".git", "sequencer"))
	state := pausedSequenceStatus(t, root).WorkingTreeState()
	assert.True(t, state.CherryPicking)
	assert.False(t, state.SequencePaused)

	resolveAndCommit(t, root)
	state = pausedSequenceStatus(t, root).WorkingTreeState()
	assert.False(t, state.Any())
	assert.False(t, state.SequencePaused)
}

func TestPausedSequenceNoOperation(t *testing.T) {
	root, _ := pausedSequenceRepo(t)

	state := pausedSequenceStatus(t, root).WorkingTreeState()
	assert.False(t, state.Any())
	assert.False(t, state.SequencePaused)
}

func TestPausedSequenceFirstCommand(t *testing.T) {
	scenarios := []struct {
		name     string
		content  string
		expected todo.TodoCommand
	}{
		{name: "pick", content: "pick abc123 subject\n", expected: todo.Pick},
		{name: "short pick", content: "p abc123 subject\n", expected: todo.Pick},
		{name: "revert", content: "revert abc123 subject\n", expected: todo.Revert},
		{name: "short revert", content: "r abc123 subject\n", expected: todo.Revert},
		{name: "skips blank and comment lines", content: "\n# comment\n   \nrevert abc123 subject\n", expected: todo.Revert},
		{name: "other command", content: "exec make\n", expected: noSequencerCommand},
		{name: "empty", content: "", expected: noSequencerCommand},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			assert.Equal(t, scenario.expected, firstSequencerCommand([]byte(scenario.content)))
		})
	}
}

func TestPausedSequenceFirstEntryCommitted(t *testing.T) {
	root, commits := pausedSequenceRepo(t)
	gitStopsOnConflict(t, root, "cherry-pick", commits[0], commits[1], commits[2])
	common := pausedSequenceStatus(t, root).GitCommon

	assert.False(t, common.pausedSequenceFirstEntryCommitted(), "conflicted pick is not committed")

	resolveAndCommit(t, root)
	assert.True(t, common.pausedSequenceFirstEntryCommitted(), "committed pick differs from abort-safety")

	// Git's `--continue` re-picks the first entry when abort-safety equals HEAD.
	head := gitTest(t, root, "rev-parse", "HEAD")
	abortSafety := filepath.Join(root, ".git", "sequencer", "abort-safety")
	assert.NoError(t, os.WriteFile(abortSafety, []byte(head+"\n"), 0o600))
	assert.False(t, common.pausedSequenceFirstEntryCommitted(), "abort-safety equal to HEAD is not committed")
}
