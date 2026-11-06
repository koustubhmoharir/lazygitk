package git_commands

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// checkoutByNameRepo is a repository with two commits on master: first is the
// older one and second is HEAD.
func checkoutByNameRepo(t *testing.T) (root, first, second string) {
	t.Helper()
	root, first, second = transactionHistory(t)
	writeTestFile(t, filepath.Join(root, "README"), []byte("readme\n"), 0o644)
	gitTest(t, root, "add", "README")
	gitTest(t, root, "commit", "-qm", "readme")
	second = gitTest(t, root, "rev-parse", "HEAD")
	return root, first, second
}

func classifyCheckoutByName(t *testing.T, root, input string) CheckoutByNameTarget {
	t.Helper()
	target, err := transactionCommands(t, root).ClassifyCheckoutByName(input)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func TestClassifyCheckoutByNameLocalBranch(t *testing.T) {
	root, first, _ := checkoutByNameRepo(t)
	gitTest(t, root, "branch", "topic", first)

	target := classifyCheckoutByName(t, root, "topic")
	assert.Equal(t, CheckoutByNameLocalBranch, target.Kind)
	assert.Equal(t, "refs/heads/topic", target.FullRef)
}

func TestClassifyCheckoutByNameLocalBranchWinsOverTag(t *testing.T) {
	root, first, _ := checkoutByNameRepo(t)
	gitTest(t, root, "branch", "v1", first)
	gitTest(t, root, "tag", "v1", first)

	target := classifyCheckoutByName(t, root, "v1")
	assert.Equal(t, CheckoutByNameLocalBranch, target.Kind)
	assert.Equal(t, "refs/heads/v1", target.FullRef)
}

func TestClassifyCheckoutByNameRemoteBranch(t *testing.T) {
	root, first, _ := checkoutByNameRepo(t)
	gitTest(t, root, "update-ref", "refs/remotes/origin/topic", first)

	target := classifyCheckoutByName(t, root, "origin/topic")
	assert.Equal(t, CheckoutByNameRemoteBranch, target.Kind)
	assert.Equal(t, "refs/remotes/origin/topic", target.FullRef)
}

func TestClassifyCheckoutByNameTag(t *testing.T) {
	root, first, _ := checkoutByNameRepo(t)
	gitTest(t, root, "tag", "v2", first)

	target := classifyCheckoutByName(t, root, "v2")
	assert.Equal(t, CheckoutByNameTag, target.Kind)
	assert.Equal(t, "refs/tags/v2", target.FullRef)
}

func TestClassifyCheckoutByNameAbbreviatedHash(t *testing.T) {
	root, first, _ := checkoutByNameRepo(t)
	short := gitTest(t, root, "rev-parse", "--short=10", first)

	target := classifyCheckoutByName(t, root, short)
	assert.Equal(t, CheckoutByNameCommit, target.Kind)
	assert.Equal(t, first, target.OID)
}

func TestClassifyCheckoutByNamePrevious(t *testing.T) {
	root, _, _ := checkoutByNameRepo(t)

	target := classifyCheckoutByName(t, root, " - ")
	assert.Equal(t, CheckoutByNamePrevious, target.Kind)
}

func TestClassifyCheckoutByNameFullRefs(t *testing.T) {
	root, first, _ := checkoutByNameRepo(t)
	gitTest(t, root, "branch", "topic", first)
	gitTest(t, root, "tag", "v2", first)
	gitTest(t, root, "update-ref", "refs/remotes/origin/topic", first)

	for input, want := range map[string]struct {
		kind CheckoutByNameKind
		ref  string
	}{
		"refs/heads/topic":          {CheckoutByNameLocalBranch, "refs/heads/topic"},
		"refs/remotes/origin/topic": {CheckoutByNameRemoteBranch, "refs/remotes/origin/topic"},
		"refs/tags/v2":              {CheckoutByNameTag, "refs/tags/v2"},
		"refs/heads/missing":        {CheckoutByNameNotFound, "refs/heads/missing"},
		"refs/stash":                {CheckoutByNameNotFound, "refs/stash"},
	} {
		target := classifyCheckoutByName(t, root, input)
		assert.Equal(t, want.kind, target.Kind, input)
		assert.Equal(t, want.ref, target.FullRef, input)
	}
}

func TestClassifyCheckoutByNameAmbiguousRemoteAndTag(t *testing.T) {
	root, first, _ := checkoutByNameRepo(t)
	gitTest(t, root, "update-ref", "refs/remotes/origin/topic", first)
	gitTest(t, root, "tag", "origin/topic", first)

	target := classifyCheckoutByName(t, root, "origin/topic")
	assert.Equal(t, CheckoutByNameAmbiguous, target.Kind)
	assert.Equal(t, []string{"refs/remotes/origin/topic", "refs/tags/origin/topic"}, target.Refs)

	// The full ref names are literal, so they are not ambiguous.
	target = classifyCheckoutByName(t, root, "refs/tags/origin/topic")
	assert.Equal(t, CheckoutByNameTag, target.Kind)
}

func TestClassifyCheckoutByNameAmbiguousWithGitDWIMRef(t *testing.T) {
	root, _, _ := checkoutByNameRepo(t)
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/repo.git")
	head := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "update-ref", "refs/remotes/origin/main", head)
	gitTest(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	target := classifyCheckoutByName(t, root, "origin")
	assert.Equal(t, CheckoutByNameAmbiguous, target.Kind)
	assert.Equal(t, []string{"refs/remotes/origin/HEAD"}, target.Refs)
}

func TestClassifyCheckoutByNameRefusesSymbolicRemote(t *testing.T) {
	root, _, _ := checkoutByNameRepo(t)
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/repo.git")
	gitTest(t, root, "update-ref", "refs/remotes/origin/main", "HEAD")
	gitTest(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	_, err := transactionCommands(t, root).ClassifyCheckoutByName("refs/remotes/origin/HEAD")
	assert.ErrorContains(t, err, "symbolic")
}

func TestClassifyCheckoutByNameTrackedPathIsNotARef(t *testing.T) {
	root, _, _ := checkoutByNameRepo(t)

	target := classifyCheckoutByName(t, root, "README")
	assert.Equal(t, CheckoutByNameNotFound, target.Kind)
	assert.Equal(t, "", target.FullRef)
	assert.Equal(t, "", target.OID)
}

func TestClassifyCheckoutByNameUnknownName(t *testing.T) {
	root, _, _ := checkoutByNameRepo(t)

	for _, input := range []string{"new-branch", "no such thing", "a..b"} {
		target := classifyCheckoutByName(t, root, input)
		assert.Equal(t, CheckoutByNameNotFound, target.Kind, input)
	}
}
