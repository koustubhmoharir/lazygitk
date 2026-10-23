package components

import (
	"fmt"
	"strings"
)

// ReviewBranchCreateMenu verifies the captured source and HEAD before a policy
// is chosen. Match the three policy rows by their stable leading wording.
func (self *Common) ReviewBranchCreateMenu(sourceRef, sourceOID, headRef, headOID string) *MenuDriver {
	menu := self.t.ExpectPopup().Menu()
	menu.Title(Equals("Create branch and check it out"))
	menu.getViewDriver().Content(Contains(fmt.Sprintf("Source: %s (%s)", sourceRef, sourceOID)).Contains(fmt.Sprintf("Current HEAD: %s (%s)", headRef, headOID)))
	menu.ContainsLines(
		MatchesRegexp(`^[[:space:]]*Do not set up upstream tracking`),
		MatchesRegexp(`^[[:space:]]*Use Git's configured tracking policy`),
		MatchesRegexp(`^[[:space:]]*Track the selected source branch directly`),
	)
	return menu
}

// AssertNoStashes checks that a refused checkout did not create or pop a stash.
func (self *Common) AssertNoStashes() *Common {
	self.t.Git().expect([]string{"git", "stash", "list"}, func(output string) (bool, string) {
		actual := strings.TrimSpace(output)
		return actual == "", fmt.Sprintf("Expected no stashes, but got %q", actual)
	})
	return self
}

// AssertBranchRefAbsent checks that a refused creation left no target ref.
func (self *Common) AssertBranchRefAbsent(name string) *Common {
	self.t.Git().assertWithRetries(func() (bool, string) {
		output, err := self.t.Git().shell.runCommandWithOutput([]string{"git", "show-ref", "--verify", "--hash", "refs/heads/" + name})
		if err != nil {
			return true, ""
		}
		return false, fmt.Sprintf("Expected refs/heads/%s to be absent, but it resolves to %s", name, strings.TrimSpace(output))
	})
	return self
}
