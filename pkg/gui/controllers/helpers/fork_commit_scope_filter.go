package helpers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/filtering"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the commit scope filters (w a, w d, w b) show the selected commits with their
// ancestors or descendants, or a branch, remote branch or tag with its history. Unlike
// the path filter they keep the screen mode, and they are applied and cleared the same
// way otherwise (see ModeHelper).

// SetCommitScopeFilter applies scope in place of any path, author or scope filter.
// cursorHash is the commit the cursor is on, which stays selected, and Commits is
// focused. Commits passes its own selection; Branches passes the branch's tip,
// because its own selection is not a commit.
func (self *ModeHelper) SetCommitScopeFilter(scope filtering.CommitScope, cursorHash string) error {
	return self.changeFilteringWithLayout(
		func() {
			// As for path filters, the scope replaces whatever filter was set.
			self.c.Modes().Filtering.Reset()
			self.c.Modes().Filtering.SetCommitScope(scope)
			self.c.Modes().Filtering.SetSelectedCommitHash(cursorHash)
		},
		func() {
			// The cursor commit is one of the scope's commits, so it stays visible.
			if !self.c.Contexts().LocalCommits.SelectCommitByHash(cursorHash) {
				self.c.Contexts().LocalCommits.SetSelection(0)
			}
		},
		self.commitScopeFilteringLayout,
	)
}

// commitScopeFilteringLayout leaves the screen mode alone when a commit scope is
// applied or cleared, and focuses Commits. The screen mode is the user's: a half
// screen may have come from + as well as from a path filter that a scope replaced,
// and the two cannot be told apart.
func (self *ModeHelper) commitScopeFilteringLayout(bool) {
	self.c.Context().Push(self.c.Contexts().LocalCommits, types.OnFocusOpts{})
}

// commitScopeInfoLabel describes the active scope, with the reset button.
func (self *ModeHelper) commitScopeInfoLabel() string {
	scope := self.c.Modes().Filtering.GetCommitScope()
	count := len(scope.Hashes)

	var text string
	switch {
	case scope.Ref != "":
		text = self.refScopeInfoText(scope.Ref)
	case scope.Descendants && count == 1:
		text = fmt.Sprintf(self.c.Tr.ShowingCommitDescendantsOne, count)
	case scope.Descendants:
		text = fmt.Sprintf(self.c.Tr.ShowingCommitDescendantsMany, count)
	case count == 1:
		text = fmt.Sprintf(self.c.Tr.ShowingCommitAncestorsOne, count)
	default:
		text = fmt.Sprintf(self.c.Tr.ShowingCommitAncestorsMany, count)
	}
	return self.withResetButton(text, style.FgRed)
}

// refScopeInfoText describes a ref scope. A local branch names its upstream, as the
// model knows it when the label is drawn; the loader uses the same upstream, so a gone
// upstream counts as none.
func (self *ModeHelper) refScopeInfoText(ref string) string {
	if name, isBranch := strings.CutPrefix(ref, "refs/heads/"); isBranch {
		if upstream, ok := LocalBranchScopeUpstream(self.c.Model().Branches, name); ok {
			return fmt.Sprintf(self.c.Tr.ShowingBranchWithUpstream, name, upstream)
		}
		return fmt.Sprintf(self.c.Tr.ShowingBranchNoUpstream, name)
	}
	if name, isTag := strings.CutPrefix(ref, "refs/tags/"); isTag {
		return fmt.Sprintf(self.c.Tr.ShowingTagHistory, name)
	}
	return fmt.Sprintf(self.c.Tr.ShowingRemoteBranchHistory, strings.TrimPrefix(ref, "refs/remotes/"))
}

// LocalBranchScopeUpstream returns the upstream that a ref scope shows for the local
// branch called name, and false when it has none. It is shared by the status label
// and the w b menu, so that they agree with the loader about the upstream.
func LocalBranchScopeUpstream(branches []*models.Branch, name string) (string, bool) {
	for _, branch := range branches {
		if branch.Name == name && branch.IsTrackingRemote() && !branch.UpstreamGone {
			return branch.ShortUpstreamRefName(), true
		}
	}
	return "", false
}
