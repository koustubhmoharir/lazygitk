package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchActionMergePreviewCleanConflictAndReadOnly(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(map[bool]string{false: "clean", true: "conflict"}[conflict], func(t *testing.T) {
			root, base, _ := transactionHistory(t)
			gitTest(t, root, "branch", "source", base)
			gitTest(t, root, "branch", "current", base)
			if conflict {
				mergeFixtureCommit(t, root, "source", "shared", "source\n", "source")
				mergeFixtureCommit(t, root, "current", "shared", "current\n", "current")
			} else {
				mergeFixtureCommit(t, root, "source", "source-only", "source\n", "source")
				mergeFixtureCommit(t, root, "current", "current-only", "current\n", "current")
			}
			gitTest(t, root, "checkout", "-q", "current")
			sourceOID := gitTest(t, root, "rev-parse", "refs/heads/source")
			commands, target, before := mergePreviewCaptured(t, root, sourceOID)
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
			assert.NoError(t, err)
			if conflict {
				assert.Contains(t, paths, "shared")
			} else {
				assert.Contains(t, paths, "source-only")
				assert.Contains(t, paths, "current-only")
			}
			assert.Equal(t, before, mergePreviewSnapshot(t, root))
			entries, readErr := os.ReadDir(tmp)
			assert.NoError(t, readErr)
			assert.Empty(t, entries, "private preview object storage must be removed")
		})
	}
}

func TestBranchActionMergePreviewDirectoryRenameAndIgnoredCollision(t *testing.T) {
	t.Run("directory rename result path", func(t *testing.T) {
		root := mergePreviewRepo(t)
		writeTestFile(t, filepath.Join(root, "old", "a"), []byte("base\n"), 0o644)
		gitTest(t, root, "add", "old/a")
		gitTest(t, root, "commit", "-qm", "base")
		base := gitTest(t, root, "rev-parse", "HEAD")
		gitTest(t, root, "branch", "source", base)
		gitTest(t, root, "branch", "current", base)
		gitTest(t, root, "checkout", "-q", "source")
		gitTest(t, root, "mv", "old", "new")
		gitTest(t, root, "commit", "-qam", "move")
		gitTest(t, root, "checkout", "-q", "current")
		writeTestFile(t, filepath.Join(root, "old", "added"), []byte("tracked\n"), 0o644)
		gitTest(t, root, "add", "old/added")
		gitTest(t, root, "commit", "-qm", "add under old")
		gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore"))
		writeTestFile(t, filepath.Join(root, "ignore"), []byte("/new/added\n"), 0o644)
		writeTestFile(t, filepath.Join(root, "new", "added"), []byte("precious ignored bytes\n"), 0o600)
		gitTest(t, root, "check-ignore", "-q", "new/added")
		commands, target, before := mergePreviewCaptured(t, root, gitTest(t, root, "rev-parse", "refs/heads/source"))
		paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
		assert.NoError(t, err)
		assert.Contains(t, paths, "new/added")
		assert.Equal(t, []byte("precious ignored bytes\n"), mustRead(t, filepath.Join(root, "new", "added")))
		assert.Equal(t, before, mergePreviewSnapshot(t, root))
	})
	t.Run("ignored path collision", func(t *testing.T) {
		root, base, _ := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		gitTest(t, root, "branch", "current", base)
		mergeFixtureCommit(t, root, "source", "ignored-item", "source\n", "source")
		gitTest(t, root, "checkout", "-q", "current")
		mergeFixtureCommit(t, root, "current", "ignore-marker", "marker\n", "marker")
		gitTest(t, root, "config", "core.excludesFile", filepath.Join(root, "ignore"))
		writeTestFile(t, filepath.Join(root, "ignore"), []byte("/ignored-item\n"), 0o644)
		writeTestFile(t, filepath.Join(root, "ignored-item"), []byte("precious\x00ignored\n"), 0o600)
		commands, target, before := mergePreviewCaptured(t, root, gitTest(t, root, "rev-parse", "refs/heads/source"))
		paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
		assert.NoError(t, err)
		assert.Contains(t, paths, "ignored-item")
		assert.Equal(t, []byte("precious\x00ignored\n"), mustRead(t, filepath.Join(root, "ignored-item")))
		assert.Equal(t, before, mergePreviewSnapshot(t, root))
	})
}

