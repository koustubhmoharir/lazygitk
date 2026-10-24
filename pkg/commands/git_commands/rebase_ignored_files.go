package git_commands

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

func (self *RebaseCommands) selectedCommitRebaseIgnoredPaths(target, head string) ([]string, string, error) {
	return self.rebaseIgnoredPaths(target, target, head)
}

// The destination tree and replay range can differ for a marked-base --onto
// rebase. Keep the existing collision/fingerprint policy shared by both flows.
func (self *RebaseCommands) rebaseIgnoredPaths(target, replayBase, head string) ([]string, string, error) {
	targetTree, err := self.selectedRebaseGitOutput(NewGitCmd("ls-tree").Arg("-r", "-z", "--name-only", target+"^{commit}").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list rebase target paths: %w", err)
	}
	replayed, err := self.selectedRebaseGitOutput(NewGitCmd("log").Arg("--format=", "--name-only", "--no-renames", "--root", "-m", "-z", replayBase+"^{commit}"+".."+head+"^{commit}", "--").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list replayed rebase paths: %w", err)
	}
	candidates := make(map[string]struct{})
	for _, raw := range [][]byte{targetTree, replayed} {
		for _, path := range nulPaths(raw) {
			candidates[path] = struct{}{}
		}
	}
	return self.ignoredRisksForPaths(candidates)
}

// Share collision and content-fingerprint policy with other captured history
// operations. Callers supply exactly the paths their native operation may touch.
func (self *RebaseCommands) ignoredRisksForPaths(candidates map[string]struct{}) ([]string, string, error) {
	ignoredOutput, err := self.selectedRebaseGitOutput(NewGitCmd("ls-files").Arg("--others", "--ignored", "--exclude-standard", "-z").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list ignored files: %w", err)
	}
	ignoreCase, err := self.selectedRebaseIgnoreCase()
	if err != nil {
		return nil, "", err
	}
	candidateSet := make(map[string]struct{}, len(candidates))
	candidatePaths := make([]string, 0, len(candidates))
	for candidate := range candidates {
		key := rebasePathKey(candidate, ignoreCase)
		candidateSet[key] = struct{}{}
		candidatePaths = append(candidatePaths, key)
	}
	sort.Strings(candidatePaths)
	riskSet := make(map[string]struct{})
	for _, ignored := range nulPaths(ignoredOutput) {
		if rebaseCandidatePathCollides(rebasePathKey(ignored, ignoreCase), candidateSet, candidatePaths) {
			riskSet[ignored] = struct{}{}
		}
	}
	paths := make([]string, 0, len(riskSet))
	for path := range riskSet {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	fingerprint, err := fingerprintIgnoredPaths(self.repoPaths.WorktreePath(), paths)
	if err != nil {
		return nil, "", fmt.Errorf("fingerprint ignored rebase risks: %w", err)
	}
	return paths, fingerprint, nil
}

func (self *RebaseCommands) selectedRebaseGitOutput(args []string) ([]byte, error) {
	output, _, err := self.cmd.New(args).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

func (self *RebaseCommands) selectedRebaseIgnoreCase() (bool, error) {
	output, err := self.selectedRebaseGitOutput(NewGitCmd("config").Arg("--type=bool", "--get", "core.ignoreCase").ToArgv())
	if err != nil {
		if commandExitedWithOne(err) {
			return false, nil
		}
		return false, fmt.Errorf("query core.ignoreCase: %w", err)
	}
	switch strings.TrimSpace(string(output)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid core.ignoreCase response %q", output)
	}
}

func nulPaths(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	parts := strings.Split(string(raw), "\x00")
	paths := make([]string, 0, len(parts))
	for _, path := range parts {
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func rebasePathKey(path string, ignoreCase bool) string {
	if ignoreCase {
		return strings.ToLower(path)
	}
	return path
}

func rebaseCandidatePathCollides(path string, candidates map[string]struct{}, sortedCandidates []string) bool {
	if _, exists := candidates[path]; exists {
		return true
	}
	for slash := strings.IndexByte(path, '/'); slash >= 0; {
		if _, exists := candidates[path[:slash]]; exists {
			return true
		}
		nextSlash := strings.IndexByte(path[slash+1:], '/')
		if nextSlash < 0 {
			break
		}
		slash += nextSlash + 1
	}
	prefix := path + "/"
	index, found := slices.BinarySearch(sortedCandidates, prefix)
	if found {
		return true
	}
	return index < len(sortedCandidates) && strings.HasPrefix(sortedCandidates[index], prefix)
}

func fingerprintIgnoredPaths(root string, paths []string) (string, error) {
	h := sha256.New()
	for _, path := range paths {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		info, err := os.Lstat(fullPath)
		if err != nil {
			return "", err
		}
		writeFingerprintString(h, path)
		writeFingerprintString(h, fmt.Sprintf("%d:%d", info.Mode().Type(), info.Mode().Perm()))
		switch {
		case info.Mode().IsRegular():
			if err := fingerprintRegularFile(h, fullPath, info); err != nil {
				return "", err
			}
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(fullPath)
			if err != nil {
				return "", err
			}
			writeFingerprintString(h, link)
			after, err := os.Lstat(fullPath)
			if err != nil || !os.SameFile(info, after) || info.Mode() != after.Mode() {
				return "", fmt.Errorf("symlink changed while fingerprinting %q", path)
			}
		default:
			return "", fmt.Errorf("unsupported ignored path type %q (%s)", path, info.Mode())
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fingerprintRegularFile(h hash.Hash, path string, before os.FileInfo) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		_ = f.Close()
		return fmt.Errorf("file changed before fingerprinting %q", path)
	}
	contentHash := sha256.New()
	copied, err := io.Copy(contentHash, f)
	if err != nil {
		_ = f.Close()
		return err
	}
	closedErr := f.Close()
	after, err := os.Lstat(path)
	if closedErr != nil {
		return closedErr
	}
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || copied != before.Size() || !before.ModTime().Equal(after.ModTime()) {
		return fmt.Errorf("file changed while fingerprinting %q", path)
	}
	_, _ = h.Write(contentHash.Sum(nil))
	return nil
}

func writeFingerprintString(h hash.Hash, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = h.Write(length[:])
	_, _ = io.WriteString(h, value)
}
