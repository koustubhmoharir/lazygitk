package git_commands

import (
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRebaseIgnoredFilesApprovalAndFreshFingerprint(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target := selectedCommitRebaseCommit(t, root, "collision", "target\n", "target")
	gitTest(t, root, "checkout", "-q", "topic")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("collision\n"), 0o644)
	collision := filepath.Join(root, "collision")
	writeTestFile(t, collision, []byte("precious\n"), 0o600)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{"collision"}, plan.IgnoredFiles())
	plan.IgnoredFiles()[0] = "mutated"
	_, err = commands.SelectedCommitRebaseCommand(plan)
	assert.ErrorIs(t, err, ErrRebaseIgnoredApprovalRequired)
	approved := plan.WithIgnoredFilesApproved()
	cmd, err := commands.SelectedCommitRebaseCommand(approved)
	selectedRebaseRequireNoError(t, err)
	selectedRebaseRequireNoError(t, cmd.Run())
	contents, err := os.ReadFile(collision)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, "target\n", string(contents))
}

func TestRebaseIgnoredFilesAmbiguousTargetRefKeepsFirstRiskPath(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	topicOID := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target := selectedCommitRebaseCommit(t, root, "aaa-risk", "target\n", "first target path")
	gitTest(t, root, "update-ref", "refs/heads/"+target, topicOID)
	gitTest(t, root, "checkout", "-q", "topic")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("aaa-risk\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "aaa-risk"), []byte("user bytes\n"), 0o600)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{"aaa-risk"}, plan.IgnoredFiles())
	_, err = commands.SelectedCommitRebaseCommand(plan)
	assert.ErrorIs(t, err, ErrRebaseIgnoredApprovalRequired)
}

func TestRebaseIgnoredFilesANSIPathBytes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("control bytes are not valid in Windows filenames")
	}
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	gitTest(t, root, "checkout", "-q", baseBranch)
	path := "aaa-\x1b[31m-risk"
	target := selectedCommitRebaseCommit(t, root, path, "target\n", "ANSI path")
	gitTest(t, root, "checkout", "-q", "topic")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("aaa-*\n"), 0o644)
	writeTestFile(t, filepath.Join(root, path), []byte("user bytes\n"), 0o600)
	plan, err := selectedCommitRebaseCommands(t, root).PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{path}, plan.IgnoredFiles())
}

func TestRebaseIgnoredFilesChangedAfterApprovalRequiresReview(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target := selectedCommitRebaseCommit(t, root, "risk", "target\n", "target")
	gitTest(t, root, "checkout", "-q", "topic")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("risk\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "risk"), []byte("one"), 0o644)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	approved := plan.WithIgnoredFilesApproved()
	writeTestFile(t, filepath.Join(root, "risk"), []byte("two"), 0o644)
	_, err = commands.SelectedCommitRebaseCommand(approved)
	assert.ErrorIs(t, err, ErrSelectedCommitRebaseChanged)
}

func TestRebaseIgnoredFilesApprovalInvalidatedByRiskSetAndMode(t *testing.T) {
	for _, change := range []string{"new", "deleted", "mode", "type"} {
		t.Run(change, func(t *testing.T) {
			if change == "mode" && runtime.GOOS == "windows" {
				t.Skip("Windows does not reliably expose Unix permission mode changes")
			}
			root := initHistoricalRepo(t)
			selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
			baseBranch := gitTest(t, root, "branch", "--show-current")
			gitTest(t, root, "checkout", "-qb", "topic")
			selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
			gitTest(t, root, "checkout", "-q", baseBranch)
			target := selectedCommitRebaseCommit(t, root, "risk", "target\n", "target")
			gitTest(t, root, "checkout", "-q", "topic")
			writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("risk\nnew-risk\n"), 0o644)
			riskPath := filepath.Join(root, "risk")
			if change != "new" {
				writeTestFile(t, riskPath, []byte("risk"), 0o600)
			}
			commands := selectedCommitRebaseCommands(t, root)
			plan, err := commands.PrepareSelectedCommitRebase(target, false)
			selectedRebaseRequireNoError(t, err)
			if change == "new" {
				writeTestFile(t, riskPath, []byte("new"), 0o600)
			} else if change == "deleted" {
				if err := os.Remove(riskPath); err != nil {
					t.Fatal(err)
				}
			} else if change == "type" {
				if err := os.Remove(riskPath); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("destination", riskPath); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			} else if err := os.Chmod(riskPath, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err = commands.SelectedCommitRebaseCommand(plan.WithIgnoredFilesApproved())
			assert.ErrorIs(t, err, ErrSelectedCommitRebaseChanged)
		})
	}
}

func TestRebaseIgnoredFilesUnicodeNewlinePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows filenames cannot contain line feeds")
	}
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target := selectedCommitRebaseCommit(t, root, "雪\nfile", "target\n", "unicode newline path")
	gitTest(t, root, "checkout", "-q", "topic")
	writeTestFile(t, filepath.Join(root, ".git", "info", "exclude"), []byte("*\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "雪\nfile"), []byte("ignored bytes"), 0o600)
	plan, err := selectedCommitRebaseCommands(t, root).PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{"雪\nfile"}, plan.IgnoredFiles())
}

