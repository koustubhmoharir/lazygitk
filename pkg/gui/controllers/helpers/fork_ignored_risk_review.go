package helpers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the review of ignored or untracked files that an operation may overwrite
// is shared by the rebase starts (controllers/fork_rebase_ignored_review.go) and
// the hard reset (captured_reset.go). The approval leads, so the reviewed action
// is selected; Cancel follows; each path row reopens the review and never approves.

// IgnoredRiskReview describes one review of the files an operation would overwrite.
type IgnoredRiskReview struct {
	// CommandPrefix names the menu's command IDs, such as "rebase" or "reset".
	CommandPrefix string
	Title         string
	Prompt        string
	ApproveLabel  string
	Paths         []string
	// Approve runs the reviewed action under its approval.
	Approve func() error
	// Reopen shows this review again, for a path row.
	Reopen func() error
}

// ShowIgnoredRiskMenu opens the review. The menu hides the default cancel, because
// Cancel is one of its own rows.
func ShowIgnoredRiskMenu(c *HelperCommon, review IgnoredRiskReview) error {
	items := []*types.MenuItem{
		{
			CommandID: review.CommandPrefix + ".approve-ignored-overwrite", Verb: review.CommandPrefix,
			Label: review.ApproveLabel, Keys: menuKey('p'),
			OnPress: review.Approve,
		},
		{
			CommandID: review.CommandPrefix + ".cancel-ignored-risk", Verb: "cancel",
			Label: c.Tr.Cancel, OnPress: func() error { return nil },
		},
	}
	// File rows keep the full risk list scrollable. Enter on a file reopens the
	// review, and never approves it.
	for _, path := range review.Paths {
		quoted := fmt.Sprintf("%q", path)
		items = append(items, &types.MenuItem{
			CommandID: review.CommandPrefix + ".inspect-ignored-path", Verb: "inspect",
			Label: quoted, Tooltip: quoted, OpensMenu: true,
			OnPress: review.Reopen,
		})
	}
	return c.Menu(types.CreateMenuOptions{
		Title: review.Title, Prompt: review.Prompt,
		HideCancel: true, Items: items,
	})
}
