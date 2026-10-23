package components

import (
	"fmt"
	"strings"
)

// ReviewBranchDeleteMenu checks the captured refs/HEAD shown by branch deletion
// and the stable accelerator-prefixed menu entries.
func (self *Common) ReviewBranchDeleteMenu(title, sourceRef, sourceOID, headRef, headOID string, reserved bool) *MenuDriver {
	menu := self.t.ExpectPopup().Menu()
	menu.Title(Equals(title))
	menu.getViewDriver().Content(Contains(sourceRef + " (" + sourceOID + ")").Contains("Current HEAD:").Contains(fmt.Sprintf("%q at %s", headRef, headOID)).DoesNotContain("Detach worktree").DoesNotContain("Remove worktree"))
	menu.ContainsLines(
		MatchesRegexp(`^[[:space:]]*c Delete local branch`),
		MatchesRegexp(`^[[:space:]]*r Delete remote branch`),
		MatchesRegexp(`^[[:space:]]*b Delete local and remote branch`),
	)
	if reserved {
		reservation := Contains("reserved").Contains(strings.TrimPrefix(sourceRef, "refs/heads/"))
		menu.Select(MatchesRegexp(`^[[:space:]]*c Delete local branch`)).Tooltip(reservation)
		menu.Select(MatchesRegexp(`^[[:space:]]*b Delete local and remote branch`)).Tooltip(reservation)
	}
	return menu
}

// ReviewBranchDeleteConfirmation checks the captured ref and history warning.
func (self *Common) ReviewBranchDeleteConfirmation(title, sourceRef, sourceOID, headOID string) *ConfirmationDriver {
	return self.t.ExpectPopup().Confirmation().
		Title(Equals(title)).
		Content(Contains(fmt.Sprintf("%s (%s)", sourceRef, sourceOID)).Contains(headOID))
}

// AssertRefOID checks a fully-qualified Git ref without resolving an ambiguous name.
func (self *Common) AssertRefOID(ref, expected string) *Common {
	self.t.assertWithRetries(func() (bool, string) {
		actual := self.t.Git().GetCommitHash(ref)
		return actual == expected, fmt.Sprintf("Expected %s to remain at %s, but got %s", ref, expected, actual)
	})
	return self
}
