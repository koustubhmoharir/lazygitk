package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const capturedCherryPickDate = "2020-01-01T00:00:00Z"

func TestCapturedCherryPickOrdersByGitAncestryAndCopiesPlanData(t *testing.T) {
	root := initHistoricalRepo(t)
	base := capturedCherryPickCommit(t, root, "base.txt", "base\n", "base", "2000-01-01T00:00:00Z")
	first := capturedCherryPickCommit(t, root, "first.txt", "first\n", "first", "2030-01-01T00:00:00Z")
	second := capturedCherryPickCommit(t, root, "second.txt", "second\n", "second", "2001-01-01T00:00:00Z")
	commands := capturedCherryPickCommands(t, root)
	input := []string{second, first, first}
	indexBefore, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	capturedCherryPickRequireNoError(t, err)
	refsBefore := gitTest(t, root, "show-ref")
	plan, err := commands.PrepareCherryPickPlan(input)
	capturedCherryPickRequireNoError(t, err)
	assert.Equal(t, []string{second, first, first}, input)
	assert.Equal(t, base, plan.Commits()[0].Parents[0])
	assert.Equal(t, []string{first, second}, capturedCherryPickOIDs(plan))
	assert.Equal(t, "first", plan.Commits()[0].Subject)
	assert.Equal(t, second, plan.HeadOID())
	assert.Equal(t, "refs/heads/"+gitTest(t, root, "branch", "--show-current"), plan.HeadRef())
	commits := plan.Commits()
	commits[0].OID = "changed"
	commits[0].Subject = "changed"
	commits[0].Parents[0] = "changed"
	assert.Equal(t, first, plan.Commits()[0].OID)
	assert.Equal(t, "first", plan.Commits()[0].Subject)
	assert.Equal(t, base, plan.Commits()[0].Parents[0])
	assert.False(t, plan.ignoredApproved)
	assert.True(t, plan.WithIgnoredFilesApproved().ignoredApproved)
	assert.False(t, plan.ignoredApproved)
	assert.Equal(t, 0, plan.Mainline())
	_, err = plan.WithMainline(1)
	assert.Error(t, err)
	_, err = plan.WithMainline(0)
	capturedCherryPickRequireNoError(t, err)
	indexAfter, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	capturedCherryPickRequireNoError(t, err)
	assert.Equal(t, indexBefore, indexAfter)
	assert.Equal(t, refsBefore, gitTest(t, root, "show-ref"))
	assert.Empty(t, gitTest(t, root, "status", "--porcelain"))
}

func TestCapturedCherryPickUnrelatedTipsHaveStableReviewOrder(t *testing.T) {
	root := initHistoricalRepo(t)
	base := capturedCherryPickCommit(t, root, "base", "base\n", "base", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "left")
	left := capturedCherryPickCommit(t, root, "left", "left\n", "left", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "right", base)
	right := capturedCherryPickCommit(t, root, "right", "right\n", "right", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "destination", base)
	commands := capturedCherryPickCommands(t, root)
	plan, err := commands.PrepareCherryPickPlan([]string{right, left})
	capturedCherryPickRequireNoError(t, err)
	for _, input := range [][]string{{left, right}, {right, left, right}, capturedCherryPickOIDs(plan)} {
		fresh, err := commands.PrepareCherryPickPlan(input)
		capturedCherryPickRequireNoError(t, err)
		assert.Equal(t, plan.Commits(), fresh.Commits())
		cmd, err := commands.CherryPickCommand(fresh)
		capturedCherryPickRequireNoError(t, err)
		assert.Equal(t, append(capturedCherryPickArgv(commands.version), capturedCherryPickOIDs(plan)...), cmd.GetCmd().Args)
	}
	cmd, err := commands.CherryPickCommand(plan)
	capturedCherryPickRequireNoError(t, err)
	capturedCherryPickRequireNoError(t, cmd.Run())
	assert.Equal(t, "left", gitTest(t, root, "show", "HEAD:left"))
	assert.Equal(t, "right", gitTest(t, root, "show", "HEAD:right"))
}

