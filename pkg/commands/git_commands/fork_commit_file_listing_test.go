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
	treePaths := []string{
		"changed.txt",
		"dir/added.txt",
		"dir/untouched.txt",
		"new name.txt",
		"untouched.txt",
	}

	got := MergeTreePathsIntoCommitFiles(diffFiles, treePaths)

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
	got := MergeTreePathsIntoCommitFiles(nil, []string{"a.txt"})
	assert.Equal(t, []*models.CommitFile{{Path: "a.txt"}}, got)
}

func TestMergeTreePathsIntoCommitFilesDoesNotMutateDiffFiles(t *testing.T) {
	diffFiles := []*models.CommitFile{{ChangeStatus: "M", Path: "a.txt"}}
	MergeTreePathsIntoCommitFiles(diffFiles, []string{"a.txt", "b.txt"})
	assert.Equal(t, []*models.CommitFile{{ChangeStatus: "M", Path: "a.txt"}}, diffFiles)
}

func TestGetFilesInTreeListsEveryPathOfTheCommit(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"ls-tree", "-r", "-z", "--name-only", "--full-tree", "abc123"}, "a/b.txt\x00c with space.txt\x00", nil)
	gitCommon := buildGitCommon(commonDeps{runner: runner})
	loader := NewCommitFileLoader(gitCommon.Common, gitCommon.cmd)

	paths, err := loader.GetFilesInTree("abc123")

	assert.NoError(t, err)
	assert.Equal(t, []string{"a/b.txt", "c with space.txt"}, paths)
	runner.CheckForMissingCalls()
}
