package git_commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func testHistoricalDiffCommands(t *testing.T) *DiffCommands {
	t.Helper()
	root := t.TempDir()
	inner := oscommands.NewDummyCmdObjBuilder(oscommands.NewFakeRunner(t))
	common := NewGitCommon(nil, nil, rootedTestBuilder{inner: inner, root: root}, nil, MockRepoPaths(root), nil, nil)
	return NewDiffCommands(common)
}

type rootedTestBuilder struct {
	inner oscommands.ICmdObjBuilder
	root  string
}

func (builder rootedTestBuilder) New(args []string) *oscommands.CmdObj {
	return builder.inner.New(args).SetWd(builder.root)
}

func (builder rootedTestBuilder) NewShell(command, functionsFile string) *oscommands.CmdObj {
	return builder.inner.NewShell(command, functionsFile).SetWd(builder.root)
}

func (builder rootedTestBuilder) Quote(value string) string { return builder.inner.Quote(value) }

func realHistoricalDiffCommands(t *testing.T, root string) *DiffCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	paths := MockRepoPaths(root)
	return NewDiffCommands(NewGitCommon(nil, nil, osCommand.Cmd, osCommand, paths, nil, nil))
}

func gitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func initHistoricalRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitTest(t, root, "init", "-q")
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	return root
}

func writeTestFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

// The Git configured difftool runs this test binary as its fake external tool.
func TestHistoricalDifftoolHelper(t *testing.T) {
	if os.Getenv("LAZYGIT_HISTORICAL_DIFF_HELPER") != "1" {
		return
	}
	if os.Getenv("LAZYGIT_HISTORICAL_DIFF_EXIT") != "" {
		code, _ := strconv.Atoi(os.Getenv("LAZYGIT_HISTORICAL_DIFF_EXIT"))
		os.Exit(code)
	}
	leftPath, rightPath := os.Args[len(os.Args)-2], os.Args[len(os.Args)-1]
	left, err := os.ReadFile(leftPath)
	if err != nil {
		os.Exit(92)
	}
	right, err := os.ReadFile(rightPath)
	if err != nil {
		os.Exit(93)
	}
	log, err := json.Marshal(struct {
		LeftPath  string
		Left      []byte
		RightPath string
		Right     []byte
	}{leftPath, left, rightPath, right})
	if err != nil {
		os.Exit(94)
	}
	if err := os.WriteFile(os.Getenv("LAZYGIT_HISTORICAL_DIFF_LOG"), log, 0o600); err != nil {
		os.Exit(94)
	}
}

func runHistoricalTool(t *testing.T, diff *HistoricalFileDiff, logPath string, toolExit string) error {
	t.Helper()
	diff.Command.AddEnvVars("LAZYGIT_HISTORICAL_DIFF_HELPER=1", "LAZYGIT_HISTORICAL_DIFF_LOG="+logPath, "LAZYGIT_HISTORICAL_DIFF_EXIT="+toolExit)
	runErr := diff.Command.GetCmd().Run()
	return diff.Command.SubprocessError(runErr)
}

