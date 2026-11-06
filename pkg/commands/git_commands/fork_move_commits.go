package git_commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Fork: g m n (move unpushed commits to a new branch) replaces upstream's
// reset-first flow. Upstream reset the branch before its destination existed,
// read HEAD and the upstream after its prompts, and stashed automatically.
//
// This file captures the checked-out branch, its upstream, the base branch of
// the base variant and the unpushed commits when the action starts. Every write
// is preceded by a fresh check of that capture and of the tracked files, and the
// writes are ordered so that the commits stay on a branch until the new branch
// holds them. Nothing is stashed, and no step assumes a rollback.

// MoveCommitsVariant selects where the new branch starts.
type MoveCommitsVariant int

const (
	// MoveCommitsStacked creates the new branch at the current commit and moves
	// the current branch back to its upstream.
	MoveCommitsStacked MoveCommitsVariant = iota
	// MoveCommitsFromBase creates the new branch off the base branch, cherry-picks
	// the unpushed commits onto it, and then moves the current branch back to its
	// upstream once the picks have all landed.
	MoveCommitsFromBase
)

var (
	// ErrMoveCommitsTrackedChanges means tracked files have staged or unstaged
	// changes. The flow refuses; it never stashes.
	ErrMoveCommitsTrackedChanges = errors.New("tracked files have changes")
	// ErrMoveCommitsChanged means the captured branch, upstream or base no longer
	// has the captured value.
	ErrMoveCommitsChanged = errors.New("the branch changed since the action started")
	// ErrMoveCommitsNameOption means the name would be read as a Git option.
	ErrMoveCommitsNameOption = errors.New("branch name starts with '-'")
	// ErrMoveCommitsNameExists means refs/heads/<name> already exists.
	ErrMoveCommitsNameExists = errors.New("branch already exists")
	// ErrMoveCommitsOldBranchChanged means the branch the commits leave no longer
	// has its captured tip.
	ErrMoveCommitsOldBranchChanged = errors.New("the branch the commits move from changed")
	// ErrMoveCommitsOldBranchInUse means another worktree or an operation holds
	// the branch the commits move from.
	ErrMoveCommitsOldBranchInUse = errors.New("the branch the commits move from is in use elsewhere")
	// ErrMoveCommitsIncomplete means the new branch does not hold one commit per
	// captured commit, for example because a pick was skipped.
	ErrMoveCommitsIncomplete = errors.New("the new branch does not hold every moved commit")
	// ErrMoveCommitsNoUpstream means the checked-out branch has no upstream.
	ErrMoveCommitsNoUpstream = errors.New("the branch has no upstream")
	// ErrMoveCommitsNotAhead means the upstream is not an ancestor of the branch,
	// so there is no unpushed commit range to move.
	ErrMoveCommitsNotAhead = errors.New("the branch is not ahead of its upstream")
	// ErrMoveCommitsNothingToMove means the upstream range is empty.
	ErrMoveCommitsNothingToMove = errors.New("there are no unpushed commits to move")
)

// MoveCommitsStep names the write that failed, so the flow can say what state
// the repository is left in.
type MoveCommitsStep int

const (
	// MoveCommitsStepCreate: the stacked variant's create-only ref write failed.
	// Nothing changed.
	MoveCommitsStepCreate MoveCommitsStep = iota
	// MoveCommitsStepCheckout: the stacked variant created the branch but could
	// not check it out. The current branch still holds the commits.
	MoveCommitsStepCheckout
	// MoveCommitsStepCreateFromBase: the base variant's guarded checkout failed.
	// Git creates nothing when it refuses.
	MoveCommitsStepCreateFromBase
	// MoveCommitsStepMoveOld: the current branch was not moved back to its
	// upstream. The new branch holds the commits.
	MoveCommitsStepMoveOld
)

// MoveCommitsStepError is a write that failed after the first write began.
type MoveCommitsStepError struct {
	Step  MoveCommitsStep
	Cause error
}

