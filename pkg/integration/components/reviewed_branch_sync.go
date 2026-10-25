package components

import "fmt"

// ReviewCurrentBranchPull covers the standard named-remote tracking namespace.
// It approves only the fetch and configured method, returning the checked final
// integration confirmation so callers can verify state before approving it.
// Custom storage and local upstream fixtures review those choices explicitly.
func (self *TestDriver) ReviewCurrentBranchPull(name, remote, upstream, sourceOID, targetOID string) *ConfirmationDriver {
	self.ExpectPopup().Menu().Title(Equals("Pull checked-out branch")).
		Select(Contains("Fetch upstream for review")).Confirm()
	self.ConfirmFastForwardFetch(name, remote, upstream)
	self.ExpectPopup().Menu().Title(Equals("Integrate reviewed incoming commit")).
		Select(MatchesRegexp(`Use Git's configured pull method|Configured (merge|rebase)`)).Confirm()
	return self.ExpectPopup().Confirmation().Title(Equals("Review current-branch integration")).
		Content(Contains(fmt.Sprintf("branch %q", name)).Contains(sourceOID).Contains(targetOID))
}

// ReviewBranchForcePush returns the Cancel-first reviewed lease menu without
// authorizing a force push. The caller must explicitly select its execute row.
func (self *TestDriver) ReviewBranchForcePush(name, destination, sourceOID, leaseRef, leaseOID string) *MenuDriver {
	self.ExpectPopup().Menu().Title(Equals("Push selected branch")).
		Select(Contains("Force with reviewed lease")).Confirm()
	menu := self.ExpectPopup().Menu().Title(Equals("Confirm force with lease"))
	self.Views().Menu().Content(Contains(fmt.Sprintf("source %q", name)).Contains(sourceOID).
		Contains(fmt.Sprintf("%q", "refs/heads/"+destination)).
		Contains("using lease").Contains(leaseRef).Contains(leaseOID))
	menu.ContainsLines(Contains("Cancel").IsSelected())
	return menu
}
