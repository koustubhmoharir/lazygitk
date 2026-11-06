package git_commands

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

// ErrRebaseRiskAfterCommit is returned by a compound flow whose rebase is refused
// after its commit was already created. The commit stays where it was made; the
// flow never retries the commit, so the user places it by hand.
var ErrRebaseRiskAfterCommit = errors.New("the new commit was created but was not moved: its rebase would now overwrite ignored files")

// IgnoredRebaseRiskError reports that a generic history-editing rebase start would
// let Git overwrite ignored files. Nothing has been written. The caller shows Paths
// and, to proceed, retries the same start under ApproveIgnoredRebaseRisks with
// Fingerprint.
type IgnoredRebaseRiskError struct {
	Paths       []string
	Fingerprint string
}

func (err *IgnoredRebaseRiskError) Error() string {
	return ErrRebaseIgnoredApprovalRequired.Error()
}

func (err *IgnoredRebaseRiskError) Unwrap() error {
	return ErrRebaseIgnoredApprovalRequired
}

// ApproveIgnoredRebaseRisks runs run with the ignored-file risks identified by
// fingerprint approved. Only rebase starts inside run see the approval, and it ends
// when run returns. A start whose risks differ from fingerprint is refused again
// with fresh paths, so the caller reviews the changed list.
//
// The scope is shared by every start on this RebaseCommands, and the mutex guards
// only the field, not the scope. Two overlapping scopes can therefore see each
// other's approval. That is harmless: a start still compares its fresh fingerprint
// with the approved one, so it proceeds only over the risks that were reviewed.
func (self *RebaseCommands) ApproveIgnoredRebaseRisks(fingerprint string, run func() error) error {
	self.ignoredRebaseApprovalMu.Lock()
	previous := self.ignoredRebaseApproval
	self.ignoredRebaseApproval = fingerprint
	self.ignoredRebaseApprovalMu.Unlock()
	defer func() {
		self.ignoredRebaseApprovalMu.Lock()
		self.ignoredRebaseApproval = previous
		self.ignoredRebaseApprovalMu.Unlock()
	}()
	return run()
}

// currentIgnoredRebaseApproval reads the scoped approval under the mutex.
func (self *RebaseCommands) currentIgnoredRebaseApproval() string {
	self.ignoredRebaseApprovalMu.Lock()
	defer self.ignoredRebaseApprovalMu.Unlock()
	return self.ignoredRebaseApproval
}

// requireIgnoredRebaseStart is the central guard of PrepareInteractiveRebaseCommand.
// Git's initial rewind checks out the destination, and replays the range. Either can
// overwrite ignored files, which a clean tracked status does not show.
func (self *RebaseCommands) requireIgnoredRebaseStart(opts PrepareInteractiveRebaseCommandOpts) error {
	destination := opts.baseHashOrRoot
	if opts.onto != "" {
		destination = opts.onto
	} else if opts.baseHashOrRoot == "--root" {
		// A root rebase without --onto rewinds to an empty tree.
		destination = ""
	}
	paths, fingerprint, err := self.rebaseStartIgnoredRisks(destination, opts.baseHashOrRoot, nil)
	if err != nil {
		return err
	}
	return requireIgnoredRebaseApproval(paths, fingerprint, opts.approvedIgnoredFingerprint, self.currentIgnoredRebaseApproval())
}

// requireIgnoredRebaseSquashAbove is the guard of SquashAllAboveFixupCommits, which
// builds its own command. It rebases HEAD's range above baseHashOrRoot in place,
// so the destination is the base itself, or an empty tree for a root rebase.
func (self *RebaseCommands) requireIgnoredRebaseSquashAbove(baseHashOrRoot string) error {
	destination := baseHashOrRoot
	if baseHashOrRoot == "--root" {
		destination = ""
	}
	paths, fingerprint, err := self.rebaseStartIgnoredRisks(destination, baseHashOrRoot, nil)
	if err != nil {
		return err
	}
	return requireIgnoredRebaseApproval(paths, fingerprint, "", self.currentIgnoredRebaseApproval())
}

// CheckCompoundRebaseIgnoredRisks is the preflight for a flow that writes a commit
// before its rebase starts, such as AmendTo and stacked fixup placement. It runs
// before that first write. commitIndex is the commit the rebase edits, as in
// AmendTo and MoveFixupCommitDown. When commitStagedChanges is set, the new commit
// holds the staged changes, which the rebase then replays.
func (self *RebaseCommands) CheckCompoundRebaseIgnoredRisks(commits []*models.Commit, commitIndex int, commitStagedChanges bool) error {
	baseHashOrRoot := getBaseHashOrRoot(commits, commitIndex+1)
	var extra []string
	if commitStagedChanges {
		staged, err := self.selectedRebaseGitOutput(NewGitCmd("diff").Arg("--cached", "--name-only", "--no-renames", "-z", "HEAD", "--").ToArgv())
		if err != nil {
			return fmt.Errorf("list staged paths before rebase: %w", err)
		}
		extra = nulPaths(staged)
	}
	destination := baseHashOrRoot
	if baseHashOrRoot == "--root" {
		destination = ""
	}
	paths, fingerprint, err := self.rebaseStartIgnoredRisks(destination, baseHashOrRoot, extra)
	if err != nil {
		return err
	}
	return requireIgnoredRebaseApproval(paths, fingerprint, "", self.currentIgnoredRebaseApproval())
}

// rebaseStartIgnoredRisks lists the risks of rebasing HEAD's range above
// baseHashOrRoot, onto destination. An empty destination means an empty tree.
func (self *RebaseCommands) rebaseStartIgnoredRisks(destination, baseHashOrRoot string, extra []string) ([]string, string, error) {
	logRevs := []string{"HEAD^{commit}"}
	if baseHashOrRoot != "--root" {
		logRevs = []string{baseHashOrRoot + "^{commit}" + ".." + "HEAD^{commit}"}
	}
	return self.rebaseCandidateRisks(destination, logRevs, extra)
}

// requireIgnoredRebaseApproval accepts risks only when the approval fingerprint
// matches the fresh one. The explicit approval, if any, takes precedence over the
// scoped one.
func requireIgnoredRebaseApproval(paths []string, fingerprint, explicit, scoped string) error {
	if len(paths) == 0 {
		return nil
	}
	approved := explicit
	if approved == "" {
		approved = scoped
	}
	if approved == fingerprint {
		return nil
	}
	return &IgnoredRebaseRiskError{Paths: paths, Fingerprint: fingerprint}
}

// RebaseRiskAfterCommit converts a risk refusal that comes after a commit was made.
func RebaseRiskAfterCommit(err error) error {
	var riskErr *IgnoredRebaseRiskError
	if errors.As(err, &riskErr) {
		return ErrRebaseRiskAfterCommit
	}
	return err
}
