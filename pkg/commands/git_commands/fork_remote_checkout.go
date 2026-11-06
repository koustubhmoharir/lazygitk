package git_commands

import (
	"errors"
	"fmt"
	"strings"
)

// Fork: s c on a remote branch that has no local branch of the same name. Git
// creates the tracking branch and checks it out in one command, so a checkout
// that Git refuses or fails leaves no branch behind. The start point is the
// remote-tracking ref by its full name: the short origin/topic form is ambiguous
// with a tag of the same name, and the full ref is what sets the upstream.

// ErrRemoteBranchChanged means the remote branch no longer names the commit that
// was captured when the action started.
var ErrRemoteBranchChanged = errors.New("the remote branch changed after it was selected; review it again before checking it out")

// RemoteBranchCheckoutPlan is the new local branch, the remote-tracking ref and
// the commit captured for them. Only PrepareRemoteBranchCheckout marks a plan
// prepared.
type RemoteBranchCheckoutPlan struct {
	name, remoteRef, targetOID string
	prepared                   bool
}

// PrepareRemoteBranchCheckout runs the same checks as named checkout, then
// requires that the remote branch still names targetOID. It performs no writes.
func (self *BranchCommands) PrepareRemoteBranchCheckout(name, remoteRef, targetOID string) (RemoteBranchCheckoutPlan, error) {
	if !strings.HasPrefix(remoteRef, "refs/remotes/") || remoteRef == "refs/remotes/" {
		return RemoteBranchCheckoutPlan{}, fmt.Errorf("remote branch checkout needs a full remote-tracking ref, not %q", remoteRef)
	}
	named, err := self.PrepareNamedCheckout(name, targetOID)
	if err != nil {
		return RemoteBranchCheckoutPlan{}, err
	}
	if named.DestinationExists() {
		return RemoteBranchCheckoutPlan{}, fmt.Errorf("branch %q already exists; check it out from the Branches list", name)
	}
	remote, err := self.readPlacementRef(remoteRef)
	if err != nil {
		return RemoteBranchCheckoutPlan{}, err
	}
	if !remote.exists || remote.symbolic != "" || remote.oid != strings.ToLower(targetOID) {
		return RemoteBranchCheckoutPlan{}, ErrRemoteBranchChanged
	}
	return RemoteBranchCheckoutPlan{name: name, remoteRef: remoteRef, targetOID: strings.ToLower(targetOID), prepared: true}, nil
}

// ApplyRemoteBranchCheckout revalidates the plan immediately before the single
// checkout command, which creates the branch with tracking and switches to it.
// A refusal by Git returns Git's own error, and no branch is created.
func (self *BranchCommands) ApplyRemoteBranchCheckout(plan RemoteBranchCheckoutPlan) error {
	if !plan.prepared {
		return fmt.Errorf("remote branch checkout plan was not prepared")
	}
	if _, err := self.PrepareRemoteBranchCheckout(plan.name, plan.remoteRef, plan.targetOID); err != nil {
		return err
	}
	args := NewGitCmd("checkout").
		Arg("--no-overwrite-ignore", "--no-guess", "-b", plan.name, "--track", plan.remoteRef).ToArgv()
	return self.cmd.New(args).
		DisableLockRetries().
		AddEnvVars("GIT_TERMINAL_PROMPT=0").
		SetWd(self.repoPaths.WorktreePath()).Run()
}
