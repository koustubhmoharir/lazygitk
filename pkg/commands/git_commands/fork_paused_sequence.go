package git_commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stefanhaller/git-todo-parser/todo"
)

// Fork: a cherry-pick or revert sequence stays in progress after the pick that
// stopped it is committed. Git removes CHERRY_PICK_HEAD or REVERT_HEAD but keeps
// .git/sequencer/, whose todo still lists that pick first, and `git status`
// still reports the operation. Git decides whether that first entry is finished
// by comparing sequencer/abort-safety with HEAD: if they differ, the entry was
// committed by the user; if they are equal, `--continue` picks it again. The
// helpers here follow the same rule. status.go, the working tree state and
// commit_loader.go use them.

const (
	sequencerTodoPath        = "sequencer/todo"
	sequencerAbortSafetyPath = "sequencer/abort-safety"
	// noSequencerCommand is returned when there is no pick or revert to report.
	noSequencerCommand todo.TodoCommand = 0
)

// pausedSequenceCommand returns the command of the first entry in
// sequencer/todo, as todo.Pick or todo.Revert (the short forms are accepted).
// It returns noSequencerCommand when there is no todo file, no entry, or an entry that is not a
// pick or revert.
func (self *GitCommon) pausedSequenceCommand() (todo.TodoCommand, error) {
	content, err := os.ReadFile(filepath.Join(self.repoPaths.WorktreeGitDirPath(), sequencerTodoPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return noSequencerCommand, nil
		}
		return noSequencerCommand, err
	}
	return firstSequencerCommand(content), nil
}

// isPausedSequence reports whether the sequencer's first entry is a command of
// the given kind, i.e. whether a sequence of that kind is paused.
func (self *GitCommon) isPausedSequence(command todo.TodoCommand) (bool, error) {
	first, err := self.pausedSequenceCommand()
	return first == command, err
}

// pausedSequenceFirstEntryCommitted reports whether Git considers the first
// entry of sequencer/todo finished, which is the case when abort-safety differs
// from HEAD. A missing or unreadable abort-safety file, or a HEAD that cannot be
// resolved, counts as not finished, so the entry stays listed.
func (self *GitCommon) pausedSequenceFirstEntryCommitted() bool {
	abortSafety, err := os.ReadFile(filepath.Join(self.repoPaths.WorktreeGitDirPath(), sequencerAbortSafetyPath))
	if err != nil {
		return false
	}
	headArgs := NewGitCmd("rev-parse").Arg("HEAD").ToArgv()
	head, err := self.cmd.New(headArgs).DontLog().RunWithOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(abortSafety)) != strings.TrimSpace(head)
}

// withoutCommittedSequencerPick drops the first sequencer entry from commits
// when Git considers it finished. The loader lists sequencer entries in reverse,
// so that entry is the last element.
func (self *GitCommon) withoutCommittedSequencerPick(commits []*models.Commit) []*models.Commit {
	if len(commits) == 0 || !self.pausedSequenceFirstEntryCommitted() {
		return commits
	}
	return commits[:len(commits)-1]
}

// firstSequencerCommand returns the command of the first non-blank, non-comment
// line of a sequencer todo file, or noSequencerCommand if that command is not a pick or revert.
func firstSequencerCommand(content []byte) todo.TodoCommand {
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		switch fields[0] {
		case "pick", "p":
			return todo.Pick
		case "revert", "r":
			return todo.Revert
		default:
			return noSequencerCommand
		}
	}
	return noSequencerCommand
}
