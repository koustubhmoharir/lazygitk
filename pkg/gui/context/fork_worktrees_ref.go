package context

import "github.com/jesseduffield/lazygit/pkg/commands/models"

// Fork: Worktrees can open the read-only sub-commits view (roadmap item 22),
// so v shows a worktree's history without switching to it. The ref is the
// worktree's checked-out branch, or its HEAD commit when it is detached or
// mid-rebase/bisect, which is what that worktree actually has checked out.
func (self *WorktreesContext) GetSelectedRef() models.Ref {
	worktree := self.GetSelected()
	if worktree == nil {
		return nil
	}
	if worktree.Branch != "" && !worktree.IsRebasingOrBisecting {
		for _, branch := range self.c.Model().Branches {
			if branch.Name == worktree.Branch {
				return branch
			}
		}
	}
	if worktree.Head == "" {
		return nil
	}
	return models.NewCommit(self.c.Model().HashPool, models.NewCommitOpts{Hash: worktree.Head})
}

func (self *WorktreesContext) ShowBranchHeadsInSubCommits() bool {
	return true
}
