package git_commands

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// Fork: upstream runs `git mergetool` with no paths, which walks every
// conflicted file and lets Git guess a tool when merge.tool is unset. The
// fork's merge tool launch names the selected files, asks Git to skip its
// per-file prompt and, unless the user set mergetool.keepBackup, stops Git
// from leaving .orig backups in the worktree. See
// docs-master/fork/MergeGuide.md ("Difftool launches").

// MergeToolCmdObj returns `git mergetool --no-prompt -- <paths>` for the given
// paths, read literally. It adds `-c mergetool.keepBackup=false` only when that
// key is absent from every config file, so an explicit setting, even an empty
// one, wins. The key is read on each call, not cached, so a change made while
// lazygitk runs applies to the next launch. Paths are required: with none, Git
// would merge every conflicted file.
func (self *WorkingTreeCommands) MergeToolCmdObj(paths []string) (*oscommands.CmdObj, error) {
	if len(paths) == 0 {
		return nil, errors.New("git mergetool needs at least one path")
	}
	keepBackupUnset, err := self.mergeToolKeepBackupUnset()
	if err != nil {
		return nil, err
	}
	args := NewGitCmd("--literal-pathspecs").
		Arg("mergetool", "--no-prompt", "--").
		Arg(paths...)
	if keepBackupUnset {
		args.Config("mergetool.keepBackup=false")
	}
	return self.cmd.New(args.ToArgv()), nil
}

// mergeToolKeepBackupUnset reports whether git config has no entry for
// mergetool.keepBackup. git config --get exits 1 for a missing key, and an
// empty value still counts as set.
func (self *WorkingTreeCommands) mergeToolKeepBackupUnset() (bool, error) {
	_, err := self.cmd.New(NewGitCmd("config").Arg("--get", "mergetool.keepBackup").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if commandExitedWithOne(err) {
		return true, nil
	}
	return false, err
}
