package git_commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/app/daemon"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// CombineCommand builds one native interactive rebase from a reviewed capture.
// Call cleanup after execution, including a native pause or subprocess failure.
// The sequence editor moves the approved message into Git-owned rebase state;
// continuation needs neither this temporary payload nor a running lazygit.
func (self *RebaseCommands) CombineCommand(plan CapturedCombinePlan, message string) (*oscommands.CmdObj, func(), error) {
	if err := self.CheckCombinePlan(plan); err != nil {
		return nil, nil, err
	}
	if strings.ContainsRune(message, '\x00') || strings.TrimSpace(message) == "" {
		return nil, nil, fmt.Errorf("the combined commit needs a nonempty message without NUL bytes")
	}
	payload := daemon.CombineRebasePayload{KeeperOID: plan.topology.KeeperOID(), Todo: plan.topology.Todo(), Message: message}
	file, err := os.CreateTemp("", "lazygit-combine-*.json")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		if err := os.Remove(file.Name()); err != nil && !os.IsNotExist(err) {
			self.Log.WithError(err).Warn("remove temporary combine payload")
		}
	}
	writeErr := json.NewEncoder(file).Encode(payload)
	closeErr := file.Close()
	if writeErr != nil {
		cleanup()
		return nil, nil, writeErr
	}
	if closeErr != nil {
		cleanup()
		return nil, nil, closeErr
	}
	args := NewGitCmd("rebase").Config("rebase.updateRefs=false").
		Arg("--interactive", "--no-autostash", "--keep-empty", "--empty=keep", "--reapply-cherry-picks", "--no-autosquash", "--no-rebase-merges", "--reschedule-failed-exec")
	if plan.topology.BaseOID() == "" {
		args.Arg("--root")
	} else {
		args.Arg(plan.topology.BaseOID() + "^{commit}")
	}
	// Fork combine replaces index-based squash/fixup with a captured linear TODO
	// and a persistent approved message. Git retains all conflict/continue state.
	cmd := self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries().
		AddEnvVars("GIT_SEQUENCE_EDITOR="+oscommands.GetLazygitPath(), "GIT_EDITOR="+oscommands.GetLazygitPath(), "GIT_TERMINAL_PROMPT=0").
		AddEnvVars(daemon.ToEnvVars(daemon.NewCombineRebaseInstruction(file.Name()))...)
	return cmd, cleanup, nil
}
