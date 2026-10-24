package git_commands

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

// CapturedCombinePlan binds a pure topology plan to the repository snapshot
// against which it was reviewed.
type CapturedCombinePlan struct {
	owner              *GitCommon
	headRef            string
	headOID            string
	topology           LinearCombinePlan
	ignoredPaths       []string
	ignoredFingerprint string
	ignoredApproved    bool
	prepared           bool
}

func (plan CapturedCombinePlan) Topology() LinearCombinePlan {
	plan.topology.selected = cloneCombineHistory(plan.topology.selected)
	plan.topology.history = cloneCombineHistory(plan.topology.history)
	return plan.topology
}
func (plan CapturedCombinePlan) HeadRef() string        { return plan.headRef }
func (plan CapturedCombinePlan) IgnoredFiles() []string { return slices.Clone(plan.ignoredPaths) }
func (plan CapturedCombinePlan) WithIgnoredFilesApproved() CapturedCombinePlan {
	plan.ignoredApproved = true
	return plan
}

// PrepareCombinePlan captures the current first-parent history and validates
// that all selected commits form one ancestry on that history.
func (self *RebaseCommands) PrepareCombinePlan(oids []string) (CapturedCombinePlan, error) {
	if len(oids) < 2 {
		return CapturedCombinePlan{}, fmt.Errorf("select at least two commits to combine")
	}
	branches := NewBranchCommands(self.GitCommon)
	selected := make([]string, 0, len(oids))
	seen := make(map[string]struct{}, len(oids))
	for _, oid := range oids {
		if err := branches.validateLiteralCommitOID(oid); err != nil {
			return CapturedCombinePlan{}, err
		}
		oid = strings.ToLower(oid)
		if _, exists := seen[oid]; exists {
			return CapturedCombinePlan{}, fmt.Errorf("selected commit %q is duplicated", oid)
		}
		seen[oid] = struct{}{}
		selected = append(selected, oid)
	}
	bare, err := branches.namedCheckoutContextIsBare()
	if err != nil {
		return CapturedCombinePlan{}, err
	}
	if bare {
		return CapturedCombinePlan{}, fmt.Errorf("cannot combine commits in a bare repository")
	}
	if err := self.rejectWorktreeOperation("start a combine"); err != nil {
		return CapturedCombinePlan{}, err
	}
	status := self.cmd.New(cleanRebaseStatusCommand().ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog()
	if err := requireCleanRebaseStatus(status); err != nil {
		return CapturedCombinePlan{}, err
	}
	ref, head, err := branches.namedCheckoutHead()
	if err != nil {
		return CapturedCombinePlan{}, err
	}
	if head == "" {
		return CapturedCombinePlan{}, fmt.Errorf("cannot combine commits with an unborn HEAD")
	}
	if err := branches.validateLiteralCommitOID(head); err != nil {
		return CapturedCombinePlan{}, fmt.Errorf("current HEAD is not an existing commit: %w", err)
	}

	sync := NewSyncCommands(self.GitCommon)
	earliest := selected[0]
	for _, candidate := range selected[1:] {
		candidateFollowsEarliest, err := sync.branchSyncIsCommitAncestor(earliest, candidate)
		if err != nil {
			return CapturedCombinePlan{}, err
		}
		if candidateFollowsEarliest {
			continue
		}
		earliestFollowsCandidate, err := sync.branchSyncIsCommitAncestor(candidate, earliest)
		if err != nil {
			return CapturedCombinePlan{}, err
		}
		if !earliestFollowsCandidate {
			return CapturedCombinePlan{}, fmt.Errorf("selected commits must lie on one ancestry")
		}
		earliest = candidate
	}
	for _, oid := range selected {
		ok, err := sync.branchSyncIsCommitAncestor(oid, head)
		if err != nil {
			return CapturedCombinePlan{}, err
		}
		if !ok {
			return CapturedCombinePlan{}, fmt.Errorf("selected commit %q is not an ancestor of current HEAD", oid)
		}
	}

	start, err := self.readCombineCommit(earliest)
	if err != nil {
		return CapturedCombinePlan{}, err
	}
	if len(start.Parents) > 0 {
		if err := branches.validateLiteralCommitOID(start.Parents[0]); err != nil {
			return CapturedCombinePlan{}, fmt.Errorf("selected commit parent %q is unavailable (possibly a shallow boundary): %w", start.Parents[0], err)
		}
	}
	args := NewGitCmd("rev-list").Arg("--reverse", "--first-parent", head)
	if len(start.Parents) > 0 {
		args.Arg("^" + start.Parents[0])
	}
	args.Arg("--")
	output, diagnostic, err := self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return CapturedCombinePlan{}, fmt.Errorf("capture combine history: %w: %s", err, strings.TrimSpace(diagnostic))
	}
	history := make([]CombineHistoryCommit, 0)
	for _, oid := range strings.Fields(output) {
		commit, err := self.readCombineCommit(strings.ToLower(oid))
		if err != nil {
			return CapturedCombinePlan{}, err
		}
		history = append(history, commit)
	}
	topology, err := PlanLinearCombine(history, selected)
	if err != nil {
		return CapturedCombinePlan{}, err
	}
	if topology.HeadOID() != strings.ToLower(head) {
		return CapturedCombinePlan{}, fmt.Errorf("captured history does not end at current HEAD")
	}
	paths := make(map[string]struct{})
	for _, commit := range history {
		out, err := self.combineGitOutput(NewGitCmd("diff-tree").Arg("--root", "--no-renames", "--no-commit-id", "--name-only", "-r", "-z", commit.OID, "--").ToArgv())
		if err != nil {
			return CapturedCombinePlan{}, fmt.Errorf("list paths changed by %s: %w", commit.OID, err)
		}
		for _, path := range nulPaths(out) {
			paths[path] = struct{}{}
		}
	}
	if len(start.Parents) > 0 {
		out, err := self.combineGitOutput(NewGitCmd("ls-tree").Arg("-r", "-z", "--name-only", start.Parents[0]+"^{commit}").ToArgv())
		if err != nil {
			return CapturedCombinePlan{}, fmt.Errorf("list combine base paths: %w", err)
		}
		for _, path := range nulPaths(out) {
			paths[path] = struct{}{}
		}
	}
	ignored, fingerprint, err := self.ignoredRisksForPaths(paths)
	if err != nil {
		return CapturedCombinePlan{}, err
	}
	return CapturedCombinePlan{owner: self.GitCommon, headRef: ref, headOID: strings.ToLower(head), topology: topology, ignoredPaths: ignored, ignoredFingerprint: fingerprint, prepared: true}, nil
}

