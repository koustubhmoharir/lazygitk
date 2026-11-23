package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{312, "312 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1100, "1.1 KB"},
		{4300, "4.2 KB"},
		{1024 * 1023, "1023.0 KB"},
		{1024*1024 - 30, "1.0 MB"}, // 1023.97 KB would read as 1024.0 KB
		{1024 * 1024, "1.0 MB"},
		{1024*1024*1023 + 1024*1024/2, "1023.5 MB"},
		{1024 * 1024 * 1024 * 5, "5.0 GB"},
		{1024 * 1024 * 1024 * 1024 * 3, "3.0 TB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatFileSize(tt.size), "size %d", tt.size)
	}
}

func TestFolderListingRows(t *testing.T) {
	entries := []git_commands.TreeEntry{
		{Path: "src/util.go", Type: "blob", Object: "u", Size: 1100},
		{Path: "src/new.go", Type: "blob", Object: "n", Size: 312},
		{Path: "src/main.go", Type: "blob", Object: "m", Size: 4300},
		{Path: "src/gui", Type: "tree", Object: "g", Size: -1},
		{Path: "src/config", Type: "tree", Object: "c", Size: -1},
		{Path: "src/foo", Type: "commit", Object: "f", Size: -1},
	}
	changed := []*models.CommitFile{
		{Path: "src/main.go", ChangeStatus: "M"},
		{Path: "src/new.go", ChangeStatus: "A"},
		{Path: "src/old.go", ChangeStatus: "D"},
		{Path: "src/foo", ChangeStatus: "M", Submodule: true},
	}
	fromEntries := []git_commands.TreeEntry{
		{Path: "src/old.go", Type: "blob", Object: "o", Size: 2048},
	}

	rows := folderListingRows("src/", entries, changed, fromEntries, "(submodule)")

	assert.Equal(t, []folderRow{
		{name: "config/"},
		{name: "gui/"},
		{name: "foo", status: "M", size: "(submodule)"},
		{name: "main.go", status: "M", size: "4.2 KB"},
		{name: "new.go", status: "A", size: "312 B"},
		{name: "old.go", status: "D", size: "2.0 KB"},
		{name: "util.go", size: "1.1 KB"},
	}, rows)
}

func TestFolderListingRowsDeletedFolderAndRoot(t *testing.T) {
	// A folder the commit deleted is absent from the commit's tree, and is listed as
	// deleted once; a file at the root is listed as is.
	entries := []git_commands.TreeEntry{
		{Path: "readme.md", Type: "blob", Object: "r", Size: 10},
	}
	changed := []*models.CommitFile{
		{Path: "gone/a.go", ChangeStatus: "D"},
		{Path: "gone/b/c.go", ChangeStatus: "D"},
		{Path: "readme.md"},
	}

	rows := folderListingRows("", entries, changed, nil, "(submodule)")

	assert.Equal(t, []folderRow{
		{name: "gone/", status: "D"},
		{name: "readme.md", size: "10 B"},
	}, rows)
}

func TestFolderListingRowsDeletedFileWithoutSize(t *testing.T) {
	// Without the parent's entries, a deleted file has no size.
	changed := []*models.CommitFile{{Path: "src/old.go", ChangeStatus: "D"}}
	rows := folderListingRows("src/", nil, changed, nil, "(submodule)")
	assert.Equal(t, []folderRow{{name: "old.go", status: "D"}}, rows)
}

func TestFormatFolderRows(t *testing.T) {
	rows := []folderRow{
		{name: "config/"},
		{name: "gui/"},
		{name: "main.go", status: "M", size: "4.2 KB"},
		{name: "util.go", size: "1.1 KB"},
		{name: "new.go", status: "A", size: "312 B"},
	}

	assert.Equal(t, ""+
		"  config/\n"+
		"  gui/\n"+
		"M main.go  4.2 KB\n"+
		"  util.go  1.1 KB\n"+
		"A new.go    312 B", formatFolderRows(rows))
}

func TestFormatFolderRowsEmpty(t *testing.T) {
	assert.Equal(t, "", formatFolderRows(nil))
}

func TestFolderEntriesText(t *testing.T) {
	assert.Equal(t, "(1 entry)", folderEntriesText("({{.count}} entries)", "(1 entry)", 1))
	assert.Equal(t, "(0 entries)", folderEntriesText("({{.count}} entries)", "(1 entry)", 0))
	assert.Equal(t, "(5 entries)", folderEntriesText("({{.count}} entries)", "(1 entry)", 5))
	assert.Equal(t, "Folder: src/ (5 entries)", folderListingTitle("Folder: src/", "(5 entries)"))
}

func TestSubmoduleListingText(t *testing.T) {
	subjects := map[string]string{"1a2b3c4d": "Fix the parser", "9f8e7d6c": ""}
	subject := func(hash string) string { return subjects[hash] }
	labels := submoduleLabels{commit: "Commit:", previous: "Previous:", removed: "none, removed by this commit"}
	current := &git_commands.TreeEntry{Path: "libs/foo", Type: "commit", Object: "1a2b3c4d", Size: -1}
	previous := &git_commands.TreeEntry{Path: "libs/foo", Type: "commit", Object: "9f8e7d6c", Size: -1}

	tests := []struct {
		name     string
		current  *git_commands.TreeEntry
		previous *git_commands.TreeEntry
		want     string
	}{
		{
			name:     "changed submodule shows both commits",
			current:  current,
			previous: previous,
			want:     "Commit:    1a2b3c4d Fix the parser\nPrevious:  9f8e7d6c",
		},
		{
			name:    "unchanged submodule shows its commit",
			current: current,
			want:    "Commit:    1a2b3c4d Fix the parser",
		},
		{
			name:     "removed submodule shows the commit before it",
			previous: previous,
			want:     "Commit:    none, removed by this commit\nPrevious:  9f8e7d6c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, submoduleListingText(labels, tt.current, tt.previous, subject))
		})
	}
}
