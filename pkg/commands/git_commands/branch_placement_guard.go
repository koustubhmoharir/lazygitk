package git_commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type branchPlacementOccupation struct {
	GitDir  string
	Current bool
	Kind    string
}

// branchPlacementOccupancy checks Git's shared worktree metadata and active
// operation state. It deliberately performs no ref or worktree mutations.
func (self *BranchCommands) branchPlacementOccupancy(fullRef string) (*branchPlacementOccupation, error) {
	return self.branchOccupancy(fullRef, false)
}

// branchOccupancy checks Git's shared worktree metadata and active operation
// state. allowCurrentCheckout permits a caller that will attach this worktree
// to the branch to ignore only its existing HEAD reservation; other worktrees
// and operation reservations remain authoritative.
func (self *BranchCommands) branchOccupancy(fullRef string, allowCurrentCheckout bool) (*branchPlacementOccupation, error) {
	return self.branchOccupancyWithCurrentPolicy(fullRef, allowCurrentCheckout, false)
}

// Current-HEAD actions such as reset can intentionally abort an operation in
// this worktree, but still must not move a ref reserved by another worktree.
func (self *BranchCommands) branchOtherWorktreeOccupancy(fullRef string) (*branchPlacementOccupation, error) {
	return self.branchOccupancyWithCurrentPolicy(fullRef, false, true)
}

func (self *BranchCommands) branchOccupancyWithCurrentPolicy(fullRef string, allowCurrentCheckout, ignoreCurrent bool) (*branchPlacementOccupation, error) {
	commonDir, err := canonicalMetadataPath(self.repoPaths.RepoGitDirPath())
	if err != nil {
		return nil, fmt.Errorf("resolve common Git directory: %w", err)
	}
	worktreesDir := filepath.Join(commonDir, "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read worktree metadata directory: %w", err)
	}
	metadataDirs := []string{commonDir}
	for _, entry := range entries {
		metadataDir := filepath.Join(worktreesDir, entry.Name())
		info, statErr := os.Stat(metadataDir) // follows directory symlinks conservatively
		if statErr != nil {
			return nil, fmt.Errorf("inspect worktree metadata %q: %w", metadataDir, statErr)
		}
		if info.IsDir() {
			metadataDirs = append(metadataDirs, metadataDir)
		}
	}
	currentDir, err := canonicalMetadataPath(self.repoPaths.WorktreeGitDirPath())
	if err != nil {
		return nil, fmt.Errorf("resolve current worktree Git directory: %w", err)
	}
	canonicalDirs := make([]string, 0, len(metadataDirs))
	for _, metadataDir := range metadataDirs {
		canonicalDir, pathErr := canonicalMetadataPath(metadataDir)
		if pathErr != nil {
			return nil, fmt.Errorf("resolve worktree metadata %q: %w", metadataDir, pathErr)
		}
		canonicalDirs = append(canonicalDirs, canonicalDir)
	}
	currentFound := false
	for _, dir := range canonicalDirs {
		if dir == currentDir {
			currentFound = true
			break
		}
	}
	if !currentFound {
		return nil, fmt.Errorf("current worktree Git directory %q is absent from discovered metadata", currentDir)
	}
	// Query the current worktree first so its more specific reservation reason wins.
	for i, dir := range canonicalDirs {
		if dir == currentDir {
			canonicalDirs[0], canonicalDirs[i] = canonicalDirs[i], canonicalDirs[0]
			break
		}
	}

	bare, err := self.commonRepoIsBare(commonDir)
	if err != nil {
		return nil, err
	}
	for _, metadataDir := range canonicalDirs {
		if ignoreCurrent && metadataDir == currentDir {
			continue
		}
		if metadataDir == commonDir && bare {
			continue // A bare repository's own HEAD is not a checkout reservation.
		}
		occupation, err := self.occupationInMetadata(metadataDir, currentDir, fullRef, allowCurrentCheckout)
		if err != nil || occupation != nil {
			return occupation, err
		}
	}
	return nil, nil
}

func canonicalMetadataPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("metadata path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func (self *BranchCommands) commonRepoIsBare(commonDir string) (bool, error) {
	cmd := self.cmd.New(NewGitCmd("config").GitDir(commonDir).Arg("--bool", "core.bare").ToArgv())
	cmd = ForOtherRepo(cmd).SetWd(self.repoPaths.WorktreePath()).DontLog()
	output, err := cmd.RunWithOutput()
	if err != nil {
		if commandExitedWithOne(err) {
			return false, nil
		}
		return false, fmt.Errorf("query core.bare: %w", err)
	}
	value := strings.TrimSuffix(strings.TrimSuffix(output, "\n"), "\r")
	if value == "" {
		return false, nil
	}
	if value == "true" {
		return true, nil
	}
	if value == "false" {
		return false, nil
	}
	return false, fmt.Errorf("invalid core.bare value %q", value)
}