func TestCapturedCherryPickCommandVersionsAndStaleHeadOwner(t *testing.T) {
	root, base, first, _ := capturedCherryPickReplayFixture(t)
	commands := capturedCherryPickCommands(t, root)
	actualVersion := commands.version
	plan, err := commands.PrepareCherryPickPlan([]string{first})
	capturedCherryPickRequireNoError(t, err)
	for _, version := range []*GitVersion{{Major: 2, Minor: 44, Patch: 9}, {Major: 2, Minor: 45}} {
		commands.version = version
		cmd, err := commands.CherryPickCommand(plan)
		capturedCherryPickRequireNoError(t, err)
		assert.Equal(t, append(capturedCherryPickArgv(version), first), cmd.GetCmd().Args)
		assert.True(t, cmd.LockRetriesDisabled())
		assert.Equal(t, root, cmd.GetCmd().Dir)
	}
	commands.version = actualVersion
	// Moving the source branch does not change the captured literal object.
	gitTest(t, root, "update-ref", "refs/heads/source", base)
	cmd, err := commands.CherryPickCommand(plan)
	capturedCherryPickRequireNoError(t, err)
	otherOwner := capturedCherryPickCommands(t, root)
	_, err = otherOwner.CherryPickCommand(plan)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	_, err = commands.CherryPickCommand(CherryPickPlan{})
	assert.Error(t, err)
	gitTest(t, root, "checkout", "-qb", "same-tip", base)
	_, err = commands.CherryPickCommand(plan)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	gitTest(t, root, "checkout", "-q", "destination")
	capturedCherryPickRequireNoError(t, cmd.Run())
	assert.Equal(t, "source", gitTest(t, root, "show", "HEAD:collision.txt"))
	assert.Equal(t, base, gitTest(t, root, "rev-parse", "HEAD^"))
	_, err = commands.CherryPickCommand(plan)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
}

func TestCapturedCherryPickLiteralObjectsAndHexRefShadows(t *testing.T) {
	root, base, first, second := capturedCherryPickReplayFixture(t)
	commands := capturedCherryPickCommands(t, root)
	gitTest(t, root, "tag", "-a", "-m", "annotated", "annotated", first)
	blob := gitTest(t, root, "rev-parse", base+":collision.txt")
	tree := gitTest(t, root, "rev-parse", base+"^{tree}")
	tag := gitTest(t, root, "rev-parse", "refs/tags/annotated")
	for _, input := range []string{"HEAD", "source", first[:12], first + "^0", "--all", strings.Repeat("z", 40), strings.Repeat("0", 40), blob, tree, tag} {
		_, err := commands.PrepareCherryPickPlan([]string{input})
		assert.Error(t, err, "input %q", input)
	}
	_, err := commands.PrepareCherryPickPlan(nil)
	assert.Error(t, err)
	gitTest(t, root, "update-ref", "refs/heads/"+first, second)
	plan, err := commands.PrepareCherryPickPlan([]string{strings.ToUpper(first), first})
	capturedCherryPickRequireNoError(t, err)
	assert.Equal(t, []string{first}, capturedCherryPickOIDs(plan))
	assert.Equal(t, "source change", plan.Commits()[0].Subject)
	assert.Equal(t, []string{base}, plan.Commits()[0].Parents)
	cmd, err := commands.CherryPickCommand(plan)
	capturedCherryPickRequireNoError(t, err)
	capturedCherryPickRequireNoError(t, cmd.Run())
	assert.Equal(t, "source", gitTest(t, root, "show", "HEAD:collision.txt"))
	_, err = os.Stat(filepath.Join(root, "remaining.txt"))
	assert.True(t, os.IsNotExist(err))
	// A hex ref cannot authorize an object that does not exist, including a
	// full SHA-256-looking ref in this SHA-1 repository.
	for _, absent := range []string{strings.Repeat("1", 40), strings.Repeat("2", 64)} {
		gitTest(t, root, "update-ref", "refs/heads/"+absent, second)
		_, err := commands.PrepareCherryPickPlan([]string{absent})
		assert.Error(t, err)
	}
}

