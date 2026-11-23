package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestCommitFileShowFor(t *testing.T) {
	tests := []struct {
		name        string
		fullListing bool
		file        *models.CommitFile
		hasParent   bool
		want        CommitFileShow
	}{
		{"changed-files listing shows the diff", false, &models.CommitFile{Path: "a", ChangeStatus: "M"}, true, CommitFileShowDiff},
		{"changed-files listing of a deleted file shows the diff", false, &models.CommitFile{Path: "a", ChangeStatus: "D"}, true, CommitFileShowDiff},
		{"no selection shows the diff", true, nil, true, CommitFileShowDiff},
		{"directory shows the diff", true, nil, true, CommitFileShowDiff},
		{"unchanged file shows its content", true, &models.CommitFile{Path: "a"}, true, CommitFileShowContent},
		{"modified file shows its content", true, &models.CommitFile{Path: "a", ChangeStatus: "M"}, true, CommitFileShowContent},
		{"added file shows its content", true, &models.CommitFile{Path: "a", ChangeStatus: "A"}, true, CommitFileShowContent},
		{"renamed file shows its content", true, &models.CommitFile{Path: "b", PreviousPath: "a", ChangeStatus: "R"}, true, CommitFileShowContent},
		{"deleted file shows its content before the commit", true, &models.CommitFile{Path: "a", ChangeStatus: "D"}, true, CommitFileShowDeletedContent},
		{"deleted file without a parent shows the diff", true, &models.CommitFile{Path: "a", ChangeStatus: "D"}, false, CommitFileShowDiff},
		{"submodule shows the diff", true, &models.CommitFile{Path: "sub", Submodule: true}, true, CommitFileShowDiff},
		{"deleted submodule shows the diff", true, &models.CommitFile{Path: "sub", ChangeStatus: "D", Submodule: true}, true, CommitFileShowDiff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CommitFileShowFor(tt.fullListing, tt.file, tt.hasParent))
		})
	}
}

func TestCommitFileHasParent(t *testing.T) {
	assert.True(t, CommitFileHasParent("abc123^"))
	assert.False(t, CommitFileHasParent(""))
	assert.False(t, CommitFileHasParent(models.EmptyTreeCommitHash))
}
