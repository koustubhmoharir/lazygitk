package gui

import "github.com/jesseduffield/lazygit/pkg/commands/models"

// Default s c adds naming choices to the upstream direct-ref shortcuts.
// Explicit legacy bindings keep their original handlers and menu behavior.
func (gui *Gui) forkSelectedCheckoutHandler(id string) func() error {
	var selected func() models.Ref
	switch id {
	case "branch.checkout":
		selected = func() models.Ref {
			if ref := gui.c.Contexts().Branches.GetSelected(); ref != nil {
				return ref
			}
			return nil
		}
	case "remote-branch.checkout":
		selected = func() models.Ref {
			if ref := gui.c.Contexts().RemoteBranches.GetSelected(); ref != nil {
				return ref
			}
			return nil
		}
	case "tag.checkout":
		selected = func() models.Ref {
			if ref := gui.c.Contexts().Tags.GetSelected(); ref != nil {
				return ref
			}
			return nil
		}
	default:
		return nil
	}
	return func() error {
		if ref := selected(); ref != nil {
			return gui.helpers.Refs.CreateSelectedRefCheckoutMenu(ref)
		}
		return nil
	}
}