func TestCapturedCherryPickSHA256(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "-C", root, "init", "-q", "--object-format=sha256").CombinedOutput(); err != nil {
		t.Skipf("Git does not support SHA-256 repositories: %v: %s", err, output)
	}
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	base := capturedCherryPickCommit(t, root, "base", "base\n", "base", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "source")
	first := capturedCherryPickCommit(t, root, "first", "first\n", "first", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "second", "second\n", "second", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "destination", base)
	gitTest(t, root, "update-ref", "refs/heads/"+first, second)
	commands := capturedCherryPickCommands(t, root)
	plan, err := commands.PrepareCherryPickPlan([]string{strings.ToUpper(first)})
	capturedCherryPickRequireNoError(t, err)
	assert.Len(t, first, 64)
	assert.Equal(t, []string{first}, capturedCherryPickOIDs(plan))
	assert.Equal(t, []string{base}, plan.Commits()[0].Parents)
	assert.Equal(t, "first", plan.Commits()[0].Subject)
	cmd, err := commands.CherryPickCommand(plan)
	capturedCherryPickRequireNoError(t, err)
	capturedCherryPickRequireNoError(t, cmd.Run())
	assert.Equal(t, "first", gitTest(t, root, "show", "HEAD:first"))
	gitTest(t, root, "update-ref", "refs/heads/"+strings.Repeat("3", 40), second)
	_, err = commands.PrepareCherryPickPlan([]string{strings.Repeat("3", 40)})
	assert.Error(t, err)
}

func TestCapturedCherryPickDetachedAndUnborn(t *testing.T) {
	t.Run("detached", func(t *testing.T) {
		root, base, first, _ := capturedCherryPickReplayFixture(t)
		gitTest(t, root, "checkout", "--detach", "-q", base)
		commands := capturedCherryPickCommands(t, root)
		plan, err := commands.PrepareCherryPickPlan([]string{first})
		capturedCherryPickRequireNoError(t, err)
		assert.Empty(t, plan.HeadRef())
		assert.Equal(t, base, plan.HeadOID())
		cmd, err := commands.CherryPickCommand(plan)
		capturedCherryPickRequireNoError(t, err)
		capturedCherryPickRequireNoError(t, cmd.Run())
		assert.Empty(t, gitTest(t, root, "branch", "--show-current"))
		assert.Equal(t, "source", gitTest(t, root, "show", "HEAD:collision.txt"))
	})
	for _, dirty := range []bool{false, true} {
		name := "unborn"
		if dirty {
			name = "unborn with colliding staged file"
		}
		t.Run(name, func(t *testing.T) {
			root := initHistoricalRepo(t)
			source := capturedCherryPickCommit(t, root, "file", "source\n", "root commit", capturedCherryPickDate)
			gitTest(t, root, "checkout", "--orphan", "unborn")
			gitTest(t, root, "rm", "-rf", ".")
			if dirty {
				writeTestFile(t, filepath.Join(root, "file"), []byte("local\n"), 0o644)
				gitTest(t, root, "add", "file")
			}
			commands := capturedCherryPickCommands(t, root)
			plan, err := commands.PrepareCherryPickPlan([]string{source})
			capturedCherryPickRequireNoError(t, err)
			assert.Equal(t, "refs/heads/unborn", plan.HeadRef())
			assert.Empty(t, plan.HeadOID())
			cmd, err := commands.CherryPickCommand(plan)
			capturedCherryPickRequireNoError(t, err)
			if dirty {
				assert.Error(t, cmd.Run())
				assert.Equal(t, "local", gitTest(t, root, "show", ":file"))
				assert.Error(t, exec.Command("git", "-C", root, "rev-parse", "--verify", "HEAD").Run())
			} else {
				capturedCherryPickRequireNoError(t, cmd.Run())
				assert.Equal(t, "source", gitTest(t, root, "show", "HEAD:file"))
				assert.Equal(t, "unborn", gitTest(t, root, "branch", "--show-current"))
			}
			capturedCherryPickAssertNoStash(t, root)
		})
	}
}

