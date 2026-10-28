package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// Fork: roadmap item 22. forkPreviousCheckout reads the reflog, newest first.
func TestForkPreviousCheckout(t *testing.T) {
	reflogEntry := func(name string) *models.Commit {
		return models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Name: name})
	}
	const fullSHA1 = "0123456789abcdef0123456789abcdef01234567"
	const fullSHA256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name   string
		reflog []string
		want   string
	}{
		{
			name:   "newest checkout entry wins",
			reflog: []string{"checkout: moving from feature to main", "checkout: moving from old to feature"},
			want:   "feature",
		},
		{
			name:   "non-checkout entries are skipped",
			reflog: []string{"commit: x", "rebase (finish): returning to refs/heads/a", "checkout: moving from b to c"},
			want:   "b",
		},
		{
			name:   "full sha1 source is shortened",
			reflog: []string{"checkout: moving from " + fullSHA1 + " to main"},
			want:   utils.ShortHash(fullSHA1),
		},
		{
			name:   "full sha256 source is shortened",
			reflog: []string{"checkout: moving from " + fullSHA256 + " to main"},
			want:   utils.ShortHash(fullSHA256),
		},
		{
			name:   "no checkout entry gives empty",
			reflog: []string{"commit: x", "reset: moving to HEAD~1"},
			want:   "",
		},
		{
			name:   "empty reflog gives empty",
			reflog: nil,
			want:   "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var commits []*models.Commit
			for _, name := range test.reflog {
				commits = append(commits, reflogEntry(name))
			}
			assert.Equal(t, test.want, forkPreviousCheckout(commits))
		})
	}
}