func (self *BranchCommands) occupationInMetadata(metadataDir, currentDir, fullRef string, allowCurrentCheckout bool) (*branchPlacementOccupation, error) {
	current := metadataDir == currentDir
	query := func(args ...string) (string, error) {
		builder := NewGitCmd(args[0]).GitDir(metadataDir).Arg(args[1:]...)
		cmd := self.cmd.New(builder.ToArgv())
		return ForOtherRepo(cmd).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	}
	if head, err := query("symbolic-ref", "--quiet", "HEAD"); err == nil {
		head = trimTerminalLineEnding(head)
		if head == "" || !strings.HasPrefix(head, "refs/heads/") {
			return nil, fmt.Errorf("malformed symbolic HEAD %q in %q", head, metadataDir)
		}
		if head == fullRef {
			// The caller may replace only this worktree's own checkout reservation.
			// Continue scanning operation reservations and every other worktree.
			if !(current && allowCurrentCheckout) {
				return &branchPlacementOccupation{GitDir: metadataDir, Current: current, Kind: "checkout"}, nil
			}
		}
	} else if !commandExitedWithOne(err) {
		return nil, fmt.Errorf("query HEAD in %q: %w", metadataDir, err)
	}

	for _, operation := range []struct{ directory, kind string }{{"rebase-merge", "rebase"}, {"rebase-apply", "rebase"}} {
		name := filepath.Join(metadataDir, operation.directory, "head-name")
		data, err := os.ReadFile(name)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", name, err)
		}
		value := trimTerminalLineEnding(string(data))
		if value == "detached HEAD" {
			continue
		}
		if !strings.HasPrefix(value, "refs/heads/") || !self.metadataRefIsValid(value) {
			return nil, fmt.Errorf("malformed active rebase head-name %q in %q", value, name)
		}
		if value == fullRef {
			return &branchPlacementOccupation{GitDir: metadataDir, Current: current, Kind: operation.kind}, nil
		}
	}

	bisectPath := filepath.Join(metadataDir, "BISECT_START")
	if data, err := os.ReadFile(bisectPath); err == nil {
		value := trimTerminalLineEnding(string(data))
		ref := value
		if !strings.HasPrefix(ref, "refs/") {
			ref = "refs/heads/" + ref
		}
		if !self.metadataRefIsValid(ref) {
			return nil, fmt.Errorf("malformed BISECT_START %q in %q", value, bisectPath)
		}
		if ref == fullRef {
			return &branchPlacementOccupation{GitDir: metadataDir, Current: current, Kind: "bisect"}, nil
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %q: %w", bisectPath, err)
	}

	updateRefsPath := filepath.Join(metadataDir, "rebase-merge", "update-refs")
	data, err := os.ReadFile(updateRefsPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %q: %w", updateRefsPath, err)
	}
	if err == nil && len(data) > 0 {
		fields := strings.Split(string(data), "\n")
		if fields[len(fields)-1] == "" {
			fields = fields[:len(fields)-1]
		}
		for i := range fields {
			fields[i] = strings.TrimSuffix(fields[i], "\r")
		}
		if len(fields)%3 != 0 {
			return nil, fmt.Errorf("malformed rebase update-refs triples in %q", updateRefsPath)
		}
		for i := 0; i < len(fields); i += 3 {
			ref := fields[i]
			if !self.metadataRefIsValid(ref) || validateObjectID(fields[i+1]) != nil || validateObjectID(fields[i+2]) != nil {
				return nil, fmt.Errorf("malformed rebase update-refs record in %q", updateRefsPath)
			}
			if ref == fullRef {
				return &branchPlacementOccupation{GitDir: metadataDir, Current: current, Kind: "rebase update-refs"}, nil
			}
		}
	}
	return nil, nil
}

func (self *BranchCommands) metadataRefIsValid(ref string) bool {
	if !strings.HasPrefix(ref, "refs/") || ref == "refs/" {
		return false
	}
	cmd := self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv())
	return ForOtherRepo(cmd).SetWd(self.repoPaths.WorktreePath()).DontLog().Run() == nil
}

func trimTerminalLineEnding(value string) string {
	value = strings.TrimSuffix(value, "\n")
	return strings.TrimSuffix(value, "\r")
}

func commandExitedWithOne(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}