func (err *MoveCommitsStepError) Error() string {
	return fmt.Sprintf("move commits step %d failed: %v", err.Step, err.Cause)
}

func (err *MoveCommitsStepError) Unwrap() error { return err.Cause }

// MoveCommitsSource is the state captured when the action starts. It is the only
// source of values for the plan; live HEAD, upstream and model are never read
// again after capture.
type MoveCommitsSource struct {
	headRef, headOID         string
	upstreamRef, upstreamOID string
	baseRef, baseOID         string
	// commits are the unpushed commits, oldest first, with their parents.
	commits []*models.Commit
}

func (src MoveCommitsSource) HeadRef() string     { return src.headRef }
func (src MoveCommitsSource) HeadOID() string     { return src.headOID }
func (src MoveCommitsSource) UpstreamRef() string { return src.upstreamRef }
func (src MoveCommitsSource) UpstreamOID() string { return src.upstreamOID }
func (src MoveCommitsSource) BaseRef() string     { return src.baseRef }
func (src MoveCommitsSource) BaseOID() string     { return src.baseOID }
func (src MoveCommitsSource) CommitCount() int    { return len(src.commits) }

// MoveCommitsPlan is a source with a validated new name. Only
// PrepareMoveCommits marks a plan prepared.
type MoveCommitsPlan struct {
	variant  MoveCommitsVariant
	name     string
	src      MoveCommitsSource
	prepared bool
}

func (plan MoveCommitsPlan) Name() string                { return plan.name }
func (plan MoveCommitsPlan) Variant() MoveCommitsVariant { return plan.variant }
func (plan MoveCommitsPlan) Source() MoveCommitsSource   { return plan.src }

// CherryPickCommits returns the captured commits newest first, because
// Rebase.CherryPickCommits reverses its input before it runs cherry-pick.
func (plan MoveCommitsPlan) CherryPickCommits() []*models.Commit {
	return lo.Reverse(plan.src.commits)
}