func (self *RebaseCommands) readCombineCommit(oid string) (CombineHistoryCommit, error) {
	raw, err := self.combineGitOutput(NewGitCmd("cat-file").Arg("commit", oid).ToArgv())
	if err != nil {
		return CombineHistoryCommit{}, fmt.Errorf("read commit %s: %w", oid, err)
	}
	separator := bytes.Index(raw, []byte("\n\n"))
	if separator < 0 {
		return CombineHistoryCommit{}, fmt.Errorf("commit %s has malformed headers", oid)
	}
	var parents []string
	for _, line := range strings.Split(string(raw[:separator]), "\n") {
		if strings.HasPrefix(line, "parent ") {
			parents = append(parents, strings.TrimPrefix(line, "parent "))
		}
	}
	return CombineHistoryCommit{OID: strings.ToLower(oid), Parents: parents, Message: string(raw[separator+2:])}, nil
}

func (self *RebaseCommands) combineGitOutput(args []string) ([]byte, error) {
	stdout, _, err := self.cmd.New(args).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	return []byte(stdout), err
}

// CheckCombinePlan confirms the reviewed selection and repository snapshot are
// still current and enforces explicit approval for ignored-file collisions.
func (self *RebaseCommands) CheckCombinePlan(plan CapturedCombinePlan) error {
	if !plan.prepared || plan.owner != self.GitCommon {
		return ErrBranchActionChanged
	}
	fresh, err := self.PrepareCombinePlan(plan.topology.SelectedOIDs())
	if err != nil {
		return err
	}
	if fresh.owner != plan.owner || fresh.headRef != plan.headRef || fresh.headOID != plan.headOID ||
		fresh.topology.HeadOID() != plan.topology.HeadOID() || fresh.topology.KeeperOID() != plan.topology.KeeperOID() ||
		fresh.topology.Todo() != plan.topology.Todo() || fresh.topology.Summary() != plan.topology.Summary() ||
		fresh.topology.Description() != plan.topology.Description() || !slices.Equal(fresh.ignoredPaths, plan.ignoredPaths) ||
		fresh.ignoredFingerprint != plan.ignoredFingerprint {
		return ErrBranchActionChanged
	}
	if len(fresh.ignoredPaths) > 0 && !plan.ignoredApproved {
		return ErrRebaseIgnoredApprovalRequired
	}
	return nil
}
