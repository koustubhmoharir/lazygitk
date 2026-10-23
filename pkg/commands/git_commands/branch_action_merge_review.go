package git_commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

// BranchActionMergeReview captures native merge inputs without running merge
// drivers or writing the index. It is not permission to merge: the chosen mode
// still needs reservation, ignored-file and staged-change approval checks.
type BranchActionMergeReview struct {
	target                         BranchActionTarget
	applicationArguments           string
	arguments                      []string
	records                        []branchActionMergeArgument
	configSignature, indexEntries  string
	defaultFF                      string
	defaultStrategies, stagedPaths []string
	canFastForward, alreadyMerged  bool
}

func (review BranchActionMergeReview) Target() BranchActionTarget { return review.target }
func (review BranchActionMergeReview) Arguments() []string        { return slices.Clone(review.arguments) }

func (review BranchActionMergeReview) StagedFiles() []string { return slices.Clone(review.stagedPaths) }
func (review BranchActionMergeReview) CanFastForward() bool  { return review.canFastForward }
func (review BranchActionMergeReview) AlreadyMerged() bool   { return review.alreadyMerged }

// PrepareBranchActionMergeReview retains Git's last branch.mergeOptions value,
// followed by application arguments. The eventual command must clear implicit
// branch options and pass these captured arguments once, in the same order.
func (self *BranchCommands) PrepareBranchActionMergeReview(target BranchActionTarget, applicationArguments string) (BranchActionMergeReview, error) {
	if err := self.CheckBranchActionTarget(target); err != nil {
		return BranchActionMergeReview{}, err
	}
	bare, err := self.namedCheckoutContextIsBare()
	if err != nil {
		return BranchActionMergeReview{}, err
	}
	if bare || target.HeadRef() == "" {
		return BranchActionMergeReview{}, fmt.Errorf("merge requires a checked-out local branch")
	}
	state, err := self.readPlacementRef(target.HeadRef())
	if err != nil {
		return BranchActionMergeReview{}, err
	}
	if state.symbolic != "" {
		return BranchActionMergeReview{}, fmt.Errorf("merge requires a direct current branch, not a symbolic alias")
	}
	if err := self.rejectWorktreeOperation("start a merge"); err != nil {
		return BranchActionMergeReview{}, err
	}
	review := BranchActionMergeReview{target: target, applicationArguments: applicationArguments}
	if err := self.readBranchActionMergeConfiguration(&review); err != nil {
		return BranchActionMergeReview{}, err
	}
	output, _, err := self.cmd.New(NewGitCmd("ls-files").Arg("--stage", "-z").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return BranchActionMergeReview{}, err
	}
	if err := validateBranchActionMergeIndex(output); err != nil {
		return BranchActionMergeReview{}, err
	}
	review.indexEntries = output
	output, _, err = self.cmd.New(NewGitCmd("diff").Arg("--cached", "--name-only", "--no-renames", "--no-ext-diff", "--no-textconv", "-z", "--").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return BranchActionMergeReview{}, err
	}
	if output != "" && !strings.HasSuffix(output, "\x00") {
		return BranchActionMergeReview{}, fmt.Errorf("Git returned unterminated staged paths")
	}
	review.stagedPaths = nulPaths([]byte(output))
	if target.HeadOID() == "" {
		review.canFastForward = true
	} else {
		sync := &SyncCommands{GitCommon: self.GitCommon}
		review.canFastForward, err = sync.branchSyncIsCommitAncestor(target.HeadOID(), target.SourceOID())
		if err != nil {
			return BranchActionMergeReview{}, err
		}
		review.alreadyMerged, err = sync.branchSyncIsCommitAncestor(target.SourceOID(), target.HeadOID())
		if err != nil {
			return BranchActionMergeReview{}, err
		}
	}
	if err := self.CheckBranchActionTarget(target); err != nil {
		return BranchActionMergeReview{}, err
	}
	return review, nil
}

// CheckBranchActionMergeReview deliberately does not invoke merge-tree again.
// Final validation must not repeat configured low-level merge drivers.
func (self *BranchCommands) CheckBranchActionMergeReview(review BranchActionMergeReview) error {
	fresh, err := self.PrepareBranchActionMergeReview(review.target, review.applicationArguments)
	if err != nil {
		return err
	}
	if fresh.configSignature != review.configSignature || fresh.indexEntries != review.indexEntries ||
		!slices.Equal(fresh.stagedPaths, review.stagedPaths) || !slices.Equal(fresh.arguments, review.arguments) ||
		fresh.defaultFF != review.defaultFF || !slices.Equal(fresh.defaultStrategies, review.defaultStrategies) ||
		fresh.canFastForward != review.canFastForward || fresh.alreadyMerged != review.alreadyMerged {
		return ErrBranchActionChanged
	}
	return nil
}

func (self *BranchCommands) readBranchActionMergeConfiguration(review *BranchActionMergeReview) error {
	sync := &SyncCommands{GitCommon: self.GitCommon}
	entries, err := sync.branchSyncConfigEntries()
	if err != nil {
		return err
	}
	digest := sha256.New()
	var branchOptions string
	for _, entry := range entries {
		// Keep this digest private: configuration may contain credentials. It
		// also binds hook, signing and externally included configuration values.
		writeBranchSyncConfigEntry(digest, entry)
		switch entry.key {
		case "branch." + strings.TrimPrefix(review.target.HeadRef(), "refs/heads/") + ".mergeoptions":
			if !entry.hasValue {
				return fmt.Errorf("current branch mergeOptions requires a value")
			}
			branchOptions = entry.value
		case "pull.twohead":
			if !entry.hasValue {
				return fmt.Errorf("pull.twohead requires a strategy value")
			}
			// Native merge splits on literal spaces, retaining empty entries.
			// Tabs, quotes and repeated spaces must not acquire new semantics.
			review.defaultStrategies = strings.Split(entry.value, " ")
		}
	}
	review.configSignature = hex.EncodeToString(digest.Sum(nil))
	review.defaultFF = "true"
	ff, found, err := sync.pullReviewConfig("merge.ff")
	if err != nil {
		return err
	}
	if found && (ff == "true" || ff == "false" || ff == "only") {
		review.defaultFF = ff // Git ignores unrecognized future values.
	}
	for _, value := range []string{branchOptions, review.applicationArguments} {
		args, records, err := parseBranchActionMergeOptionRecords(value)
		if err != nil {
			return err
		}
		review.arguments = append(review.arguments, args...)
		review.records = append(review.records, records...)
	}
	return nil
}

func validateBranchActionMergeIndex(output string) error {
	for output != "" {
		entry, rest, terminated := strings.Cut(output, "\x00")
		metadata, path, hasPath := strings.Cut(entry, "\t")
		fields := strings.Fields(metadata)
		if !terminated || !hasPath || path == "" || len(fields) != 3 || !IsFullObjectID(fields[1]) {
			return fmt.Errorf("Git returned invalid merge index entries")
		}
		if fields[2] != "0" {
			return fmt.Errorf("resolve unmerged index entries before starting a merge")
		}
		output = rest
	}
	return nil
}