func TestBranchActionMergePreviewQuotedRepositoryPathAndInheritedAlternate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("newline and Unix path punctuation fixture is unsupported on Windows")
	}
	parent := t.TempDir()
	root := filepath.Join(parent, "repo:quote' space-雪\nline")
	assert.NoError(t, os.Mkdir(root, 0o755))
	initMergePreviewRepo(t, root)
	writeTestFile(t, filepath.Join(root, "base"), []byte("base\n"), 0o644)
	gitTest(t, root, "add", "base")
	gitTest(t, root, "commit", "-qm", "base")
	gitTest(t, root, "branch", "current")
	// The source commit is created in a separate owned repository. The main
	// repository intentionally lacks it; its only object source is inherited.
	other := filepath.Join(parent, "other")
	assert.NoError(t, os.Mkdir(other, 0o755))
	initMergePreviewRepo(t, other)
	gitTest(t, other, "fetch", "-q", root, "refs/heads/master:refs/remotes/origin/master")
	gitTest(t, other, "checkout", "-qb", "master", "refs/remotes/origin/master")
	gitTest(t, other, "checkout", "-qb", "source")
	writeTestFile(t, filepath.Join(other, "shared"), []byte("alternate source\n"), 0o644)
	gitTest(t, other, "add", "shared")
	gitTest(t, other, "commit", "-qm", "source")
	source := gitTest(t, other, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", "current")
	// Install only the base in the primary object store; source's commit/tree/blob
	// are supplied by the second repository as an inherited alternate.
	primaryObjects := gitTest(t, root, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	alternateObjects := gitTest(t, other, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	// The source ref resolves only through the inherited alternate.
	if _, statErr := os.Stat(filepath.Join(primaryObjects, source[:2], source[2:])); !os.IsNotExist(statErr) {
		t.Fatalf("source commit unexpectedly exists in the primary object store: %v", statErr)
	}
	t.Setenv("GIT_ALTERNATE_OBJECT_DIRECTORIES", alternateObjects)
	gitTest(t, root, "update-ref", "refs/heads/source", source)
	commands, target, before := mergePreviewCaptured(t, root, source)
	paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
	assert.NoError(t, err)
	assert.Contains(t, paths, "shared")
	assert.Equal(t, before, mergePreviewSnapshot(t, root))
	assert.DirExists(t, primaryObjects)
}

func TestBranchActionMergePreviewTargetValidationAndUnbornHead(t *testing.T) {
	root, first, second := transactionHistory(t)
	commands := transactionCommands(t, root)
	zero := BranchActionTarget{}
	_, err := commands.PreviewBranchActionOrtMerge(zero, BranchActionMergePreviewOptions{})
	assert.Error(t, err)
	target, err := commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.NoError(t, err)
	other, _, _ := transactionHistory(t)
	_, err = transactionCommands(t, other).PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
	assert.Error(t, err)
	target, err = commands.PrepareBranchActionTarget("refs/heads/master", second)
	assert.NoError(t, err)
	gitTest(t, root, "update-ref", "refs/heads/master", first, second)
	_, err = commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
	assert.ErrorIs(t, err, ErrBranchActionChanged)

	gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/unborn")
	commands = transactionCommands(t, root)
	target, err = commands.PrepareBranchActionTarget("refs/heads/master", first)
	assert.NoError(t, err)
	before := mergePreviewSnapshot(t, root)
	paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
	assert.NoError(t, err)
	assert.Contains(t, paths, "tracked")
	assert.Equal(t, before, mergePreviewSnapshot(t, root))
}

func TestBranchActionMergePreviewUnrelatedHistoriesRequiresExplicitOption(t *testing.T) {
	root := mergePreviewRepo(t)
	writeTestFile(t, filepath.Join(root, "current"), []byte("current\n"), 0o644)
	gitTest(t, root, "add", "current")
	gitTest(t, root, "commit", "-qm", "current root")
	gitTest(t, root, "checkout", "--orphan", "source")
	gitTest(t, root, "rm", "-rfq", ".")
	writeTestFile(t, filepath.Join(root, "source"), []byte("source\n"), 0o644)
	gitTest(t, root, "add", "source")
	gitTest(t, root, "commit", "-qm", "source root")
	source := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", "master")
	commands, target, before := mergePreviewCaptured(t, root, source)
	_, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
	assert.Error(t, err, "unrelated histories require an explicit choice")
	paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{AllowUnrelatedHistories: true})
	assert.NoError(t, err)
	assert.Contains(t, paths, "current")
	assert.Contains(t, paths, "source")
	assert.Equal(t, before, mergePreviewSnapshot(t, root))
}

func TestBranchActionMergePreviewPreservesDirtyIndexWorktreeAndIgnoredFiles(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	gitTest(t, root, "branch", "current", base)
	mergeFixtureCommit(t, root, "source", "source-only", "source\n", "source")
	gitTest(t, root, "checkout", "-q", "current")
	stagedPath := "staged-unrelated"
	writeTestFile(t, filepath.Join(root, stagedPath), []byte("staged bytes\n"), 0o644)
	gitTest(t, root, "add", stagedPath)
	trackedPath := filepath.Join(root, "tracked")
	writeTestFile(t, trackedPath, []byte("unstaged current bytes\n"), 0o644)
	ignoreFile := filepath.Join(t.TempDir(), "global-ignore")
	writeTestFile(t, ignoreFile, []byte("/precious-ignored\n"), 0o644)
	gitTest(t, root, "config", "core.excludesFile", ignoreFile)
	ignoredPath := filepath.Join(root, "precious-ignored")
	writeTestFile(t, ignoredPath, []byte("precious ignored bytes\x00\n"), 0o600)
	commands, target, before := mergePreviewCaptured(t, root, gitTest(t, root, "rev-parse", "refs/heads/source"))
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
	assert.NoError(t, err)
	assert.Contains(t, paths, "source-only")
	assert.Equal(t, []byte("staged bytes\n"), mustRead(t, filepath.Join(root, stagedPath)))
	assert.Equal(t, []byte("unstaged current bytes\n"), mustRead(t, trackedPath))
	assert.Equal(t, []byte("precious ignored bytes\x00\n"), mustRead(t, ignoredPath))
	assert.Equal(t, before, mergePreviewSnapshot(t, root))
	entries, readErr := os.ReadDir(tmp)
	assert.NoError(t, readErr)
	assert.Empty(t, entries, "private preview object storage must be removed")
}

func TestBranchActionMergePreviewStrategyOptions(t *testing.T) {
	t.Run("native ours strategy option", func(t *testing.T) {
		root, base, _ := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		gitTest(t, root, "branch", "current", base)
		mergeFixtureCommit(t, root, "source", "shared", "source\n", "source")
		mergeFixtureCommit(t, root, "current", "shared", "current\n", "current")
		gitTest(t, root, "checkout", "-q", "current")
		commands, target, before := mergePreviewCaptured(t, root, gitTest(t, root, "rev-parse", "refs/heads/source"))
		paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{StrategyOptions: []string{"ours"}})
		assert.NoError(t, err)
		assert.Contains(t, paths, "shared")
		assert.Equal(t, before, mergePreviewSnapshot(t, root))
	})
	t.Run("invalid option preserves native error and repository", func(t *testing.T) {
		root, base, _ := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		gitTest(t, root, "branch", "current", base)
		mergeFixtureCommit(t, root, "source", "source-only", "source\n", "source")
		gitTest(t, root, "checkout", "-q", "current")
		commands, target, before := mergePreviewCaptured(t, root, gitTest(t, root, "rev-parse", "refs/heads/source"))
		tmp := t.TempDir()
		t.Setenv("TMPDIR", tmp)
		_, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{StrategyOptions: []string{"not-a-real-option"}})
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "unknown strategy option")
		assert.Equal(t, before, mergePreviewSnapshot(t, root), "refs, HEAD, index, worktree, and source object store must stay unchanged")
		entries, readErr := os.ReadDir(tmp)
		assert.NoError(t, readErr)
		assert.Empty(t, entries, "private preview object storage must be removed on errors")
	})
}

