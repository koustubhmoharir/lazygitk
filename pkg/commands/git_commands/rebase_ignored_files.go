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
	return self.rebaseCandidateRisks(rebaseReplay{
		destination: target,
		upstream:    replayBase,
		head:        head,
		logRevs:     []string{replayBase + "^{commit}" + ".." + head + "^{commit}"},
	})
}

// rebaseReplay describes the replay that a rebase start runs. It is the input of
// rebaseCandidateRisks and of the prediction in rebase_ignored_prediction.go.
type rebaseReplay struct {
	// destination is the commit the range is replayed onto, or "" for a root rebase,
	// which rewinds to an empty tree.
	destination string
	// upstream is the commit that the replayed range excludes, or "" for --root.
	upstream string
	// head is the tip that is replayed.
	head string
	// logRevs are the revisions of the replayed range, as git log reads them.
	logRevs []string
	// reordered is set when the todo may move commits, so the replay order may
	// differ from the order that the prediction reads.
	reordered bool
	// extra adds paths that the rebase will create, such as a commit made before
	// the rebase starts. extraRenames are the renames of that commit.
	extra        []string
	extraRenames []renamePair
}

// rebaseCandidateRisks lists the ignored files that collide with the candidate
// paths of a rebase: the destination tree, the replayed paths, the extras and the
// paths that Git's merge backend may create during replay.
func (self *RebaseCommands) rebaseCandidateRisks(replay rebaseReplay) ([]string, string, error) {
	candidates := make(map[string]struct{})
	if replay.destination != "" {
		targetTree, err := self.selectedRebaseGitOutput(NewGitCmd("ls-tree").Arg("-r", "-z", "--name-only", replay.destination+"^{commit}").ToArgv())
		if err != nil {
			return nil, "", fmt.Errorf("list rebase target paths: %w", err)
		}
		for _, path := range nulPaths(targetTree) {
			candidates[path] = struct{}{}
		}
	}
	replayed, err := self.selectedRebaseGitOutput(NewGitCmd("log").Arg("--format=", "--name-only", "--no-renames", "--root", "-m", "-z").Arg(replay.logRevs...).Arg("--").ToArgv())
	if err != nil {
		return nil, "", fmt.Errorf("list replayed rebase paths: %w", err)
	}
	for _, path := range nulPaths(replayed) {
		candidates[path] = struct{}{}
	}
	for _, path := range replay.extra {
		candidates[path] = struct{}{}
	}
	ignored, err := self.ignoredFilesSnapshot()
	if err != nil {
		return nil, "", err
	}
	// Without ignored files nothing can be at risk, so the replay prediction, which
	// costs several Git calls, is skipped.
	if len(ignored.paths) > 0 {
		known := append(nulPaths(replayed), replay.extra...)
		predicted, err := self.predictedRebaseCandidates(replay, known, ignored)
		if err != nil {
			return nil, "", err
		}
		for path := range predicted {
			candidates[path] = struct{}{}
		}
	}
	return self.risksFromSnapshot(candidates, ignored)
}

// Share collision and content-fingerprint policy with other captured history
// operations. Callers supply exactly the paths their native operation may touch.
func (self *RebaseCommands) ignoredRisksForPaths(candidates map[string]struct{}) ([]string, string, error) {
	ignored, err := self.ignoredFilesSnapshot()
	if err != nil {
		return nil, "", err
	}
	return self.risksFromSnapshot(candidates, ignored)
}

// ignoredFiles is one reading of the worktree's ignored files and the case policy,
// shared by the checks of one risk calculation.
type ignoredFiles struct {
	paths      []string
	ignoreCase bool
}

func (self *RebaseCommands) ignoredFilesSnapshot() (ignoredFiles, error) {
	ignoredOutput, err := self.selectedRebaseGitOutput(NewGitCmd("ls-files").Arg("--others", "--ignored", "--exclude-standard", "-z").ToArgv())
	if err != nil {
		return ignoredFiles{}, fmt.Errorf("list ignored files: %w", err)
	}
	ignoreCase, err := self.selectedRebaseIgnoreCase()
	if err != nil {
		return ignoredFiles{}, err
	}
	return ignoredFiles{paths: nulPaths(ignoredOutput), ignoreCase: ignoreCase}, nil
}

// risksFromSnapshot lists the ignored files that collide with candidates, with their
// fingerprint.
func (self *RebaseCommands) risksFromSnapshot(candidates map[string]struct{}, ignored ignoredFiles) ([]string, string, error) {
	paths := collidingIgnoredPaths(candidates, ignored)
	fingerprint, err := fingerprintIgnoredPaths(self.repoPaths.WorktreePath(), paths)
	if err != nil {
		return nil, "", fmt.Errorf("fingerprint ignored rebase risks: %w", err)
	}
	return paths, fingerprint, nil
}

// collidingIgnoredPaths lists, sorted, the ignored files that collide with candidates.
func collidingIgnoredPaths(candidates map[string]struct{}, ignored ignoredFiles) []string {
	ignoreCase := ignored.ignoreCase
	candidateSet := make(map[string]struct{}, len(candidates))
	candidatePaths := make([]string, 0, len(candidates))
	for candidate := range candidates {
		key := rebasePathKey(candidate, ignoreCase)
		candidateSet[key] = struct{}{}
		candidatePaths = append(candidatePaths, key)
	}
	sort.Strings(candidatePaths)
	riskSet := make(map[string]struct{})
	for _, path := range ignored.paths {
		if rebaseCandidatePathCollides(rebasePathKey(path, ignoreCase), candidateSet, candidatePaths) {
			riskSet[path] = struct{}{}
		}
	}
	paths := make([]string, 0, len(riskSet))
	for path := range riskSet {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
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
	// The empty candidate is the root directory. It holds every top-level path.
	if _, exists := candidates[""]; exists && strings.IndexByte(path, '/') < 0 {
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
