package git_commands

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// BranchActionSquashCommitOptions is captured from the explicit message review
// and application settings. It must not be read from live GUI configuration in
// a queued post-merge callback.
type BranchActionSquashCommitOptions struct {
	Message   string
	SignOff   bool
	SkipHooks bool
}

type BranchActionSquashCommitPlan struct {
	review   BranchActionMergeReview
	options  BranchActionSquashCommitOptions
	args     []string
	prepared bool
}

// PrepareBranchActionSquashCommit is called only after the first native command
// succeeded. False means no new staged merge result: do not commit existing
// staged changes merely because Git reported a successful no-op squash.
func (self *BranchCommands) PrepareBranchActionSquashCommit(merge BranchActionMergePlan, options BranchActionSquashCommitOptions) (BranchActionSquashCommitPlan, bool, error) {
	if !merge.prepared || merge.mode != BranchActionMergeSquashAndCommit {
		return BranchActionSquashCommitPlan{}, false, fmt.Errorf("a squash-and-commit merge plan is required")
	}
	if err := self.CheckBranchActionTarget(merge.review.target); err != nil {
		return BranchActionSquashCommitPlan{}, false, err
	}
	if merge.review.alreadyMerged {
		return BranchActionSquashCommitPlan{}, false, nil
	}
	if len(merge.review.stagedPaths) > 0 && !merge.stagedApproved {
		return BranchActionSquashCommitPlan{}, false, ErrMergeStagedApprovalRequired
	}
	fresh, err := self.PrepareBranchActionMergeReview(merge.review.target, merge.review.applicationArguments)
	if err != nil {
		return BranchActionSquashCommitPlan{}, false, err
	}
	if fresh.configSignature != merge.review.configSignature {
		return BranchActionSquashCommitPlan{}, false, ErrBranchActionChanged
	}
	if err := self.checkBranchActionMergeReservation(merge); err != nil {
		return BranchActionSquashCommitPlan{}, false, err
	}
	if fresh.indexEntries == merge.review.indexEntries || len(fresh.stagedPaths) == 0 {
		return BranchActionSquashCommitPlan{}, false, nil
	}
	if strings.TrimSpace(options.Message) == "" || strings.ContainsRune(options.Message, 0) {
		return BranchActionSquashCommitPlan{}, false, fmt.Errorf("review a nonempty squash commit message")
	}
	plan := BranchActionSquashCommitPlan{review: fresh, options: options, prepared: true}
	// Retain explicitly configured signing, editor and hook choices in the
	// separate commit phase. The modal's reviewed message owns its text.
	for _, record := range merge.review.records {
		switch record.Name {
		case "--gpg-sign", "--cleanup":
			arg := record.Name
			if record.HasValue {
				arg += "=" + record.Value
			}
			plan.args = append(plan.args, arg)
		case "--no-gpg-sign", "--edit", "--no-edit", "--signoff", "--no-signoff", "--verify", "--no-verify", "--no-cleanup":
			plan.args = append(plan.args, record.Name)
		}
	}
	return plan, true, nil
}

func (self *BranchCommands) BranchActionSquashCommitCommand(plan BranchActionSquashCommitPlan) (*oscommands.CmdObj, error) {
	if !plan.prepared {
		return nil, fmt.Errorf("squash commit plan was not prepared")
	}
	if err := self.CheckBranchActionMergeReview(plan.review); err != nil {
		return nil, err
	}
	occupation, err := self.branchOtherWorktreeOccupancy(plan.review.target.HeadRef())
	if err != nil {
		return nil, err
	}
	if occupation != nil {
		return nil, fmt.Errorf("current branch is reserved by another worktree (%s, Git directory %q)", occupation.Kind, occupation.GitDir)
	}
	args := NewGitCmd("commit").Arg("--message="+plan.options.Message).
		ArgIf(plan.options.SignOff, "--signoff").ArgIf(plan.options.SkipHooks, "--no-verify").Arg(plan.args...)
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries(), nil
}
