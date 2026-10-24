package git_commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// CherryPickCommit is the immutable identity and display summary of a commit
// reviewed for a captured cherry-pick. Parents are in Git's stored order.
type CherryPickCommit struct {
	OID     string
	Subject string
	Parents []string
}

// CherryPickPlan captures an ordered commit selection and the checkout state
// against which it was reviewed. A detached checkout has an empty HeadRef;
// an unborn symbolic branch has a HeadRef and empty HeadOID.
type CherryPickPlan struct {
	owner                     *GitCommon
	headRef, headOID          string
	commits                   []CherryPickCommit
	ignoredPaths              []string
	ignoredFingerprint        string
	mainline                  int
	ignoredApproved, prepared bool
}

func (plan CherryPickPlan) HeadRef() string { return plan.headRef }
func (plan CherryPickPlan) HeadOID() string { return plan.headOID }
func (plan CherryPickPlan) Commits() []CherryPickCommit {
	commits := make([]CherryPickCommit, len(plan.commits))
	for i, commit := range plan.commits {
		commits[i] = commit
		commits[i].Parents = slices.Clone(commit.Parents)
	}
	return commits
}
func (plan CherryPickPlan) IgnoredFiles() []string { return slices.Clone(plan.ignoredPaths) }
func (plan CherryPickPlan) Mainline() int          { return plan.mainline }

// WithMainline selects one parent number for every merge in this plan.
func (plan CherryPickPlan) WithMainline(n int) (CherryPickPlan, error) {
	minimum := 0
	for _, commit := range plan.commits {
		if len(commit.Parents) > 1 && (minimum == 0 || len(commit.Parents) < minimum) {
			minimum = len(commit.Parents)
		}
	}
	if minimum == 0 {
		if n != 0 {
			return CherryPickPlan{}, fmt.Errorf("mainline must be 0 when no selected commit is a merge")
		}
		plan.mainline = 0
		return plan, nil
	}
	if n < 1 || n > minimum {
		return CherryPickPlan{}, fmt.Errorf("mainline must be between 1 and %d for the selected merge commits", minimum)
	}
	plan.mainline = n
	return plan, nil
}

func (plan CherryPickPlan) WithIgnoredFilesApproved() CherryPickPlan {
	plan.ignoredApproved = true
	return plan
}

var ErrCherryPickMainlineRequired = errors.New("a mainline parent must be selected for cherry-picking merge commits")

// PrepareCherryPickPlan validates exact commit IDs and captures Git ancestry,
// the checkout, and ignored-file risks without modifying repository state.
func (self *RebaseCommands) PrepareCherryPickPlan(oids []string) (CherryPickPlan, error) {
	if len(oids) == 0 {
		return CherryPickPlan{}, fmt.Errorf("select at least one commit to cherry-pick")
	}
	branches := NewBranchCommands(self.GitCommon)
	seen := make(map[string]struct{}, len(oids))
	selected := make([]string, 0, len(oids))
	for _, oid := range oids {
		if err := branches.validateLiteralCommitOID(oid); err != nil {
			return CherryPickPlan{}, err
		}
		oid = strings.ToLower(oid)
		if _, exists := seen[oid]; !exists {
			seen[oid] = struct{}{}
			selected = append(selected, oid)
		}
	}
	// Git uses argument order to break ties between unrelated tips. Keep the
	// walk stable when revalidation supplies the already reviewed sequence.
	slices.Sort(selected)
	bare, err := branches.namedCheckoutContextIsBare()
	if err != nil {
		return CherryPickPlan{}, err
	}
	if bare {
		return CherryPickPlan{}, fmt.Errorf("cannot cherry-pick in a bare repository")
	}
	ref, head, err := branches.namedCheckoutHead()
	if err != nil {
		return CherryPickPlan{}, err
	}

	args := NewGitCmd("rev-list").Arg("--reverse", "--topo-order").Arg(selected...).Arg("--").ToArgv()
	output, diagnostic, err := self.cmd.New(args).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return CherryPickPlan{}, fmt.Errorf("order selected cherry-pick commits: %w: %s", err, strings.TrimSpace(diagnostic))
	}
	ordered := make([]string, 0, len(selected))
	for _, oid := range strings.Fields(output) {
		if _, ok := seen[strings.ToLower(oid)]; ok {
			ordered = append(ordered, strings.ToLower(oid))
		}
	}
	if len(ordered) != len(selected) {
		return CherryPickPlan{}, fmt.Errorf("Git did not return every selected commit in ancestry order")
	}

	plan := CherryPickPlan{owner: self.GitCommon, headRef: ref, headOID: head, prepared: true}
	candidates := make(map[string]struct{})
	for _, oid := range ordered {
		commit, err := self.capturedCherryPickCommit(oid)
		if err != nil {
			return CherryPickPlan{}, err
		}
		plan.commits = append(plan.commits, commit)
		pathArgs := NewGitCmd("diff-tree").Arg("--no-commit-id", "--root", "-r", "--no-renames", "--name-only", "-z")
		if len(commit.Parents) > 1 {
			pathArgs.Arg("-m")
		}
		pathArgs.Arg(oid, "--")
		paths, _, err := self.cmd.New(pathArgs.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
		if err != nil {
			return CherryPickPlan{}, fmt.Errorf("list paths changed by commit %s: %w", oid, err)
		}
		for _, path := range nulPaths([]byte(paths)) {
			candidates[path] = struct{}{}
		}
	}
	plan.ignoredPaths, plan.ignoredFingerprint, err = self.ignoredRisksForPaths(candidates)
	if err != nil {
		return CherryPickPlan{}, err
	}
	return plan, nil
}

