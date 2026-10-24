package git_commands

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestResolveHistoricalComparisonCommit(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, root+"/tracked", []byte("one"), 0o644)
	gitTest(t, root, "add", "tracked")
	gitTest(t, root, "commit", "-qm", "one")
	one := gitTest(t, root, "rev-parse", "HEAD")
	writeTestFile(t, root+"/tracked", []byte("two"), 0o644)
	gitTest(t, root, "commit", "-qam", "two")
	two := gitTest(t, root, "rev-parse", "HEAD")
	commands := realHistoricalDiffCommands(t, root)

	for _, input := range []string{"", "abc", "HEAD", "HEAD~", "--help", "zzzz", strings.Repeat("a", 65)} {
		_, err := commands.ResolveHistoricalComparisonCommit(input)
		assert.Error(t, err, input)
	}
	resolved, err := commands.ResolveHistoricalComparisonCommit(" \t" + strings.ToUpper(one[:8]) + "\n")
	assert.NoError(t, err)
	assert.Equal(t, one, resolved.OID)
	resolved, err = commands.ResolveHistoricalComparisonCommit(two)
	assert.NoError(t, err)
	assert.Equal(t, two, resolved.OID)

	blob := gitTest(t, root, "rev-parse", one+":tracked")
	_, err = commands.ResolveHistoricalComparisonCommit(blob)
	assert.Error(t, err, "blob must not be accepted as a commit")
	tree := gitTest(t, root, "rev-parse", one+"^{tree}")
	_, err = commands.ResolveHistoricalComparisonCommit(tree)
	assert.Error(t, err, "tree must not be accepted as a commit")
	gitTest(t, root, "tag", "-a", "annotated", one, "-m", "tag")
	tagObject := gitTest(t, root, "rev-parse", "refs/tags/annotated")
	_, err = commands.ResolveHistoricalComparisonCommit(tagObject)
	assert.Error(t, err, "annotated tag object must not be peeled and accepted")

	// A hexadecimal ref that shadows the short spelling must not affect object lookup.
	gitTest(t, root, "update-ref", "refs/heads/"+one[:8], two)
	resolved, err = commands.ResolveHistoricalComparisonCommit(one[:8])
	assert.NoError(t, err)
	assert.Equal(t, one, resolved.OID)
	gitTest(t, root, "update-ref", "refs/heads/"+one, two)
	resolved, err = commands.ResolveHistoricalComparisonCommit(one)
	assert.NoError(t, err)
	assert.Equal(t, one, resolved.OID)
}

func TestResolveHistoricalComparisonCommitAmbiguousPrefix(t *testing.T) {
	root := t.TempDir()
	runner := oscommands.NewFakeRunner(t).ExpectGitArgs(
		[]string{"rev-parse", "--disambiguate=abcd"},
		"abcd000000000000000000000000000000000000\nabcd111111111111111111111111111111111111\n",
		nil,
	)
	inner := oscommands.NewDummyCmdObjBuilder(runner)
	common := NewGitCommon(nil, nil, rootedTestBuilder{inner: inner, root: root}, nil, MockRepoPaths(root), nil, nil)
	_, err := NewDiffCommands(common).ResolveHistoricalComparisonCommit("abcd")
	assert.ErrorContains(t, err, "ambiguous")
	runner.CheckForMissingCalls()
}

func TestResolveHistoricalComparisonCommitSHA256(t *testing.T) {
	root := t.TempDir()
	init := exec.Command("git", "-C", root, "init", "-q", "--object-format=sha256")
	if output, err := init.CombinedOutput(); err != nil {
		t.Skipf("Git does not support SHA-256 repositories: %v: %s", err, output)
	}
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("sha256"), 0o644)
	gitTest(t, root, "add", "tracked")
	gitTest(t, root, "commit", "-qm", "sha256")
	oid := gitTest(t, root, "rev-parse", "HEAD")
	resolved, err := realHistoricalDiffCommands(t, root).ResolveHistoricalComparisonCommit(oid)
	assert.NoError(t, err)
	assert.Equal(t, oid, resolved.OID)
}

func TestHistoricalCommitComparisonCmdObjArguments(t *testing.T) {
	commands := testHistoricalDiffCommands(t)
	left := HistoricalDiffCommit{OID: "0123456789abcdef0123456789abcdef01234567"}
	right := HistoricalDiffCommit{OID: "abcdef0123456789abcdef0123456789abcdef01"}
	paths := []string{"-leading", "odd\tname", "line\nbreak", ":(glob)name"}
	cmd, err := commands.HistoricalCommitComparisonCmdObj(left, right, paths, true)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, append([]string{"git", "--literal-pathspecs", "difftool", "--no-prompt", "--dir-diff", left.OID, right.OID, "--"}, paths...), cmd.Args())
	assert.Equal(t, commands.repoPaths.WorktreePath(), cmd.GetCmd().Dir)
	fileCmd, err := commands.HistoricalCommitComparisonCmdObj(left, right, []string{"file.txt"}, false)
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", left.OID, right.OID, "--", "file.txt"}, fileCmd.Args())
	}
	for _, path := range []string{"/absolute", "../escape", "nul\x00path"} {
		_, err := commands.HistoricalCommitComparisonCmdObj(left, right, []string{path}, false)
		assert.Error(t, err, path)
	}
	_, err = commands.HistoricalCommitComparisonCmdObj(left, HistoricalDiffCommit{OID: "HEAD"}, nil, false)
	assert.Error(t, err)
}
