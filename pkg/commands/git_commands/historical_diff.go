package git_commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

type HistoricalDiffCommit struct {
	// OID is the full object ID of the selected commit.
	OID string
	// FirstParent is empty for a root commit or an unavailable shallow parent.
	FirstParent string
}

// HistoricalFileDiff owns a read-only temporary copy of a historical blob.
// Keep it open until Command has finished, then call Close to remove its private
// temporary directory. Close may be retried if cleanup fails.
type HistoricalFileDiff struct {
	Command         *oscommands.CmdObj
	SourcePath      string
	DestinationPath string
	tempDir         string
}

// ResolveHistoricalDiffCommit validates and resolves a full commit object ID,
// returning its first parent when Git has that parent available.
func (self *DiffCommands) ResolveHistoricalDiffCommit(oid string) (HistoricalDiffCommit, error) {
	if err := validateObjectID(oid); err != nil {
		return HistoricalDiffCommit{}, err
	}
	worktree := self.repoPaths.WorktreePath()
	resolved, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", "--end-of-options", oid+"^{commit}").ToArgv()).SetWd(worktree).RunWithOutput()
	if err != nil {
		return HistoricalDiffCommit{}, err
	}
	resolved = strings.TrimSpace(resolved)
	if err := validateObjectID(resolved); err != nil {
		return HistoricalDiffCommit{}, fmt.Errorf("git returned invalid commit object ID: %w", err)
	}
	parents, err := self.cmd.New(NewGitCmd("rev-list").Arg("--parents", "-n", "1", resolved, "--").ToArgv()).SetWd(worktree).RunWithOutput()
	if err != nil {
		return HistoricalDiffCommit{}, err
	}
	fields := strings.Fields(parents)
	if len(fields) == 0 || fields[0] != resolved {
		return HistoricalDiffCommit{}, fmt.Errorf("git returned invalid commit parent list")
	}
	result := HistoricalDiffCommit{OID: resolved}
	if len(fields) > 1 {
		if err := validateObjectID(fields[1]); err != nil {
			return HistoricalDiffCommit{}, fmt.Errorf("git returned invalid first parent: %w", err)
		}
		result.FirstParent = fields[1]
	}
	return result, nil
}

// HistoricalRevisionDiffCmdObj builds a difftool command for the selected
// commit against its first parent or the working tree, scoped to literal paths.
func (self *DiffCommands) HistoricalRevisionDiffCmdObj(commit HistoricalDiffCommit, previous bool, paths []string, directory bool) (*oscommands.CmdObj, error) {
	if err := validateObjectID(commit.OID); err != nil {
		return nil, err
	}
	if previous {
		if err := validateObjectID(commit.FirstParent); err != nil {
			return nil, fmt.Errorf("previous-commit diff requires a valid first parent: %w", err)
		}
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, path := range paths {
		if path == "." {
			continue
		}
		if !validTreePath(path) {
			return nil, fmt.Errorf("invalid repository path %q", path)
		}
	}
	args := NewGitCmd("--literal-pathspecs").Arg("difftool", "--no-prompt").ArgIf(directory, "--dir-diff")
	if previous {
		args.Arg(commit.FirstParent, commit.OID)
	} else {
		args.Arg(commit.OID)
	}
	args.Arg("--").Arg(paths...)
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()), nil
}