func TestBranchActionMergePreviewRejectsMergeDriverOutputBeforeResultTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell merge-driver fixture requires Unix")
	}
	root, _, _ := transactionHistory(t)
	writeTestFile(t, filepath.Join(root, ".gitattributes"), []byte("shared merge=loggy\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "shared"), []byte("base\n"), 0o644)
	gitTest(t, root, "add", ".gitattributes", "shared")
	gitTest(t, root, "commit", "-qm", "add custom-merged file")
	gitTest(t, root, "branch", "source")
	gitTest(t, root, "branch", "current")
	mergeFixtureCommit(t, root, "source", "shared", "source version\n", "source change")
	writeTestFile(t, filepath.Join(root, "incoming"), []byte("source addition\n"), 0o644)
	gitTest(t, root, "add", "incoming")
	gitTest(t, root, "commit", "-qm", "source addition")
	mergeFixtureCommit(t, root, "current", "shared", "current version\n", "current change")
	gitTest(t, root, "checkout", "-q", "current")

	currentTree := gitTest(t, root, "rev-parse", "HEAD^{tree}")
	driver := filepath.Join(t.TempDir(), "loggy-driver")
	writeTestFile(t, driver, []byte("#!/bin/sh\nprintf '%s\\0' '"+currentTree+"'\n"), 0o755)
	gitTest(t, root, "config", "merge.loggy.driver", driver)
	commands, target, before := mergePreviewCaptured(t, root, gitTest(t, root, "rev-parse", "refs/heads/source"))
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	paths, err := commands.PreviewBranchActionOrtMerge(target, BranchActionMergePreviewOptions{})
	if err != nil {
		assert.Contains(t, strings.ToLower(err.Error()), "merge preview")
	} else {
		assert.Contains(t, paths, "incoming", "a driver's valid-looking tree header must not hide result paths")
	}
	assert.Equal(t, before, mergePreviewSnapshot(t, root), "driver output must not cause repository writes")
	entries, readErr := os.ReadDir(tmp)
	assert.NoError(t, readErr)
	assert.Empty(t, entries, "private preview object storage must be removed")
}

func mergePreviewCaptured(t *testing.T, root, sourceOID string) (*BranchCommands, BranchActionTarget, string) {
	t.Helper()
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/source", sourceOID)
	assert.NoError(t, err)
	return commands, target, mergePreviewSnapshot(t, root)
}

func mergePreviewRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	initMergePreviewRepo(t, root)
	return root
}

