package helpers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/gui/modes/filtering"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the commit scope filters (w a, w d) show the selected commits with their
// ancestors or descendants. Unlike the path filter they keep the screen mode, and
// they are applied and cleared the same way otherwise (see ModeHelper).

// SetCommitScopeFilter applies scope in place of any path, author or scope filter.
// The cursor stays on the commit it is on, and Commits is focused.
func (self *ModeHelper) SetCommitScopeFilter(scope filtering.CommitScope) error {
	cursorHash := self.c.Contexts().LocalCommits.GetSelectedCommitHash()

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

// commitScopeFilteringLayout leaves the screen mode alone while a commit scope is
// applied, and focuses Commits. Clearing a filter still restores a half screen,
// which a path filter may have left behind when a scope replaced it.
func (self *ModeHelper) commitScopeFilteringLayout(filtering bool) {
	repoState := self.c.State().GetRepoState()
	if !filtering && repoState.GetScreenMode() == types.SCREEN_HALF {
		repoState.SetScreenMode(types.SCREEN_NORMAL)
	}
	self.c.Context().Push(self.c.Contexts().LocalCommits, types.OnFocusOpts{})
}

// commitScopeInfoLabel describes the active scope, with the reset button.
func (self *ModeHelper) commitScopeInfoLabel() string {
	scope := self.c.Modes().Filtering.GetCommitScope()
	count := len(scope.Hashes)

	var text string
	switch {
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
