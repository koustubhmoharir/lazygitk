package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamedCheckoutSparseCheckoutPreservesConeAndDirtyCarry(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "included", "file"), []byte("included first\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "included", "carry"), []byte("carry base\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "outside", "file"), []byte("outside first\n"), 0o644)
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-qm", "first")
	first := gitTest(t, root, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(root, "included", "file"), []byte("included second\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "outside", "file"), []byte("outside second\n"), 0o644)
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-qm", "second")

	gitTest(t, root, "sparse-checkout", "init", "--cone")
	gitTest(t, root, "sparse-checkout", "set", "included")
	writeTestFile(t, filepath.Join(root, "included", "carry"), []byte("dirty carry\n"), 0o644)

	plan, err := transactionCommands(t, root).PrepareNamedCheckout("sparse-first", first)
	namedCheckoutMustNoError(t, err)
	namedCheckoutMustNoError(t, transactionCommands(t, root).ApplyNamedCheckout(plan))

	assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"))
	included, err := os.ReadFile(filepath.Join(root, "included", "file"))
	namedCheckoutMustNoError(t, err)
	assert.Equal(t, "included first\n", string(included))
	assert.NoFileExists(t, filepath.Join(root, "outside", "file"))
	carry, err := os.ReadFile(filepath.Join(root, "included", "carry"))
	namedCheckoutMustNoError(t, err)
	assert.Equal(t, "dirty carry\n", string(carry))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", "core.sparseCheckout"))
	assert.Equal(t, "included", gitTest(t, root, "sparse-checkout", "list"))
}

type namedCheckoutSubmoduleFixture struct {
	root        string
	firstParent string
	firstChild  string
	secondChild string
	childPath   string
}

func newNamedCheckoutSubmoduleFixture(t *testing.T) namedCheckoutSubmoduleFixture {
	t.Helper()
	child := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(child, "tracked"), []byte("child first\n"), 0o644)
	gitTest(t, child, "add", "tracked")
	gitTest(t, child, "commit", "-qm", "child first")
	firstChild := gitTest(t, child, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(child, "tracked"), []byte("child second\n"), 0o644)
	gitTest(t, child, "commit", "-qam", "child second")
	secondChild := gitTest(t, child, "rev-parse", "HEAD")

	root := initHistoricalRepo(t)
	gitTest(t, root, "-c", "protocol.file.allow=always", "submodule", "add", child, "child")
	gitTest(t, root, "update-index", "--cacheinfo", "160000,"+firstChild+",child")
	gitTest(t, root, "commit", "-qm", "parent first")
	firstParent := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "update-index", "--cacheinfo", "160000,"+secondChild+",child")
	gitTest(t, root, "commit", "-qm", "parent second")
	return namedCheckoutSubmoduleFixture{
		root: root, firstParent: firstParent,
		firstChild: firstChild, secondChild: secondChild, childPath: filepath.Join(root, "child"),
	}
}

