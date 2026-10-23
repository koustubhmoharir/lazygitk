package git_commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

type BranchActionMergeMode uint8

const (
	BranchActionMergeConfigured BranchActionMergeMode = iota
	BranchActionMergeAllowFF
	BranchActionMergeNoFF
	BranchActionMergeSquash
	BranchActionMergeSquashAndCommit
)

// BranchActionMergePlan approves only the first native merge invocation.
// Squash-and-commit needs a separately validated post-merge commit phase; a
// conflict or a successful no-op must never be permission to commit user data.
type BranchActionMergePlan struct {
	review                          BranchActionMergeReview
	mode                            BranchActionMergeMode
	policy                          branchActionMergePolicy
	modeArguments, predictedPaths   []string
	ignoredPaths                    []string
	ignoredFingerprint              string
	defaultMessage                  string
	allIgnoredAtRisk                bool
	ignoredApproved, stagedApproved bool
	prepared                        bool
}

var (
	ErrMergeIgnoredApprovalRequired = errors.New("ignored files may be overwritten by this merge; review and explicitly approve them first")
	ErrMergeStagedApprovalRequired  = errors.New("already-staged changes would enter the squash commit; review and explicitly approve their inclusion first")
)

func (plan BranchActionMergePlan) Review() BranchActionMergeReview { return plan.review }
func (plan BranchActionMergePlan) Mode() BranchActionMergeMode     { return plan.mode }
func (plan BranchActionMergePlan) IgnoredFiles() []string          { return slices.Clone(plan.ignoredPaths) }
func (plan BranchActionMergePlan) ConservativeIgnoredReview() bool {
	return plan.allIgnoredAtRisk && len(plan.ignoredPaths) > 0
}

func (plan BranchActionMergePlan) WithIgnoredFilesApproved() BranchActionMergePlan {
	plan.ignoredApproved = true
	return plan
}

func (plan BranchActionMergePlan) WithStagedFilesApproved() BranchActionMergePlan {
	plan.stagedApproved = true
	return plan
}

