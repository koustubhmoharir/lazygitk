package components

import "fmt"

// ReviewFastForwardFetch opens and approves the fetch review.
func (self *TestDriver) ReviewFastForwardFetch(branchName, remote, upstreamName string) {
	self.ExpectPopup().Menu().
		Title(Equals("Review selected branch fast-forward")).
		Select(Contains("Fetch upstream for review")).
		Confirm()
	self.ConfirmFastForwardFetch(branchName, remote, upstreamName)
}

// ConfirmFastForwardFetch covers fixtures with standard remote-tracking refs.
// Custom namespaces must assert their exact chosen storage separately instead
// of relying on this test-only convenience convention.
func (self *TestDriver) ConfirmFastForwardFetch(branchName, remote, upstreamName string) {
	self.ExpectPopup().Confirmation().
		Title(Equals("Review upstream fetch")).
		Content(Contains(fmt.Sprintf("remote %q", remote)).
			Contains(fmt.Sprintf("branch %q", "refs/heads/"+upstreamName)).
			Contains(fmt.Sprintf("tracking ref %q", "refs/remotes/"+remote+"/"+upstreamName)).
			Contains(fmt.Sprintf("for local branch %q", branchName))).
		Confirm()
}

// Return the checked confirmation so callers can assert that nothing moved
// before approval. This also distinguishes native current-worktree integration
// from a ref-only off-branch update.
func (self *TestDriver) ExpectFastForwardReview(branchName, sourceOID, targetOID string, current bool) *ConfirmationDriver {
	title := "Review branch fast-forward"
	if current {
		title = "Review current-branch integration"
	}
	content := Contains(fmt.Sprintf("branch %q", branchName)).Contains(sourceOID).Contains(targetOID)
	if current {
		content = content.Contains("Fast-forward only")
	}
	return self.ExpectPopup().Confirmation().Title(Equals(title)).Content(content)
}