func TestNamedCheckoutSubmoduleRespectsRecurseConfiguration(t *testing.T) {
	t.Run("disabled leaves submodule worktree at its current commit", func(t *testing.T) {
		fixture := newNamedCheckoutSubmoduleFixture(t)
		gitTest(t, fixture.root, "config", "submodule.recurse", "false")
		plan, err := transactionCommands(t, fixture.root).PrepareNamedCheckout("parent-first", fixture.firstParent)
		namedCheckoutMustNoError(t, err)
		namedCheckoutMustNoError(t, transactionCommands(t, fixture.root).ApplyNamedCheckout(plan))

		assert.Equal(t, fixture.firstParent, gitTest(t, fixture.root, "rev-parse", "HEAD"))
		assert.Equal(t, fixture.firstChild, gitTest(t, fixture.root, "rev-parse", ":child"))
		assert.Equal(t, fixture.secondChild, gitTest(t, fixture.childPath, "rev-parse", "HEAD"))
		contents, readErr := os.ReadFile(filepath.Join(fixture.childPath, "tracked"))
		namedCheckoutMustNoError(t, readErr)
		assert.Equal(t, "child second\n", string(contents))
	})

	t.Run("enabled updates a clean submodule worktree", func(t *testing.T) {
		fixture := newNamedCheckoutSubmoduleFixture(t)
		gitTest(t, fixture.root, "config", "submodule.recurse", "true")
		plan, err := transactionCommands(t, fixture.root).PrepareNamedCheckout("parent-first", fixture.firstParent)
		namedCheckoutMustNoError(t, err)
		namedCheckoutMustNoError(t, transactionCommands(t, fixture.root).ApplyNamedCheckout(plan))

		assert.Equal(t, fixture.firstParent, gitTest(t, fixture.root, "rev-parse", "HEAD"))
		assert.Equal(t, fixture.firstChild, gitTest(t, fixture.root, "rev-parse", ":child"))
		assert.Equal(t, fixture.firstChild, gitTest(t, fixture.childPath, "rev-parse", "HEAD"))
		contents, readErr := os.ReadFile(filepath.Join(fixture.childPath, "tracked"))
		namedCheckoutMustNoError(t, readErr)
		assert.Equal(t, "child first\n", string(contents))
	})

	t.Run("enabled protects dirty tracked child bytes", func(t *testing.T) {
		fixture := newNamedCheckoutSubmoduleFixture(t)
		gitTest(t, fixture.root, "config", "submodule.recurse", "true")
		writeTestFile(t, filepath.Join(fixture.childPath, "tracked"), []byte("dirty child bytes\n"), 0o644)
		beforeHead := gitTest(t, fixture.root, "rev-parse", "HEAD")
		beforeIndex := gitTest(t, fixture.root, "write-tree")
		beforeChildHead := gitTest(t, fixture.childPath, "rev-parse", "HEAD")

		plan, err := transactionCommands(t, fixture.root).PrepareNamedCheckout("parent-first", fixture.firstParent)
		namedCheckoutMustNoError(t, err)
		applyErr := transactionCommands(t, fixture.root).ApplyNamedCheckout(plan)
		assert.Error(t, applyErr, "Git must reject overwriting dirty tracked submodule content")
		assert.Equal(t, beforeHead, gitTest(t, fixture.root, "rev-parse", "HEAD"))
		assert.Equal(t, beforeChildHead, gitTest(t, fixture.childPath, "rev-parse", "HEAD"))
		contents, readErr := os.ReadFile(filepath.Join(fixture.childPath, "tracked"))
		namedCheckoutMustNoError(t, readErr)
		assert.Equal(t, "dirty child bytes\n", string(contents))
		branchErr := execGitTest(t, fixture.root, "show-ref", "--verify", "refs/heads/parent-first")
		assert.Error(t, branchErr, "failed checkout must not create the destination branch")

		baseline := newNamedCheckoutSubmoduleFixture(t)
		gitTest(t, baseline.root, "config", "submodule.recurse", "true")
		writeTestFile(t, filepath.Join(baseline.childPath, "tracked"), []byte("dirty child bytes\n"), 0o644)
		baselineHead := gitTest(t, baseline.root, "rev-parse", "HEAD")
		baselineIndex := gitTest(t, baseline.root, "write-tree")
		baselineParentFile := mustReadNamedCheckoutFile(t, filepath.Join(baseline.root, "child", "tracked"))
		cliErr := execGitTest(t, baseline.root, "checkout", "--no-overwrite-ignore", "--no-guess", "-B", "parent-first", baseline.firstParent+"^{commit}")
		assert.Error(t, cliErr, "direct Git checkout should reject the same dirty submodule transition")
		if gitTest(t, baseline.root, "rev-parse", "HEAD") == baselineHead {
			assert.Equal(t, beforeHead, gitTest(t, fixture.root, "rev-parse", "HEAD"))
		}
		if gitTest(t, baseline.root, "write-tree") == baselineIndex {
			assert.Equal(t, beforeIndex, gitTest(t, fixture.root, "write-tree"), "parent index preservation is only asserted when native Git preserves it")
		}
		if got := mustReadNamedCheckoutFile(t, filepath.Join(baseline.root, "child", "tracked")); string(got) == string(baselineParentFile) {
			assert.Equal(t, baselineParentFile, mustReadNamedCheckoutFile(t, filepath.Join(fixture.root, "child", "tracked")))
		}
	})
}

func execGitTest(t *testing.T, root string, args ...string) error {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	return cmd.Run()
}

func mustReadNamedCheckoutFile(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	namedCheckoutMustNoError(t, err)
	return contents
}
