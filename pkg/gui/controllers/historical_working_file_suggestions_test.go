package controllers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func requireSuggestionSetup(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestHistoricalWorkingFileSuggestionsExcludeActualGitDirectories(t *testing.T) {
	root := t.TempDir()
	metadata := filepath.Join(root, "repository-storage")
	requireSuggestionSetup(t, os.Mkdir(metadata, 0o700))
	requireSuggestionSetup(t, os.WriteFile(filepath.Join(metadata, "config"), []byte("private"), 0o600))
	requireSuggestionSetup(t, os.WriteFile(filepath.Join(root, "visible"), []byte("worktree"), 0o600))
	find := historicalWorkingFileSuggestions(root, metadata)
	for _, suggestion := range find("") {
		assert.NotContains(t, suggestion.Value, "repository-storage")
	}
	assert.Nil(t, find("repository-storage/"))
	assert.Nil(t, find(metadata+string(filepath.Separator)))
}

func TestHistoricalWorkingFileSuggestions(t *testing.T) {
	root := t.TempDir()
	writeSuggestionFixture := func(name, content string) {
		path := filepath.Join(root, name)
		requireSuggestionSetup(t, os.MkdirAll(filepath.Dir(path), 0o700))
		requireSuggestionSetup(t, os.WriteFile(path, []byte(content), 0o600))
	}
	writeSuggestionFixture("ordinary.txt", "ordinary")
	writeSuggestionFixture("tracked.txt", "tracked")
	writeSuggestionFixture("untracked.txt", "untracked")
	writeSuggestionFixture("ignored.txt", "ignored")
	writeSuggestionFixture(".gitignore", "ignored.txt\n")
	writeSuggestionFixture(filepath.Join("nested", "inside.txt"), "inside")
	writeSuggestionFixture(filepath.Join("space name", "file name.txt"), "space")
	writeSuggestionFixture("line\nbreak.txt", "newline")
	if runtime.GOOS != "windows" {
		writeSuggestionFixture(`back\slash.txt`, "backslash")
	}
	requireSuggestionSetup(t, os.Mkdir(filepath.Join(root, "browse"), 0o700))
	requireSuggestionSetup(t, os.Mkdir(filepath.Join(root, ".git"), 0o700))
	requireSuggestionSetup(t, os.WriteFile(filepath.Join(root, ".git", "metadata"), []byte("metadata"), 0o600))

	find := historicalWorkingFileSuggestions(root)
	values := func(input string) []string {
		suggestions := find(input)
		result := make([]string, 0, len(suggestions))
		for _, suggestion := range suggestions {
			assert.Equal(t, suggestion.Value, suggestion.Label)
			result = append(result, suggestion.Value)
		}
		return result
	}

	t.Run("empty input lists root entries in stable sorted order", func(t *testing.T) {
		got := values("")
		expected := []string{".gitignore", "browse/", "ignored.txt", "line\nbreak.txt", "nested/", "ordinary.txt", "space name/", "tracked.txt", "untracked.txt"}
		if runtime.GOOS != "windows" {
			expected = append(expected[:1], append([]string{`back\slash.txt`}, expected[1:]...)...)
		}
		if runtime.GOOS == "windows" {
			for i := range expected {
				expected[i] = filepath.FromSlash(expected[i])
			}
		}
		assert.Equal(t, expected, got)
		assert.Equal(t, got, values(""))
		assert.Equal(t, got, values(""))
	})

	t.Run("prefix and nested directory browsing", func(t *testing.T) {
		assert.Equal(t, []string{"ordinary.txt"}, values("ord"))
		assert.Empty(t, values("missing"))
		assert.Equal(t, []string{"nested/"}, values("nest"))
		assert.Equal(t, []string{"nested/inside.txt"}, values("nested/in"))
		assert.Equal(t, []string{"space name/file name.txt"}, values("space name/file"))
	})

	t.Run("absolute paths inside worktree retain absolute form", func(t *testing.T) {
		got := values(filepath.Join(root, "nested", "in"))
		assert.Equal(t, []string{filepath.Join(root, "nested", "inside.txt")}, got)
	})

	t.Run("outside paths and git metadata are rejected", func(t *testing.T) {
		outside := t.TempDir()
		requireSuggestionSetup(t, os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0o600))
		relativeOutside, err := filepath.Rel(root, outside)
		requireSuggestionSetup(t, err)
		assert.Nil(t, find(relativeOutside+string(filepath.Separator)))
		assert.Nil(t, find(outside+string(filepath.Separator)))
		assert.Nil(t, find(".git/"))
		assert.Nil(t, find(root+string(filepath.Separator)+".git"+string(filepath.Separator)))
	})

	t.Run("invalid roots and inputs return nil", func(t *testing.T) {
		assert.Nil(t, historicalWorkingFileSuggestions(filepath.Join(root, "missing"))(""))
		assert.Nil(t, find("nested/missing/"))
		assert.Nil(t, find("bad\x00name"))
	})

	t.Run("symlinked entries and traversal are excluded", func(t *testing.T) {
		outside := t.TempDir()
		requireSuggestionSetup(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))
		links := []struct {
			name, target string
			dir          bool
		}{
			{"linked-file", filepath.Join(root, "ordinary.txt"), false},
			{"linked-dir", filepath.Join(root, "nested"), true},
			{"outside-link", outside, true},
		}
		for _, link := range links {
			err := os.Symlink(link.target, filepath.Join(root, link.name))
			if err != nil {
				t.Skipf("OS does not permit symlink creation: %v", err)
			}
		}
		assert.NotContains(t, values(""), "linked-file")
		assert.NotContains(t, values(""), "linked-dir/")
		assert.NotContains(t, values(""), "outside-link/")
		assert.Nil(t, find("linked-dir/"))
		assert.Nil(t, find("outside-link/"))
	})

	t.Run("symlinked root is canonicalized and linked worktree git file excluded", func(t *testing.T) {
		alias := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(root, alias); err != nil {
			t.Skipf("OS does not permit symlink creation: %v", err)
		}
		worktree := t.TempDir()
		requireSuggestionSetup(t, os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: elsewhere\n"), 0o600))
		requireSuggestionSetup(t, os.WriteFile(filepath.Join(worktree, "visible"), []byte("visible"), 0o600))
		got := historicalWorkingFileSuggestions(alias)("")
		gotValues := make([]string, 0, len(got))
		for _, suggestion := range got {
			gotValues = append(gotValues, suggestion.Value)
		}
		assert.Contains(t, gotValues, "ordinary.txt")
		worktreeValues := historicalWorkingFileSuggestions(worktree)("")
		visible := false
		for _, suggestion := range worktreeValues {
			assert.NotEqual(t, ".git", suggestion.Value)
			visible = visible || suggestion.Value == "visible"
		}
		assert.True(t, visible)
	})

	t.Run("suggestions are data and do not mutate files", func(t *testing.T) {
		before, err := os.ReadFile(filepath.Join(root, "ordinary.txt"))
		requireSuggestionSetup(t, err)
		_ = values("line\n")
		_ = values(`back\`)
		after, err := os.ReadFile(filepath.Join(root, "ordinary.txt"))
		requireSuggestionSetup(t, err)
		assert.Equal(t, before, after)
	})
}