func (self *RebaseCommands) capturedCherryPickCommit(oid string) (CherryPickCommit, error) {
	output, diagnostic, err := self.cmd.New(NewGitCmd("cat-file").Arg("commit", oid).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return CherryPickCommit{}, fmt.Errorf("read selected commit %s: %w: %s", oid, err, strings.TrimSpace(diagnostic))
	}
	commit := CherryPickCommit{OID: oid}
	lines := strings.Split(output, "\n")
	message := false
	for _, line := range lines {
		if message {
			commit.Subject = strings.TrimSuffix(line, "\r")
			break
		}
		if line == "" {
			message = true
		} else if strings.HasPrefix(line, "parent ") {
			parent := strings.TrimPrefix(line, "parent ")
			if err := validateObjectID(parent); err != nil {
				return CherryPickCommit{}, fmt.Errorf("commit %s has invalid parent: %w", oid, err)
			}
			commit.Parents = append(commit.Parents, parent)
		}
	}
	return commit, nil
}

// CherryPickCommand re-prepares the selection and returns one native Git
// command only when the reviewed checkout, sequence, and ignored risks match.
func (self *RebaseCommands) CherryPickCommand(plan CherryPickPlan) (*oscommands.CmdObj, error) {
	if !plan.prepared {
		return nil, fmt.Errorf("cherry-pick plan was not prepared by this repository")
	}
	if plan.owner != self.GitCommon {
		return nil, ErrBranchActionChanged
	}
	oids := make([]string, len(plan.commits))
	for i, commit := range plan.commits {
		oids[i] = commit.OID
	}
	fresh, err := self.PrepareCherryPickPlan(oids)
	if err != nil {
		return nil, err
	}
	if fresh.headRef != plan.headRef || fresh.headOID != plan.headOID ||
		!cherryPickCommitsEqual(fresh.commits, plan.commits) || !slices.Equal(fresh.ignoredPaths, plan.ignoredPaths) ||
		fresh.ignoredFingerprint != plan.ignoredFingerprint {
		return nil, ErrBranchActionChanged
	}
	hasMerge := false
	for _, commit := range fresh.commits {
		hasMerge = hasMerge || len(commit.Parents) > 1
	}
	if hasMerge && plan.mainline == 0 {
		return nil, ErrCherryPickMainlineRequired
	}
	if len(fresh.ignoredPaths) > 0 && !plan.ignoredApproved {
		return nil, ErrRebaseIgnoredApprovalRequired
	}
	args := NewGitCmd("cherry-pick").Arg("--allow-empty").ArgIf(self.version.IsAtLeast(2, 45, 0), "--empty=keep")
	if hasMerge {
		args.Arg(fmt.Sprintf("-m%d", plan.mainline))
	}
	for _, commit := range fresh.commits {
		args.Arg(commit.OID)
	}
	return self.cmd.New(args.ToArgv()).DisableLockRetries().SetWd(self.repoPaths.WorktreePath()), nil
}

func cherryPickCommitsEqual(left, right []CherryPickCommit) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].OID != right[i].OID || left[i].Subject != right[i].Subject || !slices.Equal(left[i].Parents, right[i].Parents) {
			return false
		}
	}
	return true
}