func initMergePreviewRepo(t *testing.T, root string) {
	t.Helper()
	if output, err := exec.Command("git", "-C", root, "init", "-q", "-b", "master").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	gitTest(t, root, "config", "user.name", "Merge Preview Test")
	gitTest(t, root, "config", "user.email", "merge-preview@example.invalid")
}

func mergeFixtureCommit(t *testing.T, root, branch, path, content, message string) {
	t.Helper()
	gitTest(t, root, "checkout", "-q", branch)
	writeTestFile(t, filepath.Join(root, filepath.FromSlash(path)), []byte(content), 0o644)
	gitTest(t, root, "add", "--", path)
	gitTest(t, root, "commit", "-qm", message)
}

func mergePreviewSnapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	for _, args := range [][]string{{"symbolic-ref", "-q", "HEAD"}, {"ls-files", "--stage", "-z"}, {"for-each-ref", "--format=%(refname) %(objectname)"}, {"--no-optional-locks", "status", "--porcelain=v2", "--untracked-files=all"}} {
		b.WriteString(strings.Join(args, " "))
		b.WriteByte('\n')
		b.WriteString(gitTest(t, root, args...))
		b.WriteByte('\n')
	}
	gitDir := gitTest(t, root, "rev-parse", "--absolute-git-dir")
	for _, name := range []string{"HEAD", "index"} {
		b.WriteString(name)
		b.Write(mustRead(t, filepath.Join(gitDir, name)))
	}
	objects := gitTest(t, root, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	var objectFiles []string
	assert.NoError(t, filepath.WalkDir(objects, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			objectFiles = append(objectFiles, path)
		}
		return nil
	}))
	sort.Strings(objectFiles)
	for _, path := range objectFiles {
		rel, err := filepath.Rel(objects, path)
		assert.NoError(t, err)
		b.WriteString(rel)
		b.Write(mustRead(t, path))
	}
	var files []string
	assert.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path == filepath.Join(root, ".git") {
			return filepath.SkipDir
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	}))
	sort.Strings(files)
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		assert.NoError(t, err)
		b.WriteString(rel)
		b.Write(mustRead(t, path))
	}
	return b.String()
}
