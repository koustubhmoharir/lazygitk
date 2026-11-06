package helpers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// Fork: undo and redo of a checkout (controllers/undo_controller.go) replace
// upstream's CheckoutRef, which runs a plain `git checkout` and offers an
// autostash. This checks out the reflog's target with the same guarded flows as
// s c and s d: the named checkout of a local branch, or the captured detached
// checkout of a commit. Nothing is stashed, and ignored files are not overwritten.

// CheckoutReflogTarget checks out target, the reflog text of an undo or redo of a
// checkout. An existing local branch of that name gets the named checkout. An
// abbreviated or full commit ID that Git resolves to one commit gets the captured
// detached checkout. Anything else is refused with a message that names it.
// envVars are set on the Git command that writes the reflog, so the entry is
// marked as an undo or redo.
func (self *RefsHelper) CheckoutReflogTarget(target string, envVars []string) error {
	repository := captureActionRepository(self.c)
	return self.c.WithWaitingStatus(self.c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		resolved, err := repository.git.Branch.ClassifyReflogCheckoutTarget(target)
		if err != nil {
			return err
		}
		switch resolved.Kind {
		case git_commands.ReflogCheckoutBranch:
			oid, err := repository.git.Branch.ResolveBranchActionCommit(resolved.FullRef)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(resolved.FullRef, "refs/heads/")
			self.c.OnUIThread(func() error {
				flow := self.namedCheckoutFlowAt(oid, repository)
				flow.envVars = envVars
				return flow.prepare(name)
			})
		case git_commands.ReflogCheckoutCommit:
			oid := resolved.OID
			self.c.OnUIThread(func() error {
				return self.checkoutCapturedDetached(oid, repository, envVars...)
			})
		default:
			return fmt.Errorf(self.c.Tr.UndoCheckoutUnavailable, target)
		}
		return nil
	})
}
