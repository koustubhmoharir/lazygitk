package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestMergeTreePathsIntoCommitFiles(t *testing.T) {
	diffFiles := []*models.CommitFile{
		{ChangeStatus: "M", Path: "changed.txt"},
		{ChangeStatus: "A", Path: "dir/added.txt"},
		{ChangeStatus: "D", Path: "gone.txt"},
		{ChangeStatus: "R", PreviousPath: "old name.txt", Path: "new name.txt"},
	}
	treeFiles := []*models.CommitFile{
		{Path: "changed.txt"},
		{Path: "dir/added.txt"},
		{Path: "dir/untouched.txt"},
		{Path: "new name.txt"},
		{Path: "untouched.txt"},
	}

	got := MergeTreePathsIntoCommitFiles(diffFiles, treeFiles, nil)

	assert.Equal(t, []*models.CommitFile{
		{ChangeStatus: "M", Path: "changed.txt"},
		{ChangeStatus: "A", Path: "dir/added.txt"},
		{ChangeStatus: "D", Path: "gone.txt"},
		{ChangeStatus: "R", PreviousPath: "old name.txt", Path: "new name.txt"},
		{Path: "dir/untouched.txt"},
		{Path: "untouched.txt"},
	}, got)
}

func TestMergeTreePathsIntoCommitFilesWithoutDiff(t *testing.T) {
	got := MergeTreePathsIntoCommitFiles(nil, []*models.CommitFile{{Path: "a.txt"}}, nil)
	assert.Equal(t, []*models.CommitFile{{Path: "a.txt"}}, got)
}

func TestMergeTreePathsIntoCommitFilesDoesNotMutateDiffFiles(t *testing.T) {
	diffFiles := []*models.CommitFile{{ChangeStatus: "M", Path: "a.txt"}, {ChangeStatus: "D", Path: "sub"}}
	fromTree := []*models.CommitFile{{Path: "sub", Submodule: true}}
	got := MergeTreePathsIntoCommitFiles(diffFiles, []*models.CommitFile{{Path: "a.txt"}, {Path: "b.txt"}}, fromTree)
	assert.Equal(t, []*models.CommitFile{{ChangeStatus: "M", Path: "a.txt"}, {ChangeStatus: "D", Path: "sub"}}, diffFiles)
	assert.Equal(t, []*models.CommitFile{
		{ChangeStatus: "M", Path: "a.txt"},
		{ChangeStatus: "D", Path: "sub", Submodule: true},
		{Path: "b.txt"},
	}, got)
}

func TestMergeTreePathsMarksDeletedSubmodules(t *testing.T) {
	diffFiles := []*models.CommitFile{{ChangeStatus: "D", Path: "sub"}, {ChangeStatus: "D", Path: "file"}}
	fromTree := []*models.CommitFile{{Path: "sub", Submodule: true}, {Path: "file"}}
	got := MergeTreePathsIntoCommitFiles(diffFiles, nil, fromTree)
	assert.Equal(t, []*models.CommitFile{
		{ChangeStatus: "D", Path: "sub", Submodule: true},
		{ChangeStatus: "D", Path: "file"},
	}, got)
}

func TestGetFilesInTreeListsEveryPathOfTheCommit(t *testing.T) {
	const blob = "100644 blob 1111111111111111111111111111111111111111"
	const commit = "160000 commit 2222222222222222222222222222222222222222"
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"ls-tree", "-r", "-z", "--full-tree", "abc123"},
			blob+"\ta/b.txt\x00"+blob+"\tc with space.txt\x00"+commit+"\tsub\x00"+blob+"\ta\tb.txt\x00", nil)
	gitCommon := buildGitCommon(commonDeps{runner: runner})
	loader := NewCommitFileLoader(gitCommon.Common, gitCommon.cmd)

	files, err := loader.GetFilesInTree("abc123")

	assert.NoError(t, err)
	assert.Equal(t, []*models.CommitFile{
		{Path: "a/b.txt"},
		{Path: "c with space.txt"},
		{Path: "sub", Submodule: true},
		{Path: "a\tb.txt"},
	}, files)
	runner.CheckForMissingCalls()
}

func TestTreeFilesFromListingRejectsMalformedRecords(t *testing.T) {
	_, err := treeFilesFromListing("blob-without-tab\x00")
	assert.Error(t, err)
}

func TestTreeEntriesFromLongListing(t *testing.T) {
	output := "100644 blob 1111111111111111111111111111111111111111    312\tnew.go\x00" +
		"040000 tree 2222222222222222222222222222222222222222       -\tgui\x00" +
		"160000 commit 3333333333333333333333333333333333333333       -\tlibs/foo\x00" +
		"100644 blob 4444444444444444444444444444444444444444  12345\tdir with spaces.txt\x00"

	entries, err := treeEntriesFromLongListing(output)

	assert.NoError(t, err)
	assert.Equal(t, []TreeEntry{
		{Path: "new.go", Type: "blob", Object: "1111111111111111111111111111111111111111", Size: 312},
		{Path: "gui", Type: "tree", Object: "2222222222222222222222222222222222222222", Size: -1},
		{Path: "libs/foo", Type: "commit", Object: "3333333333333333333333333333333333333333", Size: -1},
		{Path: "dir with spaces.txt", Type: "blob", Object: "4444444444444444444444444444444444444444", Size: 12345},
	}, entries)
}

func TestTreeEntriesFromLongListingEmpty(t *testing.T) {
	entries, err := treeEntriesFromLongListing("")
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTreeEntriesFromLongListingMalformed(t *testing.T) {
	_, err := treeEntriesFromLongListing("100644 blob 1111\tnew.go\x00")
	assert.Error(t, err)
	_, err = treeEntriesFromLongListing("100644 blob 1111 abc\tnew.go\x00")
	assert.Error(t, err)
}

func TestMergeTreePathsMarksChangedSubmodules(t *testing.T) {
	diffFiles := []*models.CommitFile{
		{ChangeStatus: "M", Path: "sub"},
		{ChangeStatus: "A", Path: "added_sub"},
		{ChangeStatus: "M", Path: "file"},
	}
	treeFiles := []*models.CommitFile{
		{Path: "added_sub", Submodule: true},
		{Path: "file"},
		{Path: "sub", Submodule: true},
	}
	got := MergeTreePathsIntoCommitFiles(diffFiles, treeFiles, nil)
	assert.Equal(t, []*models.CommitFile{
		{ChangeStatus: "M", Path: "sub", Submodule: true},
		{ChangeStatus: "A", Path: "added_sub", Submodule: true},
		{ChangeStatus: "M", Path: "file"},
	}, got)
	// The diff's own files are not changed.
	assert.False(t, diffFiles[0].Submodule)
}