func TestPrepareHistoricalFileDiffUsesBlobBytesAndConfiguredTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test filename contains a literal backslash")
	}
	root := initHistoricalRepo(t)
	historicalPath := "dir\\odd:\tline\nbreak.bin"
	historicalBytes := []byte{0, 1, 2, 0xff, '\n'}
	writeTestFile(t, filepath.Join(root, filepath.FromSlash(historicalPath)), historicalBytes, 0o755)
	gitTest(t, root, "--literal-pathspecs", "add", "--", historicalPath)
	gitTest(t, root, "commit", "-qm", "source")
	commit := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "config", "core.filemode", "true")
	gitTest(t, root, "config", "diff.tool", "historical-probe")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "config", "difftool.historical-probe.cmd", fmt.Sprintf("%q -test.run=TestHistoricalDifftoolHelper \"$LOCAL\" \"$REMOTE\"", binary))
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored-target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "ignored-target")
	destinationBytes := []byte("actual untracked ignored destination\x00")
	writeTestFile(t, destination, destinationBytes, 0o600)
	logPath := filepath.Join(t.TempDir(), "tool-log")

	diff, err := realHistoricalDiffCommands(t, root).PrepareHistoricalFileDiff(commit, historicalPath, "ignored-target")
	if assert.NoError(t, err) && diff != nil {
		defer func() { assert.NoError(t, diff.Close()); assert.NoError(t, diff.Close()) }()
		assert.Equal(t, root, diff.Command.GetCmd().Dir)
		assert.Equal(t, destination, diff.DestinationPath)
		got, readErr := os.ReadFile(diff.SourcePath)
		assert.NoError(t, readErr)
		assert.Equal(t, historicalBytes, got)
		info, statErr := os.Stat(diff.SourcePath)
		assert.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o400), info.Mode().Perm())
		assert.NoError(t, runHistoricalTool(t, diff, logPath, ""))
		logged, readErr := os.ReadFile(logPath)
		assert.NoError(t, readErr)
		var recorded struct {
			LeftPath  string
			Left      []byte
			RightPath string
			Right     []byte
		}
		assert.NoError(t, json.Unmarshal(logged, &recorded))
		assert.Equal(t, diff.SourcePath, recorded.LeftPath)
		assert.Equal(t, historicalBytes, recorded.Left)
		assert.Equal(t, destination, recorded.RightPath)
		assert.Equal(t, destinationBytes, recorded.Right)
		assert.Equal(t, []string{"git", "difftool", "--no-index", "--no-prompt", "--trust-exit-code", "--", diff.SourcePath, destination}, diff.Command.Args())
		failedDiff, prepareErr := realHistoricalDiffCommands(t, root).PrepareHistoricalFileDiff(commit, historicalPath, "ignored-target")
		if assert.NoError(t, prepareErr) {
			failure := runHistoricalTool(t, failedDiff, logPath, "7")
			assert.Error(t, failure)
			var exitError *exec.ExitError
			if assert.ErrorAs(t, failure, &exitError) {
				assert.Equal(t, 128, exitError.ExitCode())
			}
			assert.NoError(t, failedDiff.Close())
			assert.NoError(t, failedDiff.Close())
			_, statErr := os.Stat(failedDiff.SourcePath)
			assert.ErrorIs(t, statErr, os.ErrNotExist)
		}
		_, statErr = os.Stat(diff.SourcePath)
		assert.NoError(t, statErr) // source remains until explicit Close
	}
	assert.Equal(t, destinationBytes, mustReadFile(t, destination))
	assert.Equal(t, historicalBytes, mustReadFile(t, filepath.Join(root, filepath.FromSlash(historicalPath))))
	assert.Equal(t, commit, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Empty(t, gitTest(t, root, "diff", "--cached", "--name-only"))
	assert.Contains(t, gitTest(t, root, "status", "--ignored", "--short"), "ignored-target")
}

func TestResolveHistoricalDiffCommitRootAndMerge(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "root"), []byte("root"), 0o600)
	gitTest(t, root, "add", "root")
	gitTest(t, root, "commit", "-qm", "root")
	rootOID := gitTest(t, root, "rev-parse", "HEAD")
	commands := realHistoricalDiffCommands(t, root)
	resolved, err := commands.ResolveHistoricalDiffCommit(rootOID)
	if assert.NoError(t, err) {
		assert.Equal(t, rootOID, resolved.OID)
		assert.Empty(t, resolved.FirstParent)
	}
	gitTest(t, root, "branch", "side")
	writeTestFile(t, filepath.Join(root, "main"), []byte("main"), 0o600)
	gitTest(t, root, "add", "main")
	gitTest(t, root, "commit", "-qm", "main")
	firstParent := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", "side")
	writeTestFile(t, filepath.Join(root, "side"), []byte("side"), 0o600)
	gitTest(t, root, "add", "side")
	gitTest(t, root, "commit", "-qm", "side")
	gitTest(t, root, "checkout", "-q", "-")
	gitTest(t, root, "merge", "--no-ff", "-qm", "merge", "side")
	merge := gitTest(t, root, "rev-parse", "HEAD")
	resolved, err = commands.ResolveHistoricalDiffCommit(merge)
	if assert.NoError(t, err) {
		assert.Equal(t, merge, resolved.OID)
		assert.Equal(t, firstParent, resolved.FirstParent)
	}
}