func TestRebaseIgnoredFilesSymlinkAndIgnoreCase(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic")
	gitTest(t, root, "checkout", "-q", baseBranch)
	target := selectedCommitRebaseCommit(t, root, "casepath", "case target\n", "case path")
	gitTest(t, root, "checkout", "-q", "topic")
	writeTestFile(t, filepath.Join(root, ".git", "info", "exclude"), []byte("*\n"), 0o644)
	if err := os.Symlink("destination 雪", filepath.Join(root, "CasePath")); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	gitTest(t, root, "config", "core.ignoreCase", "true")
	plan, err := selectedCommitRebaseCommands(t, root).PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{"CasePath"}, plan.IgnoredFiles())
}

func TestRebaseIgnoredFilesConservativeCandidatesAndUnrelatedIgnore(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "base", "base\n", "base")
	branch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	transientPath := "transient\n雪"
	if runtime.GOOS == "windows" {
		transientPath = "transient 雪"
	}
	selectedCommitRebaseCommit(t, root, transientPath, "once\n", "add transient")
	gitTest(t, root, "rm", "--", transientPath)
	gitTest(t, root, "commit", "-qm", "delete transient")
	gitTest(t, root, "checkout", "-q", branch)
	target := selectedCommitRebaseCommit(t, root, "target", "target\n", "target")
	gitTest(t, root, "checkout", "-q", "topic")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("transient*\nunrelated\n"), 0o644)
	writeTestFile(t, filepath.Join(root, transientPath), []byte("possibly gone"), 0o644)
	writeTestFile(t, filepath.Join(root, "unrelated"), []byte("safe"), 0o644)
	plan, err := selectedCommitRebaseCommands(t, root).PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{transientPath}, plan.IgnoredFiles())
}

func TestRebaseIgnoredFingerprintFramesFileContents(t *testing.T) {
	fingerprint := func(contents string) []byte {
		path := filepath.Join(t.TempDir(), "file")
		writeTestFile(t, path, []byte(contents), 0o644)
		info, err := os.Lstat(path)
		selectedRebaseRequireNoError(t, err)
		h := sha256.New()
		selectedRebaseRequireNoError(t, fingerprintRegularFile(h, path, info))
		contentHash := sha256.Sum256([]byte(contents))
		framedHash := sha256.Sum256(contentHash[:])
		assert.Equal(t, framedHash[:], h.Sum(nil), "record must contain a fixed-size content digest, not raw bytes")
		return h.Sum(nil)
	}
	first := fingerprint("abc")
	second := fingerprint("abcd")
	assert.Len(t, first, sha256.Size)
	assert.Len(t, second, sha256.Size)
	assert.NotEqual(t, first, second)
}

