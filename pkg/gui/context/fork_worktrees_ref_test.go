package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

type worktreesRefGuiFake struct {
	types.IGuiCommon
	model *types.Model
}

func (f *worktreesRefGuiFake) Model() *types.Model { return f.model }
func (f *worktreesRefGuiFake) Views() types.Views  { return types.Views{} }

// Fork: roadmap item 22. v in Worktrees views the worktree's checked-out ref.
func newWorktreesRefContext(model *types.Model) *WorktreesContext {
	return NewWorktreesContext(&ContextCommon{IGuiCommon: &worktreesRefGuiFake{model: model}})
}

func TestWorktreesGetSelectedRef(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	feature := &models.Branch{Name: "feature"}

	tests := []struct {
		name     string
		worktree *models.Worktree
		// wantBranch is set when the ref should be the matching branch.
		wantBranch *models.Branch
		// wantCommitHash is set when the ref should be a commit with that hash.
		wantCommitHash string
	}{
		{
			name:       "checked out branch is found",
			worktree:   &models.Worktree{Name: "wt", Branch: "feature", Head: head},
			wantBranch: feature,
		},
		{
			name:           "mid-rebase uses head commit",
			worktree:       &models.Worktree{Name: "wt", Branch: "feature", Head: head, IsRebasingOrBisecting: true},
			wantCommitHash: head,
		},
		{
			name:           "detached uses head commit",
			worktree:       &models.Worktree{Name: "wt", Head: head},
			wantCommitHash: head,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := newWorktreesRefContext(&types.Model{
				Branches:  []*models.Branch{{Name: "main"}, feature},
				Worktrees: []*models.Worktree{test.worktree},
				HashPool:  &utils.StringPool{},
			})
			ref := ctx.GetSelectedRef()
			if test.wantBranch != nil {
				assert.Same(t, test.wantBranch, ref)
			} else {
				commit, ok := ref.(*models.Commit)
				if assert.True(t, ok, "ref should be a commit, got %T", ref) {
					assert.Equal(t, test.wantCommitHash, commit.Hash())
				}
			}
		})
	}
}

func TestWorktreesGetSelectedRefWithoutHeadOrBranchIsNil(t *testing.T) {
	ctx := newWorktreesRefContext(&types.Model{
		Worktrees: []*models.Worktree{{Name: "wt"}},
		HashPool:  &utils.StringPool{},
	})
	assert.Nil(t, ctx.GetSelectedRef())
}

func TestWorktreesGetSelectedRefWithoutWorktreesIsNil(t *testing.T) {
	ctx := newWorktreesRefContext(&types.Model{HashPool: &utils.StringPool{}})
	assert.Nil(t, ctx.GetSelectedRef())
}
