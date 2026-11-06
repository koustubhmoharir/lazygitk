package git_commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// Fork: private object storage for Git commands that write objects. The isolated
// merge preview and the rebase ignored-file prediction both run merge-tree and
// commit-tree here, so the repository gets no loose objects, refs, index or
// worktree changes.

// privateObjectStore is a temporary object directory. Commands run in its
// environment read the repository's objects through an alternate and write only
// into the temporary directory.
type privateObjectStore struct {
	dir      string
	objects  string
	worktree string
}

func (self *GitCommon) newPrivateObjectStore() (*privateObjectStore, error) {
	output, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--path-format=absolute", "--git-path", "objects").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}
	objects := trimTerminalLineEnding(output)
	if objects == "" {
		return nil, fmt.Errorf("Git returned an empty object directory")
	}
	dir, err := os.MkdirTemp("", "lazygitk-merge-objects-")
	if err != nil {
		return nil, err
	}
	return &privateObjectStore{dir: dir, objects: objects, worktree: self.repoPaths.WorktreePath()}, nil
}

// environment routes a command's object reads and writes into the store.
func (store *privateObjectStore) environment(cmd *oscommands.CmdObj) *oscommands.CmdObj {
	alternates := quoteMergeAlternate(store.objects)
	// Inspect the command's effective environment, not just this process:
	// the builder may already supply repository-specific object stores.
	var inherited string
	for _, entry := range cmd.GetCmd().Environ() {
		if value, found := strings.CutPrefix(entry, "GIT_ALTERNATE_OBJECT_DIRECTORIES="); found {
			inherited = value
		}
	}
	if inherited != "" {
		alternates += string(os.PathListSeparator) + inherited
	}
	return cmd.RemoveEnvVar("GIT_OBJECT_DIRECTORY").RemoveEnvVar("GIT_ALTERNATE_OBJECT_DIRECTORIES").
		AddEnvVars("GIT_OBJECT_DIRECTORY="+store.dir, "GIT_ALTERNATE_OBJECT_DIRECTORIES="+alternates).
		SetWd(store.worktree).DontLog().DisableLockRetries()
}

func (store *privateObjectStore) remove() error {
	return os.RemoveAll(store.dir)
}

// writeMergeTreeInStore runs the merge-tree command that args builds, in store, and
// returns its result tree. Exit 1 denotes conflicts, not a failure: the result tree
// still describes every checkout path.
func (self *GitCommon) writeMergeTreeInStore(store *privateObjectStore, args *GitCommandBuilder) (string, error) {
	stdout, _, err := store.environment(self.cmd.New(args.ToArgv())).RunWithOutputs()
	if err != nil && !commandExitedWithOne(err) {
		return "", err
	}
	tree, parseErr := mergePreviewTreeID(stdout, err != nil)
	if parseErr != nil {
		return "", errors.Join(parseErr, err)
	}
	return tree, nil
}

// commitTreeInStore links the next merge base of a chain in store. The commit is
// throwaway, so its identity is fixed and signing is off: no user configuration can
// prompt or fail here.
func (self *GitCommon) commitTreeInStore(store *privateObjectStore, tree, parent string) (string, error) {
	cmd := store.environment(self.cmd.New(NewGitCmd("commit-tree").Arg(tree, "-p", parent, "-m", "lazygitk prediction", "--no-gpg-sign").ToArgv())).
		AddEnvVars("GIT_AUTHOR_NAME=lazygitk", "GIT_AUTHOR_EMAIL=lazygitk@invalid", "GIT_COMMITTER_NAME=lazygitk", "GIT_COMMITTER_EMAIL=lazygitk@invalid")
	output, err := cmd.RunWithOutput()
	if err != nil {
		return "", err
	}
	oid := trimTerminalLineEnding(output)
	if !IsFullObjectID(oid) {
		return "", fmt.Errorf("Git returned an invalid prediction commit")
	}
	return oid, nil
}

// treePaths lists the paths of tree. environment, when set, routes the read into a
// private store.
func (self *GitCommon) treePaths(tree string, environment func(*oscommands.CmdObj) *oscommands.CmdObj) ([]string, error) {
	cmd := self.cmd.New(NewGitCmd("ls-tree").Arg("-r", "-z", "--name-only", tree).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog()
	if environment != nil {
		cmd = environment(cmd)
	}
	stdout, _, err := cmd.RunWithOutputs()
	if err != nil {
		return nil, err
	}
	if stdout != "" && !strings.HasSuffix(stdout, "\x00") {
		return nil, fmt.Errorf("Git returned unterminated merge preview paths")
	}
	return nulPaths([]byte(stdout)), nil
}