func TestCapturedCherryPickActiveOperationsAndBare(t *testing.T) {
	root, _, first, _ := capturedCherryPickReplayFixture(t)
	commands := capturedCherryPickCommands(t, root)
	plan, err := commands.PrepareCherryPickPlan([]string{first})
	capturedCherryPickRequireNoError(t, err)
	for _, marker := range []string{"rebase-merge", "rebase-apply", "MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "sequencer", "BISECT_START"} {
		t.Run(marker, func(t *testing.T) {
			path := filepath.Join(root, ".git", marker)
			if marker == "rebase-merge" || marker == "rebase-apply" || marker == "sequencer" {
				capturedCherryPickRequireNoError(t, os.Mkdir(path, 0o755))
			} else {
				writeTestFile(t, path, []byte(first+"\n"), 0o644)
			}
			defer func() { capturedCherryPickRequireNoError(t, os.RemoveAll(path)) }()
			_, err := commands.PrepareCherryPickPlan([]string{first})
			assert.Error(t, err)
			_, err = commands.CherryPickCommand(plan)
			assert.Error(t, err)
		})
	}
	bare := filepath.Join(t.TempDir(), "bare.git")
	gitTest(t, root, "clone", "--bare", "-q", root, bare)
	_, err = capturedCherryPickCommands(t, bare).PrepareCherryPickPlan([]string{first})
	assert.ErrorContains(t, err, "bare")
}

func TestCapturedCherryPickDirtyFilesNativeProtectionAndNoStash(t *testing.T) {
	for _, path := range []string{"unrelated.txt", "collision.txt"} {
		t.Run(path, func(t *testing.T) {
			root, base, first, _ := capturedCherryPickReplayFixture(t)
			gitTest(t, root, "config", "rebase.autoStash", "true")
			gitTest(t, root, "config", "merge.autoStash", "true")
			writeTestFile(t, filepath.Join(root, path), []byte("local edits\n"), 0o644)
			commands := capturedCherryPickCommands(t, root)
			plan, err := commands.PrepareCherryPickPlan([]string{first})
			capturedCherryPickRequireNoError(t, err)
			cmd, err := commands.CherryPickCommand(plan)
			capturedCherryPickRequireNoError(t, err)
			if path == "collision.txt" {
				assert.Error(t, cmd.Run())
				assert.Equal(t, base, gitTest(t, root, "rev-parse", "HEAD"))
				assert.Equal(t, "base", gitTest(t, root, "show", ":collision.txt"))
			} else {
				capturedCherryPickRequireNoError(t, cmd.Run())
				assert.Equal(t, "source", gitTest(t, root, "show", "HEAD:collision.txt"))
				assert.Equal(t, " M unrelated.txt\n", capturedCherryPickStdout(t, root, "status", "--porcelain"))
			}
			contents, err := os.ReadFile(filepath.Join(root, path))
			capturedCherryPickRequireNoError(t, err)
			assert.Equal(t, "local edits\n", string(contents))
			capturedCherryPickAssertNoStash(t, root)
		})
	}
}

func TestCapturedCherryPickConflictLeavesNativeSequence(t *testing.T) {
	root, _, first, second := capturedCherryPickReplayFixture(t)
	destination := capturedCherryPickCommit(t, root, "collision.txt", "destination\n", "destination change", capturedCherryPickDate)
	commands := capturedCherryPickCommands(t, root)
	plan, err := commands.PrepareCherryPickPlan([]string{second, first})
	capturedCherryPickRequireNoError(t, err)
	assert.Equal(t, []string{first, second}, capturedCherryPickOIDs(plan))
	cmd, err := commands.CherryPickCommand(plan)
	capturedCherryPickRequireNoError(t, err)
	assert.Error(t, cmd.Run())
	assert.Equal(t, destination, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "CHERRY_PICK_HEAD"))
	assert.NotEmpty(t, gitTest(t, root, "ls-files", "-u"))
	todo, err := os.ReadFile(filepath.Join(root, ".git", "sequencer", "todo"))
	capturedCherryPickRequireNoError(t, err)
	remaining := false
	for _, line := range strings.Split(string(todo), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "pick" && strings.HasPrefix(second, fields[1]) {
			remaining = true
		}
	}
	assert.True(t, remaining, "native sequencer must retain second commit: %s", todo)
	capturedCherryPickAssertNoStash(t, root)
}