// CaptureMoveCommitsSource reads the checked-out branch, its upstream and the
// unpushed commits in upstream..HEAD. It refuses an in-progress operation and
// tracked changes, and performs no writes. For the base variant, the caller adds
// the base branch with WithMoveCommitsBase.
func (self *BranchCommands) CaptureMoveCommitsSource() (MoveCommitsSource, error) {
	if err := self.rejectWorktreeOperation("move commits"); err != nil {
		return MoveCommitsSource{}, err
	}
	dirty, err := self.hasTrackedChanges()
	if err != nil {
		return MoveCommitsSource{}, err
	}
	if dirty {
		return MoveCommitsSource{}, ErrMoveCommitsTrackedChanges
	}

	headRef, headOID, err := self.namedCheckoutHead()
	if err != nil {
		return MoveCommitsSource{}, err
	}
	if headRef == "" {
		return MoveCommitsSource{}, fmt.Errorf("cannot move commits from a detached HEAD")
	}
	if headOID == "" {
		return MoveCommitsSource{}, fmt.Errorf("cannot move commits from an unborn branch")
	}

	// %(upstream) names the configured upstream by its full ref. The exact-match
	// filter drops the descendants that for-each-ref's pattern also returns.
	output, err := self.cmd.New(NewGitCmd("for-each-ref").
		Arg("--format=%(refname)%00%(upstream)", "--", headRef).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return MoveCommitsSource{}, fmt.Errorf("resolve upstream of %q: %w", headRef, err)
	}
	upstreamRef := ""
	for _, record := range strings.Split(trimTerminalLineEnding(output), "\n") {
		fields := strings.Split(record, "\x00")
		if len(fields) == 2 && fields[0] == headRef {
			upstreamRef = fields[1]
		}
	}
	if upstreamRef == "" {
		return MoveCommitsSource{}, ErrMoveCommitsNoUpstream
	}
	upstream, err := self.readPlacementRef(upstreamRef)
	if err != nil {
		return MoveCommitsSource{}, err
	}
	if !upstream.exists || upstream.symbolic != "" {
		return MoveCommitsSource{}, ErrMoveCommitsChanged
	}

	ancestor := self.cmd.New(NewGitCmd("merge-base").
		Arg("--is-ancestor", upstream.oid, headOID).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run()
	if ancestor != nil {
		if commandExitedWithOne(ancestor) {
			return MoveCommitsSource{}, ErrMoveCommitsNotAhead
		}
		return MoveCommitsSource{}, fmt.Errorf("compare %q with its upstream: %w", headRef, ancestor)
	}

	commits, err := self.unpushedCommits(upstream.oid, headOID)
	if err != nil {
		return MoveCommitsSource{}, err
	}
	if len(commits) == 0 {
		return MoveCommitsSource{}, ErrMoveCommitsNothingToMove
	}

	return MoveCommitsSource{
		headRef:     headRef,
		headOID:     headOID,
		upstreamRef: upstreamRef,
		upstreamOID: upstream.oid,
		commits:     commits,
	}, nil
}

// WithMoveCommitsBase adds the base branch of the base variant to a captured
// source. It resolves the ref once and captures its tip.
func (self *BranchCommands) WithMoveCommitsBase(src MoveCommitsSource, baseRef string) (MoveCommitsSource, error) {
	if baseRef == src.headRef {
		return MoveCommitsSource{}, fmt.Errorf("the base branch %q is the current branch", baseRef)
	}
	base, err := self.readPlacementRef(baseRef)
	if err != nil {
		return MoveCommitsSource{}, err
	}
	if !base.exists || base.symbolic != "" {
		return MoveCommitsSource{}, fmt.Errorf("base branch %q is not a direct ref: %w", baseRef, ErrMoveCommitsChanged)
	}
	src.baseRef = baseRef
	src.baseOID = base.oid
	return src, nil
}

// PrepareMoveCommits validates the literal new name, refuses an existing branch,
// and checks the captured state and tracked files again. It performs no writes.
func (self *BranchCommands) PrepareMoveCommits(variant MoveCommitsVariant, src MoveCommitsSource, name string) (MoveCommitsPlan, error) {
	if variant == MoveCommitsFromBase && src.baseRef == "" {
		return MoveCommitsPlan{}, fmt.Errorf("the base variant needs a captured base branch")
	}
	dirty, err := self.hasTrackedChanges()
	if err != nil {
		return MoveCommitsPlan{}, err
	}
	if dirty {
		return MoveCommitsPlan{}, ErrMoveCommitsTrackedChanges
	}
	if strings.HasPrefix(name, "-") {
		return MoveCommitsPlan{}, ErrMoveCommitsNameOption
	}
	if err := self.validatePlacementBranchName(name); err != nil {
		return MoveCommitsPlan{}, err
	}
	state, err := self.readPlacementRef("refs/heads/" + name)
	if err != nil {
		return MoveCommitsPlan{}, err
	}
	if state.exists {
		return MoveCommitsPlan{}, ErrMoveCommitsNameExists
	}
	if err := self.rejectWorktreeOperation("move commits"); err != nil {
		return MoveCommitsPlan{}, err
	}
	if err := self.checkMoveCommitsSource(src); err != nil {
		return MoveCommitsPlan{}, err
	}
	return MoveCommitsPlan{variant: variant, name: name, src: src, prepared: true}, nil
}

// ApplyMoveCommitsStacked runs the stacked variant's three writes. Each write is
// reported with its step, and a failure leaves the repository as the step's
// comment says.
//
//  1. Create refs/heads/<name> at the captured HEAD. The create-only update
//     fails if the name exists. Nothing has changed if this fails.
//  2. Check the new branch out with the guarded named-checkout flags. Its tree
//     is the captured tree, so no file changes. If this fails, the new branch
//     holds the commits and the current branch still does too.
//  3. Move the current branch to its captured upstream. The old-value check
//     refuses if it moved. If this fails, the new branch holds the commits.
func (self *BranchCommands) ApplyMoveCommitsStacked(plan MoveCommitsPlan) error {
	plan, err := self.recheckMoveCommits(plan, MoveCommitsStacked)
	if err != nil {
		return err
	}
	if err := self.cmd.New(NewGitCmd("update-ref").
		Arg("refs/heads/"+plan.name, plan.src.headOID, "").ToArgv()).
		DisableLockRetries().SetWd(self.repoPaths.WorktreePath()).Run(); err != nil {
		return &MoveCommitsStepError{Step: MoveCommitsStepCreate, Cause: err}
	}
	if err := self.checkoutMoveCommitsBranch(plan.name); err != nil {
		return &MoveCommitsStepError{Step: MoveCommitsStepCheckout, Cause: err}
	}
	if err := self.moveCommitsOldBranch(plan.src); err != nil {
		return &MoveCommitsStepError{Step: MoveCommitsStepMoveOld, Cause: err}
	}
	return nil
}

// ApplyMoveCommitsFromBaseCreate creates the base variant's new branch at the
// captured base and checks it out in one guarded command. Git creates nothing
// when that command refuses. The caller then cherry-picks the commits.
func (self *BranchCommands) ApplyMoveCommitsFromBaseCreate(plan MoveCommitsPlan) error {
	plan, err := self.recheckMoveCommits(plan, MoveCommitsFromBase)
	if err != nil {
		return err
	}
	args := NewGitCmd("checkout").
		Arg("--no-overwrite-ignore", "--no-guess", "--no-track", "-b", plan.name, plan.src.baseOID+"^{commit}").ToArgv()
	if err := self.cmd.New(args).DisableLockRetries().
		AddEnvVars("GIT_TERMINAL_PROMPT=0").
		SetWd(self.repoPaths.WorktreePath()).Run(); err != nil {
		return &MoveCommitsStepError{Step: MoveCommitsStepCreateFromBase, Cause: err}
	}
	return nil
}

// ApplyMoveCommitsFromBaseFinish moves the current branch back to its upstream
// once the picks have finished. It refuses unless the new branch holds exactly
// one commit per captured commit above the captured base, so a skipped pick
// leaves the commits on the current branch.
func (self *BranchCommands) ApplyMoveCommitsFromBaseFinish(plan MoveCommitsPlan) error {
	if !plan.prepared || plan.variant != MoveCommitsFromBase {
		return fmt.Errorf("move commits plan was not prepared for the base variant")
	}
	output, err := self.cmd.New(NewGitCmd("rev-list").
		Arg("--count", plan.src.baseOID+"..refs/heads/"+plan.name).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return fmt.Errorf("count picked commits on %q: %w", plan.name, err)
	}
	count, err := strconv.Atoi(trimTerminalLineEnding(output))
	if err != nil {
		return fmt.Errorf("invalid picked-commit count %q: %w", output, err)
	}
	if count != len(plan.src.commits) {
		return &MoveCommitsStepError{Step: MoveCommitsStepMoveOld, Cause: ErrMoveCommitsIncomplete}
	}
	if err := self.moveCommitsOldBranch(plan.src); err != nil {
		return &MoveCommitsStepError{Step: MoveCommitsStepMoveOld, Cause: err}
	}
	return nil
}

// recheckMoveCommits runs the full preparation again immediately before a first
// write. The plan must be the one PrepareMoveCommits returned for this variant.
func (self *BranchCommands) recheckMoveCommits(plan MoveCommitsPlan, variant MoveCommitsVariant) (MoveCommitsPlan, error) {
	if !plan.prepared || plan.variant != variant {
		return MoveCommitsPlan{}, fmt.Errorf("move commits plan was not prepared for this variant")
	}
	return self.PrepareMoveCommits(plan.variant, plan.src, plan.name)
}

func (self *BranchCommands) checkoutMoveCommitsBranch(name string) error {
	// The name is a branch, and "--" keeps Git from reading it as a path.
	args := NewGitCmd("checkout").Arg("--no-overwrite-ignore", "--no-guess", name, "--").ToArgv()
	return self.cmd.New(args).DisableLockRetries().
		AddEnvVars("GIT_TERMINAL_PROMPT=0").
		SetWd(self.repoPaths.WorktreePath()).Run()
}

// moveCommitsOldBranch moves the captured branch back to its captured upstream.
// The old-value check refuses if the branch moved after the capture, and the
// branch must not be checked out or reserved in another worktree.
func (self *BranchCommands) moveCommitsOldBranch(src MoveCommitsSource) error {
	state, err := self.readPlacementRef(src.headRef)
	if err != nil {
		return err
	}
	if !state.exists || state.symbolic != "" || state.oid != src.headOID {
		return ErrMoveCommitsOldBranchChanged
	}
	occupation, err := self.branchPlacementOccupancy(src.headRef)
	if err != nil {
		return err
	}
	if occupation != nil {
		return fmt.Errorf("%w: %s", ErrMoveCommitsOldBranchInUse, occupation.Kind)
	}
	return self.cmd.New(NewGitCmd("update-ref").
		Arg(src.headRef, src.upstreamOID, src.headOID).ToArgv()).
		DisableLockRetries().SetWd(self.repoPaths.WorktreePath()).Run()
}

// checkMoveCommitsSource reports ErrMoveCommitsChanged if the current branch, its
// upstream or the base branch no longer has the captured value.
func (self *BranchCommands) checkMoveCommitsSource(src MoveCommitsSource) error {
	ref, oid, err := self.namedCheckoutHead()
	if err != nil {
		return err
	}
	if ref != src.headRef || oid != src.headOID {
		return ErrMoveCommitsChanged
	}
	upstream, err := self.readPlacementRef(src.upstreamRef)
	if err != nil {
		return err
	}
	if !upstream.exists || upstream.symbolic != "" || upstream.oid != src.upstreamOID {
		return ErrMoveCommitsChanged
	}
	if src.baseRef != "" {
		base, err := self.readPlacementRef(src.baseRef)
		if err != nil {
			return err
		}
		if !base.exists || base.symbolic != "" || base.oid != src.baseOID {
			return ErrMoveCommitsChanged
		}
	}
	return nil
}

// hasTrackedChanges reports staged or unstaged changes to tracked files.
// Untracked files are not counted.
func (self *BranchCommands) hasTrackedChanges() (bool, error) {
	output, err := self.cmd.New(NewGitCmd("status").
		Arg("--porcelain=v1", "--untracked-files=no").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return false, fmt.Errorf("inspect tracked changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// unpushedCommits lists upstream..head, oldest first and parents first, with
// the parents Rebase.CherryPickCommits needs to recognise merges.
func (self *BranchCommands) unpushedCommits(upstreamOID, headOID string) ([]*models.Commit, error) {
	output, err := self.cmd.New(NewGitCmd("rev-list").
		Arg("--reverse", "--topo-order", "--parents", upstreamOID+".."+headOID).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return nil, fmt.Errorf("list unpushed commits: %w", err)
	}
	pool := &utils.StringPool{}
	var commits []*models.Commit
	for _, line := range strings.Split(trimTerminalLineEnding(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if err := validateObjectID(fields[0]); err != nil {
			return nil, fmt.Errorf("malformed commit ID in unpushed list: %w", err)
		}
		commits = append(commits, models.NewCommit(pool, models.NewCommitOpts{
			Hash:    strings.ToLower(fields[0]),
			Parents: fields[1:],
		}))
	}
	return commits, nil
}
