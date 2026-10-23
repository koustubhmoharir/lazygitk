package git_commands

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// BranchActionCreateTracking selects Git's native branch tracking policy.
type BranchActionCreateTracking uint8

const (
	BranchActionCreateInvalid BranchActionCreateTracking = iota
	// NoTracking asks Git not to set up upstream tracking. Native Git can retain
	// orphan branch.<name>.remote/merge config that already exists.
	BranchActionCreateNoTracking
	BranchActionCreateConfiguredTracking
	BranchActionCreateDirectTracking
)

type BranchActionCreateOptions struct {
	Tracking BranchActionCreateTracking
	Checkout bool
}

// BranchActionCreatePlan is an opaque, repository-bound snapshot for native branch creation.
type BranchActionCreatePlan struct {
	owner             *GitCommon
	target            BranchActionTarget
	name              string
	options           BranchActionCreateOptions
	configSignature   string
	destinationAbsent bool
	prepared          bool
}

func (p BranchActionCreatePlan) Target() BranchActionTarget         { return p.target }
func (p BranchActionCreatePlan) Name() string                       { return p.name }
func (p BranchActionCreatePlan) Options() BranchActionCreateOptions { return p.options }

func (self *BranchCommands) PrepareBranchActionCreate(target BranchActionTarget, name string, options BranchActionCreateOptions) (BranchActionCreatePlan, error) {
	if options.Tracking < BranchActionCreateNoTracking || options.Tracking > BranchActionCreateDirectTracking {
		return BranchActionCreatePlan{}, fmt.Errorf("invalid branch tracking policy")
	}
	if err := self.validatePlacementBranchName(name); err != nil {
		return BranchActionCreatePlan{}, err
	}
	if err := self.CheckBranchActionTarget(target); err != nil {
		return BranchActionCreatePlan{}, err
	}
	if options.Tracking == BranchActionCreateDirectTracking && (IsFullObjectID(target.sourceRef) || strings.HasPrefix(target.sourceRef, "refs/tags/") || strings.HasPrefix(target.sourceRef, "refs/stash") || target.sourceRef == "HEAD") {
		return BranchActionCreatePlan{}, fmt.Errorf("direct tracking requires a local or remote branch source")
	}
	if options.Checkout {
		bare, err := self.namedCheckoutContextIsBare()
		if err != nil {
			return BranchActionCreatePlan{}, err
		}
		if bare {
			return BranchActionCreatePlan{}, fmt.Errorf("cannot check out a branch in a bare repository")
		}
		if err := self.rejectNamedCheckoutOperation(); err != nil {
			return BranchActionCreatePlan{}, err
		}
	}
	state, err := self.readPlacementRef("refs/heads/" + name)
	if err != nil {
		return BranchActionCreatePlan{}, err
	}
	if state.exists || state.symbolic != "" {
		return BranchActionCreatePlan{}, fmt.Errorf("branch %q already exists", name)
	}
	if options.Checkout {
		occupation, err := self.branchOccupancy("refs/heads/"+name, true)
		if err != nil {
			return BranchActionCreatePlan{}, err
		}
		if occupation != nil {
			return BranchActionCreatePlan{}, fmt.Errorf("branch %q is reserved by a worktree", name)
		}
	}
	if !options.Checkout {
		occupation, err := self.branchPlacementOccupancy("refs/heads/" + name)
		if err != nil {
			return BranchActionCreatePlan{}, err
		}
		if occupation != nil {
			return BranchActionCreatePlan{}, fmt.Errorf("branch %q is reserved by a worktree or Git operation (%s)", name, occupation.Kind)
		}
	}
	config, err := self.deleteConfigSignature()
	if err != nil {
		return BranchActionCreatePlan{}, err
	}
	return BranchActionCreatePlan{owner: self.GitCommon, target: target, name: name, options: options, configSignature: config, destinationAbsent: !state.exists && state.symbolic == "", prepared: true}, nil
}

// BranchActionCreateCommand rechecks the snapshot and delegates tracking inference and creation to Git.
func (self *BranchCommands) BranchActionCreateCommand(plan BranchActionCreatePlan) (*oscommands.CmdObj, error) {
	if !plan.prepared || plan.owner != self.GitCommon {
		return nil, fmt.Errorf("branch creation plan was not prepared by this repository instance")
	}
	fresh, err := self.PrepareBranchActionCreate(plan.target, plan.name, plan.options)
	if err != nil {
		return nil, err
	}
	if fresh.configSignature != plan.configSignature || fresh.destinationAbsent != plan.destinationAbsent {
		return nil, ErrBranchActionChanged
	}
	start := plan.target.sourceRef
	if plan.options.Tracking == BranchActionCreateNoTracking || IsFullObjectID(start) {
		start = plan.target.sourceOID + "^{commit}"
	}
	args := NewGitCmd("branch")
	if plan.options.Tracking == BranchActionCreateNoTracking {
		args.Arg("--no-track")
	}
	if plan.options.Tracking == BranchActionCreateDirectTracking {
		args.Arg("--track")
	}
	args.Arg("--", plan.name, start)
	if plan.options.Checkout {
		checkout := NewGitCmd("checkout").Arg("--no-overwrite-ignore", "--no-guess")
		if plan.options.Tracking == BranchActionCreateNoTracking {
			checkout.Arg("--no-track")
		}
		if plan.options.Tracking == BranchActionCreateDirectTracking {
			checkout.Arg("--track")
		}
		args = checkout.Arg("-b", plan.name, start, "--")
	}
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries(), nil
}
