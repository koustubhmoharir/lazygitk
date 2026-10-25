package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func makeRepoPathTree(t *testing.T, dirs ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func suggestionLabels(suggestions []*types.Suggestion) []string {
	return lo.Map(suggestions, func(s *types.Suggestion, _ int) string { return s.Label })
}

func TestSplitRepoPathInput(t *testing.T) {
	dir, prefix := splitRepoPathInput("/src/lazy")
	assert.Equal(t, "/src/", dir)
	assert.Equal(t, "lazy", prefix)

	dir, prefix = splitRepoPathInput("lazy")
	assert.Equal(t, "", dir)
	assert.Equal(t, "lazy", prefix)

	dir, prefix = splitRepoPathInput("/src/")
	assert.Equal(t, "/src/", dir)
	assert.Equal(t, "", prefix)
}

func TestResolveRepoPath(t *testing.T) {
	sep := string(filepath.Separator)
	root := filepath.Join(sep, "work", "lazygitk")
	home := filepath.Join(sep, "home", "me")

	assert.Equal(t, filepath.Join(sep, "abs", "dir"), resolveRepoPath("/abs/dir", root, home))
	assert.Equal(t, filepath.Join(home, "src"), resolveRepoPath("~/src", root, home))
	assert.Equal(t, home, resolveRepoPath("~", root, home))
	// A ~ that is not a whole leading segment is an ordinary name.
	assert.Equal(t, filepath.Join(root, "~src"), resolveRepoPath("~src", root, home))
	assert.Equal(t, filepath.Join(filepath.Dir(root), "other"), resolveRepoPath("../other", root, home))
	assert.Equal(t, filepath.Join(root, "sub"), resolveRepoPath("sub", root, home))
}

func TestRepoPathDefaultInputAndNormalize(t *testing.T) {
	sep := string(filepath.Separator)
	assert.Equal(t, filepath.Join(sep, "work")+sep, repoPathDefaultInput(filepath.Join(sep, "work", "lazygitk")))
	assert.Equal(t, sep, repoPathDefaultInput(sep))
	assert.Equal(t, filepath.Join(sep, "work")+sep, normalizeRepoPathInput("", filepath.Join(sep, "work", "lazygitk")))
	assert.Equal(t, "x", normalizeRepoPathInput("x", "/work/lazygitk"))
}

func TestMatchRepoPathDirsPrefixHiddenAndSorting(t *testing.T) {
	root := makeRepoPathTree(t, "beta", "alpha", "alphabet", "gamma", ".hidden", ".alpha-dot")
	if err := os.WriteFile(filepath.Join(root, "alpha-file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"alpha", "alphabet"}, matchRepoPathDirs(root, "alp"))
	// Files never match.
	assert.Equal(t, []string{"alpha", "alphabet"}, matchRepoPathDirs(root, "alpha"))
	// Hidden directories are hidden unless the prefix starts with a dot.
	assert.Equal(t, []string{"alpha", "alphabet", "beta", "gamma"}, matchRepoPathDirs(root, ""))
	assert.Equal(t, []string{".alpha-dot", ".hidden"}, matchRepoPathDirs(root, "."))
	assert.Equal(t, []string{".alpha-dot"}, matchRepoPathDirs(root, ".a"))
	assert.Empty(t, matchRepoPathDirs(filepath.Join(root, "missing"), ""))
}

func TestMatchRepoPathDirsFollowsSymlinkToDirectory(t *testing.T) {
	root := makeRepoPathTree(t, "real")
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"link", "real"}, matchRepoPathDirs(root, ""))
}

func TestRepoPathSuggestionsValuesLabelsAndCap(t *testing.T) {
	dirs := make([]string, 0, 23)
	dirs = append(dirs, "a", "b", "c")
	for i := range 20 {
		dirs = append(dirs, "proj"+string(rune('a'+i)))
	}
	root := makeRepoPathTree(t, dirs...)
	sep := string(filepath.Separator)
	repoRoot := filepath.Join(root, "current")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	// Typed absolute path with a prefix: values carry the full input and a separator.
	suggestions := repoPathSuggestions(root+sep+"pro", repoRoot, "")
	assert.Len(t, suggestions, repoPathSuggestionLimit)
	assert.Equal(t, root+sep+"proja"+sep, suggestions[0].Value)
	assert.Equal(t, "proja/", suggestions[0].Label)

	// Names are sorted and the list is capped.
	all := repoPathSuggestions(root+sep, repoRoot, "")
	assert.Len(t, all, repoPathSuggestionLimit)
	assert.Equal(t, "a/", all[0].Label)
	assert.Equal(t, "b/", all[1].Label)
	assert.Equal(t, "c/", all[2].Label)
}

func TestRepoPathSuggestionsEmptyInputMatchesPrefilledParent(t *testing.T) {
	parent := makeRepoPathTree(t, "repo", "other")
	repoRoot := filepath.Join(parent, "repo")
	sep := string(filepath.Separator)

	fromEmpty := repoPathSuggestions("", repoRoot, "")
	fromPrefill := repoPathSuggestions(repoPathDefaultInput(repoRoot), repoRoot, "")
	assert.Equal(t, fromPrefill, fromEmpty)
	assert.Equal(t, []string{"other/", "repo/"}, suggestionLabels(fromEmpty))
	assert.Equal(t, parent+sep+"other"+sep, fromEmpty[0].Value)
}

func TestRepoPathSuggestionsRelativeAndTilde(t *testing.T) {
	parent := makeRepoPathTree(t, "repo", "sibling", "home/projects")
	repoRoot := filepath.Join(parent, "repo")
	home := filepath.Join(parent, "home")
	sep := string(filepath.Separator)

	relative := repoPathSuggestions("../sib", repoRoot, home)
	assert.Len(t, relative, 1)
	assert.Equal(t, "../sibling"+sep, relative[0].Value)

	tilde := repoPathSuggestions("~"+sep, repoRoot, home)
	assert.Equal(t, []string{"projects/"}, suggestionLabels(tilde))
	assert.Equal(t, "~"+sep+"projects"+sep, tilde[0].Value)
}

func TestRepoPathSuggestionsNonexistentDirectoryGivesNone(t *testing.T) {
	repoRoot := makeRepoPathTree(t, "repo")
	assert.Empty(t, repoPathSuggestions(filepath.Join(repoRoot, "nope")+string(filepath.Separator), repoRoot, ""))
}

func TestCompleteRepoPathMatches(t *testing.T) {
	sep := string(filepath.Separator)

	// Longest common prefix of several matches, no separator.
	completed, ok := completeRepoPathMatches("/src/", "pro", []string{"project-a", "project-b"})
	assert.True(t, ok)
	assert.Equal(t, "/src/project-", completed)

	// Exactly one match: the name plus a separator.
	completed, ok = completeRepoPathMatches("/src/", "sib", []string{"sibling"})
	assert.True(t, ok)
	assert.Equal(t, "/src/sibling"+sep, completed)

	// A fully typed unique name still gains the separator.
	completed, ok = completeRepoPathMatches("/src/", "sibling", []string{"sibling"})
	assert.True(t, ok)
	assert.Equal(t, "/src/sibling"+sep, completed)

	// Several matches with nothing more in common: no progress, Tab moves on.
	completed, ok = completeRepoPathMatches("/src/", "pro", []string{"project-a", "proxy"})
	assert.False(t, ok)
	assert.Empty(t, completed)

	// A single match that is already complete still gains a separator.
	_, ok = completeRepoPathMatches("/src/", "pro", []string{"pro"})
	assert.True(t, ok)

	// No matches: no progress.
	_, ok = completeRepoPathMatches("/src/", "zzz", nil)
	assert.False(t, ok)
}

func TestCompleteRepoPathMatchesKeepsRunesWhole(t *testing.T) {
	// "é" and "è" share their first UTF-8 byte; the completion must not cut it.
	completed, ok := completeRepoPathMatches("", "", []string{"école", "èté"})
	assert.False(t, ok)
	assert.Empty(t, completed)

	completed, ok = completeRepoPathMatches("", "", []string{"éa", "éb"})
	assert.True(t, ok)
	assert.Equal(t, "é", completed)
}
