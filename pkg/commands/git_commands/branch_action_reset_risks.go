package git_commands

import (
	"fmt"
	"sort"
)

// Fork: a hard reset to a captured target overwrites or removes untracked and
// ignored files at paths that the target tracks. This lists those paths with a
// fingerprint of their contents, so the reset review can show them and the reset
// can refuse a list that changed after review. Collision and fingerprint policy
// is the one rebase_ignored_files.go uses for rebases.

// ResetIgnoredRiskError reports that the untracked and ignored files at risk from a
// hard reset differ from the reviewed ones. Nothing has run. The caller reviews
// Paths afresh and approves Fingerprint.
type ResetIgnoredRiskError struct {
	Paths       []string
	Fingerprint string
}

func (err *ResetIgnoredRiskError) Error() string {
	return "untracked or ignored files at risk from this hard reset changed; review the reset again"
}

func (err *ResetIgnoredRiskError) Unwrap() error {
	return ErrBranchActionChanged
}

// HardResetIgnoredRisks lists, sorted, the untracked and ignored files that a hard
// reset to target would overwrite or remove, with a fingerprint of their contents.
// A file is at risk when the target tracks its path and the current index does not,
// or when it collides with such a path, as a file under a target directory or as a
// directory above a target file. Nothing is written. Without risks the fingerprint
// is empty.
func (self *BranchCommands) HardResetIgnoredRisks(target BranchActionTarget) ([]string, string, error) {
	if err := self.CheckBranchActionTarget(target); err != nil {
		return nil, "", err
	}
	return self.hardResetRisks(target.SourceOID())
}

func (self *BranchCommands) hardResetRisks(sourceOID string) ([]string, string, error) {
	// The rebase helpers provide the shared Git output and ignore-case policy.
	rebase := NewRebaseCommands(self.GitCommon, nil, nil)
	targetTree, err := rebase.selectedRebaseGitOutput(NewGitCmd("ls-tree").Arg("-r", "-z", "--name-only", sourceOID+"^{commit}").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list hard reset target paths: %w", err)
	}
	indexed, err := rebase.selectedRebaseGitOutput(NewGitCmd("ls-files").Arg("-z").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list indexed paths: %w", err)
	}
	untracked, err := rebase.selectedRebaseGitOutput(NewGitCmd("ls-files").Arg("--others", "--exclude-standard", "-z").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list untracked files: %w", err)
	}
	ignored, err := rebase.selectedRebaseGitOutput(NewGitCmd("ls-files").Arg("--others", "--ignored", "--exclude-standard", "-z").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list ignored files: %w", err)
	}
	ignoreCase, err := rebase.selectedRebaseIgnoreCase()
	if err != nil {
		return nil, "", err
	}

	indexedSet := make(map[string]struct{})
	for _, path := range nulPaths(indexed) {
		indexedSet[path] = struct{}{}
	}
	candidates := make(map[string]struct{})
	for _, path := range nulPaths(targetTree) {
		if _, ok := indexedSet[path]; !ok {
			candidates[rebasePathKey(path, ignoreCase)] = struct{}{}
		}
	}
	sortedCandidates := make([]string, 0, len(candidates))
	for key := range candidates {
		sortedCandidates = append(sortedCandidates, key)
	}
	sort.Strings(sortedCandidates)

	var paths []string
	for _, path := range append(nulPaths(untracked), nulPaths(ignored)...) {
		if rebaseCandidatePathCollides(rebasePathKey(path, ignoreCase), candidates, sortedCandidates) {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return nil, "", nil
	}
	sort.Strings(paths)
	fingerprint, err := fingerprintIgnoredPaths(self.repoPaths.WorktreePath(), paths)
	if err != nil {
		return nil, "", fmt.Errorf("fingerprint hard reset risks: %w", err)
	}
	return paths, fingerprint, nil
}

// TrackedChangesPresent reports whether a tracked file, other than a submodule, has
// staged or unstaged changes. Untracked and ignored files do not count. Undo and
// redo refuse to reset while it is true, because they never stash.
func (self *BranchCommands) TrackedChangesPresent() (bool, error) {
	output, err := self.cmd.New(NewGitCmd("status").Arg("--porcelain", "--untracked-files=no", "--ignore-submodules=all", "-z").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return false, fmt.Errorf("check tracked changes: %w", err)
	}
	return output != "", nil
}