// PrepareHistoricalFileDiff materializes the exact blob at treePath from
// commitOID and builds a no-index diff against an existing worktree file.
// The returned object owns its source temporary file until Close is called.
func (self *DiffCommands) PrepareHistoricalFileDiff(commitOID, treePath, destination string) (result *HistoricalFileDiff, resultErr error) {
	if err := validateObjectID(commitOID); err != nil {
		return nil, err
	}
	if !validTreePath(treePath) {
		return nil, fmt.Errorf("invalid canonical repository tree path %q", treePath)
	}
	root, err := filepath.EvalSymlinks(self.repoPaths.WorktreePath())
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	dest, err := resolveDestination(root, destination)
	if err != nil {
		return nil, err
	}
	// A worktree can keep Git metadata somewhere other than a .git directory.
	// Exclude the actual repository directories too, including linked worktrees.
	for _, gitDir := range []string{self.repoPaths.WorktreeGitDirPath(), self.repoPaths.RepoGitDirPath()} {
		if gitDir == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(gitDir); err == nil {
			gitDir = resolved
		}
		gitDir, err = filepath.Abs(gitDir)
		if err != nil {
			return nil, err
		}
		if rel, err := filepath.Rel(gitDir, dest); err == nil && filepath.IsLocal(rel) {
			return nil, fmt.Errorf("destination cannot be inside Git metadata")
		}
	}
	info, err := os.Stat(dest)
	if err != nil {
		return nil, fmt.Errorf("could not access destination file %q: %w", dest, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("destination is not a regular file")
	}

	worktree := self.repoPaths.WorktreePath()
	lookup := self.cmd.New(NewGitCmd("--literal-pathspecs").Arg("ls-tree", "-rz", "--full-tree", commitOID, "--", treePath).ToArgv()).SetWd(worktree)
	output, err := lookup.RunWithOutput()
	if err != nil {
		return nil, err
	}
	mode, typ, blob, err := parseTreeEntry([]byte(output), treePath)
	if err != nil {
		return nil, err
	}
	if mode == "120000" {
		return nil, fmt.Errorf("historical symlinks are not supported")
	}
	if mode == "160000" {
		return nil, fmt.Errorf("historical gitlinks are not supported")
	}
	if mode != "100644" && mode != "100755" {
		return nil, fmt.Errorf("unsupported historical blob mode %q", mode)
	}
	if typ != "blob" {
		return nil, fmt.Errorf("historical path does not resolve to a blob")
	}
	if err := validateObjectID(blob); err != nil {
		return nil, fmt.Errorf("git returned invalid blob object ID: %w", err)
	}

	tempDir, err := os.MkdirTemp(self.os.GetTempDir(), "lazygit-historical-diff-")
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("clean up historical diff temporary directory: %w", cleanupErr))
			}
		}
	}()
	if err := os.Chmod(tempDir, 0o700); err != nil {
		return nil, err
	}
	base := filepath.Base(filepath.FromSlash(treePath))
	source := filepath.Join(tempDir, base)
	file, err := os.OpenFile(source, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	cat := self.cmd.New(NewGitCmd("cat-file").Arg("blob", blob).ToArgv()).SetWd(worktree)
	cat.GetCmd().Stdout = file
	var stderr bytes.Buffer
	cat.GetCmd().Stderr = &stderr
	// The normal runner captures/replaces stdout. Run this read-only command
	// directly to stream arbitrary binary bytes into the private source file,
	// retaining the repo-bound builder's cwd and environment.
	runErr := cat.GetCmd().Run()
	closeErr := file.Close()
	if runErr != nil {
		if stderr.Len() > 0 {
			runErr = fmt.Errorf("git cat-file failed: %w: %s", runErr, strings.TrimSpace(stderr.String()))
		} else {
			runErr = fmt.Errorf("git cat-file failed: %w", runErr)
		}
		return nil, errors.Join(runErr, closeErr)
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err := os.Chmod(source, 0o400); err != nil {
		return nil, err
	}
	args := NewGitCmd("difftool").Arg("--no-index", "--no-prompt", "--trust-exit-code", "--", source, dest).ToArgv()
	command := self.cmd.New(args).SetWd(worktree).AcceptSubprocessExitCodes(1)
	cleanup = false
	return &HistoricalFileDiff{Command: command, SourcePath: source, DestinationPath: dest, tempDir: tempDir}, nil
}

func (self *HistoricalFileDiff) Close() error {
	if self == nil || self.tempDir == "" {
		return nil
	}
	err := os.RemoveAll(self.tempDir)
	if err == nil {
		self.tempDir = ""
	}
	return err
}

func validTreePath(path string) bool {
	if path == "" || strings.ContainsRune(path, '\x00') || strings.HasPrefix(path, "/") || filepath.VolumeName(path) != "" || (runtime.GOOS == "windows" && strings.Contains(path, "\\")) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func resolveDestination(root, destination string) (string, error) {
	if destination == "" || strings.ContainsRune(destination, '\x00') {
		return "", fmt.Errorf("destination path is empty or invalid")
	}
	if !filepath.IsAbs(destination) {
		if !filepath.IsLocal(destination) {
			return "", fmt.Errorf("destination must stay inside the worktree")
		}
		destination = filepath.Join(root, destination)
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("destination must stay inside the worktree")
	}
	relSlash := filepath.ToSlash(rel)
	for _, component := range strings.Split(relSlash, "/") {
		if strings.EqualFold(component, ".git") {
			return "", fmt.Errorf("destination cannot be a git internal")
		}
	}
	if rel == "." {
		return "", fmt.Errorf("destination cannot be a git internal")
	}
	current := root
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("destination path contains a symlink")
		}
	}
	return abs, nil
}

func parseTreeEntry(output []byte, expectedPath string) (string, string, string, error) {
	var found []byte
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		meta, path, ok := bytes.Cut(record, []byte{'\t'})
		if ok && string(path) == expectedPath {
			if found != nil {
				return "", "", "", fmt.Errorf("historical path matched multiple tree entries")
			}
			found = meta
		}
	}
	if found == nil {
		return "", "", "", fmt.Errorf("historical path does not exist in the selected commit")
	}
	fields := strings.Fields(string(found))
	if len(fields) != 3 {
		return "", "", "", fmt.Errorf("malformed git tree entry")
	}
	return fields[0], fields[1], fields[2], nil
}