func TestPrepareHistoricalFileDiffRejectsMissingDeletedAndSymlinkPaths(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "deleted"), []byte("old"), 0o600)
	gitTest(t, root, "add", "deleted")
	gitTest(t, root, "commit", "-qm", "add")
	commit := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "mv", "deleted", "renamed")
	gitTest(t, root, "commit", "-qm", "rename")
	renameCommit := gitTest(t, root, "rev-parse", "HEAD")
	commands := realHistoricalDiffCommands(t, root)
	destination := filepath.Join(root, "dest")
	writeTestFile(t, destination, []byte("dest"), 0o600)
	renameDiff, err := commands.PrepareHistoricalFileDiff(renameCommit, "renamed", destination)
	if assert.NoError(t, err) && renameDiff != nil {
		assert.NoError(t, renameDiff.Close())
	}
	gitTest(t, root, "rm", "renamed")
	gitTest(t, root, "commit", "-qm", "delete")
	_, err = commands.PrepareHistoricalFileDiff(commit, "deleted", filepath.Join(root, "missing"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = commands.PrepareHistoricalFileDiff(gitTest(t, root, "rev-parse", "HEAD"), "deleted", destination)
	assert.Error(t, err)
	if err := os.Symlink(destination, filepath.Join(root, "link")); err == nil {
		_, err = commands.PrepareHistoricalFileDiff(commit, "deleted", "link")
		assert.Error(t, err)
	}
	_, err = commands.PrepareHistoricalFileDiff(commit, "deleted", root)
	assert.Error(t, err)
}

func TestPrepareHistoricalFileDiffCatFailureCleansPrivateDirectory(t *testing.T) {
	root := initHistoricalRepo(t)
	treePath := "blob"
	writeTestFile(t, filepath.Join(root, treePath), []byte("blob bytes"), 0o600)
	gitTest(t, root, "add", treePath)
	gitTest(t, root, "commit", "-qm", "blob")
	commit := gitTest(t, root, "rev-parse", "HEAD")
	blobOID := gitTest(t, root, "rev-parse", "HEAD:blob")
	objectPath := filepath.Join(root, ".git", "objects", blobOID[:2], blobOID[2:])
	if err := os.Remove(objectPath); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "destination")
	writeTestFile(t, destination, []byte("destination"), 0o600)
	tempRoot := t.TempDir()
	osCommand := oscommands.NewDummyOSCommandWithDeps(oscommands.OSCommandDeps{TempDir: tempRoot})
	osCommand.Cmd = oscommands.NewDummyOSCommand().Cmd
	commands := NewDiffCommands(NewGitCommon(nil, nil, osCommand.Cmd, osCommand, MockRepoPaths(root), nil, nil))
	_, err := commands.PrepareHistoricalFileDiff(commit, treePath, destination)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cat-file")
	entries, readErr := os.ReadDir(tempRoot)
	assert.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestPrepareHistoricalFileDiffRejectsSymlinkAndGitlinkModes(t *testing.T) {
	root := initHistoricalRepo(t)
	target := filepath.Join(root, "target")
	writeTestFile(t, target, []byte("target"), 0o600)
	if err := os.Symlink("target", filepath.Join(root, "symlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	gitTest(t, root, "add", "symlink", "target")
	gitTest(t, root, "commit", "-qm", "symlink")
	symlinkCommit := gitTest(t, root, "rev-parse", "HEAD")
	destination := filepath.Join(root, "destination")
	writeTestFile(t, destination, []byte("destination"), 0o600)
	commands := realHistoricalDiffCommands(t, root)
	_, err := commands.PrepareHistoricalFileDiff(symlinkCommit, "symlink", destination)
	assert.ErrorContains(t, err, "symlinks")

	gitTest(t, root, "update-index", "--add", "--cacheinfo", "160000,"+symlinkCommit+",gitlink")
	gitTest(t, root, "commit", "-qm", "gitlink")
	gitlinkCommit := gitTest(t, root, "rev-parse", "HEAD")
	_, err = commands.PrepareHistoricalFileDiff(gitlinkCommit, "gitlink", destination)
	assert.ErrorContains(t, err, "gitlinks")
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestHistoricalRevisionDiffCmdObjArguments(t *testing.T) {
	commands := testHistoricalDiffCommands(t)
	commit := HistoricalDiffCommit{OID: "0123456789abcdef0123456789abcdef01234567", FirstParent: "abcdef0123456789abcdef0123456789abcdef01"}
	paths := []string{"-leading", "odd\tname", "line\nbreak", ":(glob)name"}
	if runtime.GOOS != "windows" {
		paths = append(paths, "colon:name")
	}

	previous, err := commands.HistoricalRevisionDiffCmdObj(commit, true, paths, true)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	expectedPaths := []string{"-leading", "odd\tname", "line\nbreak", ":(glob)name"}
	if runtime.GOOS != "windows" {
		expectedPaths = append(expectedPaths, "colon:name")
	}
	assert.Equal(t, append([]string{"git", "--literal-pathspecs", "difftool", "--no-prompt", "--dir-diff", commit.FirstParent, commit.OID, "--"}, expectedPaths...), previous.Args())
	working, err := commands.HistoricalRevisionDiffCmdObj(commit, false, nil, false)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", commit.OID, "--", "."}, working.Args())
	assert.Equal(t, commands.repoPaths.WorktreePath(), working.GetCmd().Dir)
}

func TestHistoricalRevisionDiffRejectsInvalidEndpointsAndPaths(t *testing.T) {
	commands := testHistoricalDiffCommands(t)
	commit := HistoricalDiffCommit{OID: "0123456789abcdef0123456789abcdef01234567"}
	_, err := commands.HistoricalRevisionDiffCmdObj(commit, true, nil, false)
	assert.Error(t, err)
	for _, path := range []string{"/absolute", "../escape", "a/../../escape", "nul\x00path"} {
		_, err := commands.HistoricalRevisionDiffCmdObj(commit, false, []string{path}, false)
		assert.Error(t, err, path)
	}
	for _, oid := range []string{"HEAD", "", "0123", "0123456789abcdef0123456789abcdef0123456g"} {
		_, err := commands.HistoricalRevisionDiffCmdObj(HistoricalDiffCommit{OID: oid}, false, nil, false)
		assert.Error(t, err, oid)
	}
}

func TestParseHistoricalTreeEntryUsesExactNULTerminatedPath(t *testing.T) {
	path := "nested/tab\tand-newline\nname"
	output := []byte("100644 blob 0123456789abcdef0123456789abcdef01234567\tother\x00" +
		"100644 blob abcdef0123456789abcdef0123456789abcdef01\t" + path + "\x00")
	mode, typ, oid, err := parseTreeEntry(output, path)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, "100644", mode)
	assert.Equal(t, "blob", typ)
	assert.Equal(t, "abcdef0123456789abcdef0123456789abcdef01", oid)

	_, _, _, err = parseTreeEntry(output, "missing")
	assert.Error(t, err)
}

func TestDestinationPathValidation(t *testing.T) {
	root := t.TempDir()
	for _, destination := range []string{"../outside", "/outside", "", "nul\x00path", filepath.Join("nested", ".git", "file"), filepath.Join("nested", ".GiT", "file")} {
		_, err := resolveDestination(root, destination)
		assert.Error(t, err, destination)
	}
	for _, path := range []string{"", "/absolute", "../up", "a/../b"} {
		assert.False(t, validTreePath(path), path)
	}
	assert.Equal(t, runtime.GOOS != "windows", validTreePath("a\\b"))
	assert.Equal(t, runtime.GOOS != "windows", validTreePath("nested/unusual: name\tfile"))
	assert.Equal(t, runtime.GOOS != "windows", validTreePath("C:relative"))

	missing := filepath.Join(root, "missing")
	_, err := resolveDestination(root, missing)
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestHistoricalDiffRejectsNonstandardGitMetadataDestinations(t *testing.T) {
	for _, commonDir := range []bool{false, true} {
		t.Run(fmt.Sprintf("common-dir=%t", commonDir), func(t *testing.T) {
			commands := testHistoricalDiffCommands(t)
			metadata := filepath.Join(commands.repoPaths.WorktreePath(), "repository-storage")
			if commonDir {
				commands.repoPaths.repoGitDirPath = metadata
			} else {
				commands.repoPaths.worktreeGitDirPath = metadata
			}
			destination := filepath.Join(metadata, "config")
			writeTestFile(t, destination, []byte("Git configuration"), 0o600)
			_, err := commands.PrepareHistoricalFileDiff("0123456789abcdef0123456789abcdef01234567", "source", destination)
			assert.ErrorContains(t, err, "Git metadata")
		})
	}
}
