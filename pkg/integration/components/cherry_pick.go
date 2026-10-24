package components

import (
	"fmt"
	"strings"
)

// ChooseMarkedCherryPickSource selects the copied/marked commits explicitly
// when the source chooser is shown alongside a selected-commit source.
func (self *Common) ChooseMarkedCherryPickSource() {
	self.t.ExpectPopup().Menu().Title(Equals("Choose commits to cherry-pick")).
		Select(Contains("Previously marked commits")).Confirm()
}

func (self *Common) ChooseFirstCherryPickMergeParent() {
	self.t.ExpectPopup().Menu().Title(Equals("Choose the merge parent to compare against")).
		Select(Contains("Use parent 1 of each selected merge")).Confirm()
}

// ConfirmCherryPickList checks the full reviewed destination and ordered rows.
// Callers capture headOID and commit OIDs before changing branches.
func (self *Common) ConfirmCherryPickList(destination, headOID string, rows ...string) {
	content := fmt.Sprintf("Current destination: %s at %s\n\nCherry-pick these commits in this order:\n%s", destination, headOID, strings.Join(rows, "\n"))
	self.t.ExpectPopup().Alert().Title(Equals("Cherry-pick")).Content(Contains(content)).Confirm()
}

func CherryPickReviewRow(index int, oid, subject string) string {
	return fmt.Sprintf("%d. %s %s", index, oid, subject)
}