// PrepareBranchActionMerge invokes native merge drivers only for an explicit
// merge review which needs an ort path prediction. Revalidation caches that
// prediction instead of running potentially interactive drivers again.
func (self *BranchCommands) PrepareBranchActionMerge(review BranchActionMergeReview, mode BranchActionMergeMode, defaultMessage string) (BranchActionMergePlan, error) {
	if err := self.CheckBranchActionMergeReview(review); err != nil {
		return BranchActionMergePlan{}, err
	}
	plan := BranchActionMergePlan{review: review, mode: mode, defaultMessage: defaultMessage}
	switch mode {
	case BranchActionMergeConfigured:
	case BranchActionMergeAllowFF:
		plan.modeArguments = []string{"--ff"}
	case BranchActionMergeNoFF:
		plan.modeArguments = []string{"--no-ff"}
	case BranchActionMergeSquash, BranchActionMergeSquashAndCommit:
		plan.modeArguments = []string{"--squash", "--ff", "--no-commit"}
	default:
		return BranchActionMergePlan{}, fmt.Errorf("unknown merge mode")
	}
	modeRecords, err := validateBranchActionMergeArguments(plan.modeArguments)
	if err != nil {
		return BranchActionMergePlan{}, err
	}
	records := append(slices.Clone(review.records), modeRecords...)
	plan.policy = resolveBranchActionMergePolicy(review.defaultFF, review.defaultStrategies, records)
	// Native ours/subtree strategies forbid fast-forward even when --ff was
	// supplied. Other/custom strategies must retain their configured behavior.
	for _, strategy := range plan.policy.strategies {
		if strategy == "ours" || strategy == "subtree" {
			plan.policy.ff = "false"
		}
	}
	if plan.policy.squash && plan.policy.ff == "false" {
		return BranchActionMergePlan{}, fmt.Errorf("Git cannot combine squash with no-fast-forward")
	}
	if plan.policy.invalidCombination {
		return BranchActionMergePlan{}, fmt.Errorf("Git cannot combine squash with an explicit --commit")
	}
	if review.target.HeadOID() == "" && (plan.policy.squash || plan.policy.ff == "false") {
		return BranchActionMergePlan{}, fmt.Errorf("Git supports only an ordinary fast-forward merge into an unborn branch")
	}
	if plan.policy.ff == "only" && !review.canFastForward && !review.alreadyMerged {
		return BranchActionMergePlan{}, fmt.Errorf("fast-forward-only cannot merge divergent history; choose a merge mode explicitly")
	}
	if err := self.checkBranchActionMergeReservation(plan); err != nil {
		return BranchActionMergePlan{}, err
	}
	if !review.alreadyMerged {
		ignored, err := self.branchActionMergeIgnoredPaths()
		if err != nil {
			return BranchActionMergePlan{}, err
		}
		if len(ignored) == 0 {
			// If ignored files appear before launch, require a new review. There
			// is no reason to run user drivers just to predict an empty risk set.
			plan.allIgnoredAtRisk = true
		} else if review.canFastForward && plan.policy.ff != "false" {
			plan.predictedPaths, err = self.branchActionTreePaths(review.target.SourceOID()+"^{tree}", nil)
		} else if len(plan.policy.strategies) == 0 || slices.Equal(plan.policy.strategies, []string{"ort"}) {
			plan.predictedPaths, err = self.PreviewBranchActionOrtMerge(review.target, BranchActionMergePreviewOptions{
				StrategyOptions: plan.policy.strategyOptions, AllowUnrelatedHistories: plan.policy.allowUnrelatedHistories,
			})
		} else {
			// Preserve configured strategies without pretending ort predicts
			// them. The UI must explain that all ignored files are potential
			// risks and obtain explicit approval, or let the user cancel.
			plan.allIgnoredAtRisk = true
		}
		if err != nil {
			return BranchActionMergePlan{}, err
		}
		plan.ignoredPaths, plan.ignoredFingerprint, err = self.branchActionMergeIgnoredRisk(plan)
		if err != nil {
			return BranchActionMergePlan{}, err
		}
	}
	if err := self.CheckBranchActionMergeReview(review); err != nil {
		return BranchActionMergePlan{}, err
	}
	plan.prepared = true
	return plan, nil
}

func (self *BranchCommands) checkBranchActionMergeReservation(plan BranchActionMergePlan) error {
	if plan.review.alreadyMerged {
		return nil
	}
	canMoveTip := plan.mode == BranchActionMergeSquashAndCommit || (!plan.policy.squash &&
		(plan.policy.commit || (plan.review.canFastForward && plan.policy.ff != "false")))
	if !canMoveTip {
		return nil
	}
	occupation, err := self.branchOtherWorktreeOccupancy(plan.review.target.HeadRef())
	if err != nil {
		return err
	}
	if occupation != nil {
		return fmt.Errorf("current branch is reserved by another worktree (%s, Git directory %q)", occupation.Kind, occupation.GitDir)
	}
	return nil
}

func (self *BranchCommands) branchActionMergeIgnoredPaths() ([]string, error) {
	stdout, _, err := self.cmd.New(NewGitCmd("ls-files").Arg("--others", "--ignored", "--exclude-standard", "-z").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return nil, err
	}
	if stdout != "" && !strings.HasSuffix(stdout, "\x00") {
		return nil, fmt.Errorf("Git returned unterminated ignored paths")
	}
	return nulPaths([]byte(stdout)), nil
}