func TestRebaseIgnoredFilesRecursiveSubmoduleOverwriteProbe(t *testing.T) {
	root := initHistoricalRepo(t)
	childSource := filepath.Join(t.TempDir(), "child-source")
	if err := os.MkdirAll(childSource, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitAt(t, childSource, "init", "-q")
	runGitAt(t, childSource, "config", "user.name", "Test")
	runGitAt(t, childSource, "config", "user.email", "test@example.invalid")
	writeTestFile(t, filepath.Join(childSource, "base"), []byte("base\n"), 0o644)
	runGitAt(t, childSource, "add", "--", "base")
	runGitAt(t, childSource, "commit", "-qm", "child base")
	childBase := runGitAt(t, childSource, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(childSource, "collision"), []byte("target child bytes\n"), 0o644)
	runGitAt(t, childSource, "add", "--", "collision")
	runGitAt(t, childSource, "commit", "-qm", "introduce child collision")
	childTarget := runGitAt(t, childSource, "rev-parse", "HEAD")

	runGitAt(t, root, "-c", "protocol.file.allow=always", "submodule", "add", "-q", childSource, "child")
	runGitAt(t, filepath.Join(root, "child"), "checkout", "--detach", "-q", childBase)
	runGitAt(t, root, "add", "--", "child")
	runGitAt(t, root, "commit", "-qam", "add child")
	baseBranch := runGitAt(t, root, "branch", "--show-current")
	runGitAt(t, root, "checkout", "-qb", "topic")
	selectedCommitRebaseCommit(t, root, "topic", "topic\n", "topic commit")
	runGitAt(t, root, "checkout", "-q", baseBranch)
	runGitAt(t, filepath.Join(root, "child"), "checkout", "--detach", "-q", childTarget)
	runGitAt(t, root, "add", "--", "child")
	runGitAt(t, root, "commit", "-qm", "advance child")
	target := runGitAt(t, root, "rev-parse", "HEAD")
	runGitAt(t, root, "checkout", "-q", "topic")
	runGitAt(t, filepath.Join(root, "child"), "checkout", "--detach", "-q", childBase)
	runGitAt(t, filepath.Join(root, "child"), "checkout", "-qb", "child-topic")
	ignoreFile := runGitAt(t, filepath.Join(root, "child"), "rev-parse", "--path-format=absolute", "--git-path", "info/exclude")
	writeTestFile(t, ignoreFile, []byte("collision\n"), 0o644)
	collision := filepath.Join(root, "child", "collision")
	writeTestFile(t, collision, []byte("ignored child bytes\n"), 0o600)
	runGitAt(t, root, "config", "submodule.recurse", "true")

	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareSelectedCommitRebase(target, false)
	selectedRebaseRequireNoError(t, err)
	cmd, err := commands.SelectedCommitRebaseCommand(plan)
	selectedRebaseRequireNoError(t, err)
	selectedRebaseRequireNoError(t, cmd.Run())
	contents, err := os.ReadFile(collision)
	selectedRebaseRequireNoError(t, err)
	childHead := runGitAt(t, filepath.Join(root, "child"), "rev-parse", "HEAD")
	assert.Equal(t, childBase, childHead, "native parent rebase leaves the child checkout at its original gitlink")
	assert.Equal(t, "ignored child bytes\n", string(contents), "the ignored child collision remains unchanged")
}

func runGitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, output)
	}
	return strings.TrimSuffix(string(output), "\n")
}

func TestRebasePathCollisionAndNULPathParsing(t *testing.T) {
	for _, test := range []struct {
		path          string
		candidates    []string
		wantCollision bool
	}{
		{"cache", []string{"cache/file"}, true},
		{"cache/file/deeper", []string{"cache"}, true},
		{"cache/a", []string{"cache/a"}, true},
		{"cache2", []string{"cache"}, false},
	} {
		candidateSet := make(map[string]struct{}, len(test.candidates))
		for _, candidate := range test.candidates {
			candidateSet[candidate] = struct{}{}
		}
		assert.Equal(t, test.wantCollision, rebaseCandidatePathCollides(test.path, candidateSet, test.candidates))
	}
	assert.Equal(t, []string{"line\nbreak", "雪"}, nulPaths([]byte("line\nbreak\x00雪\x00")))
}
