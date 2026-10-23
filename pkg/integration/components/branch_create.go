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
	menu.getViewDriver().Content(Contains("Source: " + sourceRef).Contains("(" + sourceOID + ")").Contains("Current HEAD: " + headRef).Contains(headOID))
	for _, label := range []string{
		"Do not set up upstream tracking",
		"Use Git's configured tracking policy",
		"Track the selected source branch directly",
	} {
		menu.ContainsLines(MatchesRegexp(`^[[:space:]]*` + label))
	}
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
	ref := "refs/heads/" + name
	self.t.Git().expect([]string{"git", "for-each-ref", "--format=%(refname)", "--", ref}, func(output string) (bool, string) {
		for _, actual := range strings.Split(output, "\n") {
			if actual == ref {
				return false, fmt.Sprintf("Expected %s to be absent", ref)
			}
		}
		return true, ""
	})
	return self
}
