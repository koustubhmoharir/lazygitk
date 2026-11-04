package components

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
)

// CherryPickSelection presses the paste key in view, which cherry-picks the commits
// the view's selection covers. The selection must be made first.
func (self *Common) CherryPickSelection(view *ViewDriver) {
	view.Press(self.t.keys.Commits.PasteCommits)
}

// CherryPickSelectionWithDefaultKeys picks the view's selection with the default
// keys, c then p, for tests that load the real defaults with shared.UseDefaultKeymap.
func (self *Common) CherryPickSelectionWithDefaultKeys(view *ViewDriver) {
	view.Press(config.Keybinding{"c"})
	self.t.ExpectPopup().Menu().Title(Equals("c"))
	self.t.Views().Menu().Press(config.Keybinding{"p"})
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
