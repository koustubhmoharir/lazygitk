package git_commands

import (
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// Fork: literal-pathspec variants of three upstream diff builders. Git reads the
// paths after `--` as patterns unless literal pathspecs are on, so a selected
// file named a*.txt also selects a2.txt, and a name starting with :(exclude)
// changes the selection. These variants are used by the Files and commit-file
// difftool launches (including `d s`) and copy-diff actions, which name exactly
// the file or directory they act on. A literal directory still matches
// everything under it.
//
// Each variant wraps the upstream builder and sets GIT_LITERAL_PATHSPECS=1,
// which is what --literal-pathspecs does and which also reaches the git diff
// that difftool runs. The upstream builders stay unchanged, so their other
// callers keep Git's pathspec interpretation, and upstream changes to them
// carry over. See docs-master/fork/MergeGuide.md.

const literalPathspecsEnv = "GIT_LITERAL_PATHSPECS=1"

// OpenLiteralDiffToolCmdObj is OpenDiffToolCmdObj with literal pathspecs.
func (self *DiffCommands) OpenLiteralDiffToolCmdObj(opts DiffToolCmdOptions) *oscommands.CmdObj {
	return self.OpenDiffToolCmdObj(opts).AddEnvVars(literalPathspecsEnv)
}

// GetLiteralPathsDiff is GetDiff for the given paths, which Git reads literally.
// It returns the working-tree diff, or the staged diff when staged is true.
func (self *DiffCommands) GetLiteralPathsDiff(staged bool, paths ...string) (string, error) {
	args := append([]string{"--"}, paths...)
	return self.getDiffCmdObj(self.repoPaths.worktreePath, staged, args).AddEnvVars(literalPathspecsEnv).RunWithOutput()
}

// ShowLiteralFileDiffCmdObj is ShowFileDiffCmdObj with literal pathspecs. The
// file names include the old path of a rename, which Git needs to pair the two
// ends of it.
func (self *WorkingTreeCommands) ShowLiteralFileDiffCmdObj(from string, to string, reverse bool, fileNames []string, mode DiffMode) *oscommands.CmdObj {
	return self.ShowFileDiffCmdObj(from, to, reverse, fileNames, mode).AddEnvVars(literalPathspecsEnv)
}
