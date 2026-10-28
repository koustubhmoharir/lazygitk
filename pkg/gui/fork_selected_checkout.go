package gui

import "github.com/jesseduffield/lazygit/pkg/commands/models"

// Fork: the default s c routes run the flat checkout handlers in
// helpers/fork_flat_checkout.go instead of upstream's second-level checkout
// menus. The selection is read when the key is pressed. Explicit legacy
// bindings keep their original handlers. view picks the commit context for the
// commit route.
func (gui *Gui) forkSelectedCheckoutHandler(id, view string) func() error {
	switch id {
	case "branch.checkout":
		return func() error {
			if branch := gui.c.Contexts().Branches.GetSelected(); branch != nil {
				return gui.helpers.Refs.CheckoutSelectedBranch(branch)
			}
			return nil
		}
	case "remote-branch.checkout":
		return func() error {
			if remote := gui.c.Contexts().RemoteBranches.GetSelected(); remote != nil {
				return gui.helpers.Refs.CheckoutSelectedRemoteBranch(remote)
			}
			return nil
		}
	case "tag.checkout":
		return func() error {
			if tag := gui.c.Contexts().Tags.GetSelected(); tag != nil {
				return gui.helpers.Refs.NameCheckoutAtSelectedRef(tag.FullRefName())
			}
			return nil
		}
	case "commit.checkout-menu":
		return func() error {
			if commit := gui.forkSelectedCommit(view); commit != nil {
				return gui.helpers.Refs.NameCheckoutAtCommit(commit.Hash())
			}
			return nil
		}
	}
	return nil
}

// forkSelectedCommit returns the selected commit of the commit pane named by
// view, or nil for any other pane.
func (gui *Gui) forkSelectedCommit(view string) *models.Commit {
	switch view {
	case "commits":
		return gui.c.Contexts().LocalCommits.GetSelected()
	case "reflogCommits":
		return gui.c.Contexts().ReflogCommits.GetSelected()
	case "subCommits":
		return gui.c.Contexts().SubCommits.GetSelected()
	}
	return nil
}