func (self *BranchCommands) branchActionMergeIgnoredRisk(plan BranchActionMergePlan) ([]string, string, error) {
	ignored, err := self.branchActionMergeIgnoredPaths()
	if err != nil {
		return nil, "", err
	}
	ignoreCase, err := (&RebaseCommands{GitCommon: self.GitCommon}).selectedRebaseIgnoreCase()
	if err != nil {
		return nil, "", err
	}
	candidates := make(map[string]struct{}, len(plan.predictedPaths))
	paths := make([]string, 0, len(plan.predictedPaths))
	for _, path := range plan.predictedPaths {
		key := rebasePathKey(path, ignoreCase)
		candidates[key] = struct{}{}
		paths = append(paths, key)
	}
	sort.Strings(paths)
	risks := make([]string, 0, len(ignored))
	for _, path := range ignored {
		if plan.allIgnoredAtRisk || rebaseCandidatePathCollides(rebasePathKey(path, ignoreCase), candidates, paths) {
			risks = append(risks, path)
		}
	}
	sort.Strings(risks)
	risks = slices.Compact(risks)
	fingerprint, err := fingerprintIgnoredPaths(self.repoPaths.WorktreePath(), risks)
	return risks, fingerprint, err
}

func (self *BranchCommands) BranchActionMergeCommand(plan BranchActionMergePlan) (*oscommands.CmdObj, error) {
	if !plan.prepared {
		return nil, fmt.Errorf("merge plan was not prepared")
	}
	if err := self.CheckBranchActionMergeReview(plan.review); err != nil {
		return nil, err
	}
	if err := self.checkBranchActionMergeReservation(plan); err != nil {
		return nil, err
	}
	if !plan.review.alreadyMerged {
		paths, fingerprint, err := self.branchActionMergeIgnoredRisk(plan)
		if err != nil {
			return nil, err
		}
		if !slices.Equal(paths, plan.ignoredPaths) || fingerprint != plan.ignoredFingerprint {
			return nil, ErrBranchActionChanged
		}
		if len(paths) > 0 && !plan.ignoredApproved {
			return nil, ErrMergeIgnoredApprovalRequired
		}
		if plan.mode == BranchActionMergeSquashAndCommit && len(plan.review.stagedPaths) > 0 && !plan.stagedApproved {
			return nil, ErrMergeStagedApprovalRequired
		}
	}
	args := NewGitCmd("merge").Config("branch." + strings.TrimPrefix(plan.review.target.HeadRef(), "refs/heads/") + ".mergeOptions=").
		Config("merge.autoStash=false").Arg("--no-edit").Arg(plan.review.arguments...).Arg(plan.modeArguments...)
	if !plan.policy.customMessage && plan.defaultMessage != "" {
		args.Arg("--message=" + plan.defaultMessage)
	}
	args.Arg("--no-autostash").ArgIf(plan.ignoredApproved, "--overwrite-ignore").
		ArgIf(!plan.ignoredApproved, "--no-overwrite-ignore").Arg("--", plan.review.target.SourceOID()+"^{commit}")
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries(), nil
}

// CheckBranchActionMergeInProgress binds a queued abort to this operation's
// metadata. The mutable source ref may have moved after the merge started; its
// original commit, not its current tip, must match MERGE_HEAD.
func (self *BranchCommands) CheckBranchActionMergeInProgress(plan BranchActionMergePlan) error {
	if !plan.prepared || plan.review.target.owner != self.GitCommon || plan.policy.squash {
		return fmt.Errorf("a normal merge plan from this repository is required")
	}
	ref, oid, err := self.namedCheckoutHead()
	if err != nil {
		return err
	}
	if ref != plan.review.target.HeadRef() || oid != plan.review.target.HeadOID() {
		return ErrBranchActionChanged
	}
	for _, entry := range []struct{ name, expected string }{
		{"MERGE_HEAD", plan.review.target.SourceOID()},
		{"ORIG_HEAD", plan.review.target.HeadOID()},
	} {
		contents, err := os.ReadFile(filepath.Join(self.repoPaths.WorktreeGitDirPath(), entry.name))
		if os.IsNotExist(err) {
			return ErrBranchActionChanged
		}
		if err != nil {
			return err
		}
		if trimTerminalLineEnding(string(contents)) != entry.expected {
			return ErrBranchActionChanged
		}
	}
	return nil
}
