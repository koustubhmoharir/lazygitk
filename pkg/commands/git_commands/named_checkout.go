package git_commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NamedCheckoutPlan is an immutable snapshot of a literal branch destination,
// target commit, and current HEAD state. Only PrepareNamedCheckout can mark a
// plan prepared.
type NamedCheckoutPlan struct {
	name, targetOID, oldOID, headRef, headOID string
	prepared                                  bool
}

func (plan NamedCheckoutPlan) Name() string      { return plan.name }
func (plan NamedCheckoutPlan) TargetOID() string { return plan.targetOID }
func (plan NamedCheckoutPlan) OldOID() string    { return plan.oldOID }
func (plan NamedCheckoutPlan) HeadRef() string   { return plan.headRef }
func (plan NamedCheckoutPlan) HeadOID() string   { return plan.headOID }

func (plan NamedCheckoutPlan) DestinationExists() bool { return plan.oldOID != "" }
func (plan NamedCheckoutPlan) MovesBranch() bool {
	return plan.oldOID != "" && plan.oldOID != plan.targetOID
}

var ErrNamedCheckoutChanged = errors.New("the checkout state changed; review it again before proceeding")

// NamedCheckoutError indicates that Git did not finish checkout cleanly. Git
// may already have changed HEAD, the branch, the index, or working files.
type NamedCheckoutError struct{ Cause error }

func (err *NamedCheckoutError) Error() string {
	return "Git checkout did not finish cleanly; HEAD, branch, index, or working files may already have changed: " + err.Cause.Error()
}

func (err *NamedCheckoutError) Unwrap() error { return err.Cause }

// PrepareNamedCheckout validates and snapshots a literal local branch name,
// exact commit OID, destination ref, and current HEAD. It performs no writes.
func (self *BranchCommands) PrepareNamedCheckout(name, targetOID string) (NamedCheckoutPlan, error) {
	if err := self.validatePlacementBranchName(name); err != nil {
		return NamedCheckoutPlan{}, err
	}
	if err := self.validateLiteralCommitOID(targetOID); err != nil {
		return NamedCheckoutPlan{}, err
	}
	bare, err := self.namedCheckoutContextIsBare()
	if err != nil {
		return NamedCheckoutPlan{}, err
	}
	if bare {
		return NamedCheckoutPlan{}, fmt.Errorf("cannot check out a branch in a bare repository")
	}
	if err := self.rejectNamedCheckoutOperation(); err != nil {
		return NamedCheckoutPlan{}, err
	}

	plan := NamedCheckoutPlan{name: name, targetOID: strings.ToLower(targetOID)}
	state, err := self.readPlacementRef("refs/heads/" + name)
	if err != nil {
		return NamedCheckoutPlan{}, err
	}
	if state.symbolic != "" {
		return NamedCheckoutPlan{}, fmt.Errorf("branch %q is a symbolic alias; choose a direct local branch", name)
	}
	plan.oldOID = state.oid

	plan.headRef, plan.headOID, err = self.namedCheckoutHead()
	if err != nil {
		return NamedCheckoutPlan{}, err
	}
	occupation, err := self.branchOccupancy("refs/heads/"+name, true)
	if err != nil {
		return NamedCheckoutPlan{}, err
	}
	if occupation != nil {
		if occupation.Current {
			return NamedCheckoutPlan{}, fmt.Errorf("branch %q has an active %s reservation in the current worktree", name, occupation.Kind)
		}
		return NamedCheckoutPlan{}, fmt.Errorf("branch %q is reserved by another worktree (%s, Git directory %q)", name, occupation.Kind, occupation.GitDir)
	}
	plan.prepared = true
	return plan, nil
}

// ApplyNamedCheckout revalidates the exact prepared snapshot immediately
// before Git's combined checkout operation. Git's own concurrency model applies
// while checkout runs; this API does not promise expected-old ref atomicity.
func (self *BranchCommands) ApplyNamedCheckout(plan NamedCheckoutPlan) error {
	if !plan.prepared {
		return fmt.Errorf("named checkout plan was not prepared")
	}
	fresh, err := self.PrepareNamedCheckout(plan.name, plan.targetOID)
	if err != nil {
		return err
	}
	if fresh != plan {
		return ErrNamedCheckoutChanged
	}
	args := NewGitCmd("checkout").Arg("--no-overwrite-ignore", "--no-guess", "-B", plan.name, plan.targetOID+"^{commit}").ToArgv()
	err = self.cmd.New(args).
		DisableLockRetries().
		AddEnvVars("GIT_TERMINAL_PROMPT=0").
		SetWd(self.repoPaths.WorktreePath()).Run()
	if err != nil {
		return &NamedCheckoutError{Cause: err}
	}
	return nil
}

func (self *BranchCommands) namedCheckoutContextIsBare() (bool, error) {
	output, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--is-bare-repository").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return false, fmt.Errorf("query current repository context: %w", err)
	}
	switch trimTerminalLineEnding(output) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bare-repository response %q", output)
	}
}

func (self *BranchCommands) namedCheckoutHead() (ref, oid string, err error) {
	output, symbolicErr := self.cmd.New(NewGitCmd("symbolic-ref").
		Arg("--quiet", "--no-recurse", "HEAD").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if symbolicErr == nil {
		ref = trimTerminalLineEnding(output)
		if !strings.HasPrefix(ref, "refs/heads/") || ref == "refs/heads/" {
			return "", "", fmt.Errorf("HEAD has an unsupported symbolic target %q", ref)
		}
	} else if !commandExitedWithOne(symbolicErr) {
		return "", "", fmt.Errorf("query current symbolic HEAD: %w", symbolicErr)
	}

	output, err = self.cmd.New(NewGitCmd("rev-parse").
		Arg("--verify", "--quiet", "HEAD").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err == nil {
		oid = trimTerminalLineEnding(output)
		if err := validateObjectID(oid); err != nil {
			return "", "", fmt.Errorf("malformed current HEAD object ID: %w", err)
		}
		return ref, strings.ToLower(oid), nil
	}
	if ref == "" || !commandExitedWithOne(err) {
		return "", "", fmt.Errorf("resolve current HEAD: %w", err)
	}
	state, refErr := self.readPlacementRef(ref)
	if refErr != nil {
		return "", "", fmt.Errorf("inspect unborn HEAD branch: %w", refErr)
	}
	if state.exists {
		return "", "", fmt.Errorf("resolve current HEAD on %q: %w", ref, err)
	}
	return ref, "", nil
}

func (self *BranchCommands) rejectNamedCheckoutOperation() error {
	gitDir := self.repoPaths.WorktreeGitDirPath()
	markers := []struct{ name, operation string }{
		{"rebase-merge", "rebase"},
		{"rebase-apply", "rebase"},
		{"MERGE_HEAD", "merge"},
		{"CHERRY_PICK_HEAD", "cherry-pick"},
		{"REVERT_HEAD", "revert"},
		{"sequencer", "sequencer operation"},
		{"BISECT_START", "bisect"},
	}
	for _, marker := range markers {
		path := filepath.Join(gitDir, marker.name)
		_, err := os.Lstat(path)
		if err == nil {
			return fmt.Errorf("cannot check out a named branch while a %s is in progress; finish or abort it first", marker.operation)
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect %s operation state: %w", marker.name, err)
		}
	}
	return nil
}
