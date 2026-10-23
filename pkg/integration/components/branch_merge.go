package components

import "fmt"

// ReviewBranchMerge verifies the explicit current-destination merge review.
func (self *Common) ReviewBranchMerge(currentBranch, sourceBranch, mode string) {
	self.t.ExpectPopup().Confirmation().
		Title(Equals("Merge")).
		Content(Contains(fmt.Sprintf("Current branch: %q", currentBranch)).
			Contains(fmt.Sprintf("Selected source: %q", sourceBranch)).
			Contains("Mode: " + mode)).
		Confirm()
}