func TestCapturedCherryPickMergeRequiresAndUsesExplicitMainline(t *testing.T) {
	for _, parent := range []int{1, 2} {
		name := "first parent"
		if parent == 2 {
			name = "second parent"
		}
		t.Run(name, func(t *testing.T) {
			root := initHistoricalRepo(t)
			base := capturedCherryPickCommit(t, root, "base", "base\n", "base", capturedCherryPickDate)
			gitTest(t, root, "checkout", "-qb", "merge-main")
			main := capturedCherryPickCommit(t, root, "main.txt", "main\n", "main", capturedCherryPickDate)
			gitTest(t, root, "checkout", "-qb", "side", base)
			side := capturedCherryPickCommit(t, root, "side.txt", "side\n", "side", capturedCherryPickDate)
			gitTest(t, root, "checkout", "-q", "merge-main")
			gitTest(t, root, "merge", "--no-ff", "-qm", "merge subject", "side")
			merge := gitTest(t, root, "rev-parse", "HEAD")
			gitTest(t, root, "checkout", "-qb", "destination", base)
			commands := capturedCherryPickCommands(t, root)
			plan, err := commands.PrepareCherryPickPlan([]string{merge})
			capturedCherryPickRequireNoError(t, err)
			assert.Equal(t, "merge subject", plan.Commits()[0].Subject)
			assert.Equal(t, []string{main, side}, plan.Commits()[0].Parents)
			_, err = commands.CherryPickCommand(plan)
			assert.ErrorIs(t, err, ErrCherryPickMainlineRequired)
			for _, invalid := range []int{-1, 0, 3} {
				_, err := plan.WithMainline(invalid)
				assert.Error(t, err)
			}
			approved, err := plan.WithMainline(parent)
			capturedCherryPickRequireNoError(t, err)
			assert.Equal(t, 0, plan.Mainline())
			assert.Equal(t, parent, approved.Mainline())
			cmd, err := commands.CherryPickCommand(approved)
			capturedCherryPickRequireNoError(t, err)
			option, picked, absent := "-m1", "side.txt", "main.txt"
			if parent == 2 {
				option, picked, absent = "-m2", "main.txt", "side.txt"
			}
			assert.Equal(t, append(capturedCherryPickArgv(commands.version), option, merge), cmd.GetCmd().Args)
			capturedCherryPickRequireNoError(t, cmd.Run())
			assert.FileExists(t, filepath.Join(root, picked))
			assert.NoFileExists(t, filepath.Join(root, absent))
			assert.Equal(t, base, gitTest(t, root, "rev-parse", "HEAD^"))
		})
	}
}

func TestCapturedCherryPickIgnoredRiskRequiresFreshApproval(t *testing.T) {
	root := initHistoricalRepo(t)
	base := capturedCherryPickCommit(t, root, ".gitignore", "ignored.txt\nunrelated.txt\nancestor.txt\n", "ignore local files", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "source")
	writeTestFile(t, filepath.Join(root, "ancestor.txt"), []byte("unselected ancestor\n"), 0o644)
	gitTest(t, root, "add", "-f", "--", "ancestor.txt")
	gitTest(t, root, "commit", "-qm", "unselected addition")
	writeTestFile(t, filepath.Join(root, "ignored.txt"), []byte("committed\n"), 0o644)
	gitTest(t, root, "add", "-f", "--", "ignored.txt")
	gitTest(t, root, "commit", "-qm", "add ignored path")
	oid := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-qb", "destination", base)
	writeTestFile(t, filepath.Join(root, "ignored.txt"), []byte("untracked ignored\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "unrelated.txt"), []byte("unrelated ignored\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "ancestor.txt"), []byte("local ignored ancestor\n"), 0o644)
	commands := capturedCherryPickCommands(t, root)
	plan, err := commands.PrepareCherryPickPlan([]string{oid})
	capturedCherryPickRequireNoError(t, err)
	assert.Equal(t, []string{"ignored.txt"}, plan.IgnoredFiles())
	files := plan.IgnoredFiles()
	files[0] = "changed"
	assert.Equal(t, []string{"ignored.txt"}, plan.IgnoredFiles())
	_, err = commands.CherryPickCommand(plan)
	assert.ErrorIs(t, err, ErrRebaseIgnoredApprovalRequired)
	approved := plan.WithIgnoredFilesApproved()
	_, err = commands.CherryPickCommand(approved)
	capturedCherryPickRequireNoError(t, err)
	writeTestFile(t, filepath.Join(root, "ignored.txt"), []byte("changed ignored\n"), 0o644)
	_, err = commands.CherryPickCommand(approved)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
}

