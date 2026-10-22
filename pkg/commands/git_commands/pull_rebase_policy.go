package git_commands

import "fmt"

type pullRebasePolicy struct {
	rebase bool
}

func (self *SyncCommands) effectivePullRebasePolicy(opts PullOptions) (pullRebasePolicy, error) {
	contextual := func(args ...string) *GitCommandBuilder {
		return NewGitCmd(args[0]).Arg(args[1:]...).
			GitDirIf(opts.WorktreeGitDir != "", opts.WorktreeGitDir).
			WorktreePathIf(opts.WorktreePath != "", opts.WorktreePath)
	}

	branch, err := self.cmd.New(contextual("symbolic-ref", "--quiet", "--short", "HEAD").ToArgv()).DontLog().RunWithOutput()
	if err == nil {
		branch = trimTerminalLineEnding(branch)
		branchPolicy, found, branchErr := self.pullRebaseConfig(contextual("config", "--type=bool-or-str", "--get", "branch."+branch+".rebase"))
		if branchErr != nil {
			return pullRebasePolicy{}, branchErr
		}
		if found {
			return branchPolicy, nil
		}
	} else if !commandExitedWithOne(err) {
		return pullRebasePolicy{}, err
	}
	policy, _, err := self.pullRebaseConfig(contextual("config", "--type=bool-or-str", "--get", "pull.rebase"))
	return policy, err
}

func (self *SyncCommands) pullRebaseConfig(command *GitCommandBuilder) (pullRebasePolicy, bool, error) {
	value, err := self.cmd.New(command.ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		if commandExitedWithOne(err) {
			return pullRebasePolicy{}, false, nil
		}
		return pullRebasePolicy{}, false, err
	}
	value = trimTerminalLineEnding(value)
	switch value {
	case "true":
		return pullRebasePolicy{rebase: true}, true, nil
	case "false", "":
		return pullRebasePolicy{}, true, nil
	case "merges", "m", "interactive", "i":
		return pullRebasePolicy{rebase: true}, true, nil
	case "preserve", "p":
		// Git 2.32 (our minimum version) still accepts these legacy modes.
		// Apply the same safety policy and leave version-specific validation
		// to the actual pull, without translating its requested topology.
		return pullRebasePolicy{rebase: true}, true, nil
	default:
		return pullRebasePolicy{}, true, fmt.Errorf("unsupported pull rebase mode %q", value)
	}
}