func TestCapturedCherryPickCommonMainlineAndAllParentIgnoredRisks(t *testing.T) {
	root, base, first, second := capturedCherryPickReplayFixture(t)
	tree := gitTest(t, root, "rev-parse", second+"^{tree}")
	merge := gitTest(t, root, "commit-tree", tree, "-p", first, "-p", second, "-m", "two parents")
	octopus := gitTest(t, root, "commit-tree", tree, "-p", base, "-p", first, "-p", second, "-m", "three parents")
	commands := capturedCherryPickCommands(t, root)
	plan, err := commands.PrepareCherryPickPlan([]string{merge, octopus, first})
	capturedCherryPickRequireNoError(t, err)
	_, err = plan.WithMainline(3)
	assert.Error(t, err, "one common parent must exist in every selected merge")
	approved, err := plan.WithMainline(2)
	capturedCherryPickRequireNoError(t, err)
	cmd, err := commands.CherryPickCommand(approved)
	capturedCherryPickRequireNoError(t, err)
	assert.Equal(t, append(append(capturedCherryPickArgv(commands.version), "-m2"), capturedCherryPickOIDs(plan)...), cmd.GetCmd().Args)

	// The two-parent commit adds remaining.txt relative to first, but not
	// relative to second. Preflight must review that risk even for mainline 2.
	writeTestFile(t, filepath.Join(root, ".git", "info", "exclude"), []byte("remaining.txt\nunrelated.ignored\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "remaining.txt"), []byte("ignored local contents\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "unrelated.ignored"), []byte("unrelated\n"), 0o644)
	plan, err = commands.PrepareCherryPickPlan([]string{merge})
	capturedCherryPickRequireNoError(t, err)
	assert.Equal(t, []string{"remaining.txt"}, plan.IgnoredFiles())
	plan, err = plan.WithMainline(2)
	capturedCherryPickRequireNoError(t, err)
	_, err = commands.CherryPickCommand(plan)
	assert.ErrorIs(t, err, ErrRebaseIgnoredApprovalRequired)
}

func capturedCherryPickReplayFixture(t *testing.T) (root, base, first, second string) {
	t.Helper()
	root = initHistoricalRepo(t)
	capturedCherryPickCommit(t, root, "unrelated.txt", "unrelated\n", "unrelated base", capturedCherryPickDate)
	base = capturedCherryPickCommit(t, root, "collision.txt", "base\n", "base", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "source")
	first = capturedCherryPickCommit(t, root, "collision.txt", "source\n", "source change", capturedCherryPickDate)
	second = capturedCherryPickCommit(t, root, "remaining.txt", "remaining\n", "remaining change", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "destination", base)
	return root, base, first, second
}

func capturedCherryPickCommit(t *testing.T, root, path, contents, subject, date string) string {
	t.Helper()
	writeTestFile(t, filepath.Join(root, path), []byte(contents), 0o644)
	gitTest(t, root, "add", "--", path)
	cmd := exec.Command("git", "-C", root, "commit", "-qm", subject)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return gitTest(t, root, "rev-parse", "HEAD")
}

func capturedCherryPickCommands(t *testing.T, root string) *RebaseCommands {
	t.Helper()
	commands := selectedCommitRebaseCommands(t, root)
	version, err := ParseGitVersion(gitTest(t, root, "--version"))
	capturedCherryPickRequireNoError(t, err)
	commands.version = version
	return commands
}

func capturedCherryPickArgv(version *GitVersion) []string {
	option := "--keep-redundant-commits"
	if version.IsAtLeast(2, 45, 0) {
		option = "--empty=keep"
	}
	return []string{"git", "cherry-pick", "--allow-empty", option}
}

func capturedCherryPickOIDs(plan CherryPickPlan) []string {
	oids := make([]string, 0, len(plan.commits))
	for _, commit := range plan.Commits() {
		oids = append(oids, commit.OID)
	}
	return oids
}

func capturedCherryPickStdout(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	capturedCherryPickRequireNoError(t, err)
	return string(output)
}

func capturedCherryPickAssertNoStash(t *testing.T, root string) {
	t.Helper()
	output := capturedCherryPickStdout(t, root, "for-each-ref", "--format=%(refname)", "refs/stash")
	assert.Empty(t, output)
}

func capturedCherryPickRequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
