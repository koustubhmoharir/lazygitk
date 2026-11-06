package helpers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the palette action "Checkout by name". Upstream's CheckoutRef and
// CheckoutRemoteBranch run plain `git checkout <input>`, which can overwrite
// ignored files, discard edits when the input is a path, and offer autostash.
// This flow classifies the typed name on a worker (see
// ClassifyCheckoutByName), then dispatches it on the UI thread to the guarded
// checkout flow for that kind of ref. Nothing is passed to checkout as a name.

// CheckoutByNameFlow is one prompt of the palette action. It captures the
// repository and the selected creation source when the prompt opens, so that a
// change of repository or selection before confirming is detected.
type CheckoutByNameFlow struct {
	refs        *RefsHelper
	repository  *capturedActionRepository
	sourceRef   string
	sourceLabel string
}

// CaptureCheckoutByName captures the prompt's repository and the selected
// branch, which is the source for a branch created when the name is not found.
func (self *RefsHelper) CaptureCheckoutByName(selected *models.Branch) *CheckoutByNameFlow {
	flow := &CheckoutByNameFlow{refs: self, repository: captureActionRepository(self.c)}
	if selected != nil {
		flow.sourceRef, flow.sourceLabel = selected.FullRefName(), selected.RefName()
	}
	return flow
}

// CheckRepository reports whether the repository changed since the prompt opened.
func (flow *CheckoutByNameFlow) CheckRepository() error {
	return flow.repository.checkRepository()
}

// Confirm handles the entered name. It classifies the trimmed name on a worker
// against the captured repository, then dispatches on the UI thread.
func (flow *CheckoutByNameFlow) Confirm(response string) error {
	c := flow.refs.c
	input := strings.TrimSpace(response)
	if err := flow.CheckRepository(); err != nil {
		return err
	}
	c.LogAction("Checkout branch")
	if input == "" {
		return errors.New(c.Tr.CheckoutByNameEmpty)
	}
	return c.WithWaitingStatus(c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		if err := flow.CheckRepository(); err != nil {
			return err
		}
		target, err := flow.repository.git.Branch.ClassifyCheckoutByName(input)
		if err != nil {
			return err
		}
		c.OnUIThread(func() error {
			return flow.dispatch(input, target)
		})
		return nil
	})
}

func (flow *CheckoutByNameFlow) dispatch(input string, target git_commands.CheckoutByNameTarget) error {
	c := flow.refs.c
	if err := flow.CheckRepository(); err != nil {
		return err
	}
	switch target.Kind {
	case git_commands.CheckoutByNamePrevious:
		return flow.refs.CheckoutPreviousFlat()
	case git_commands.CheckoutByNameLocalBranch:
		return flow.refs.checkoutLocalBranchNamed(strings.TrimPrefix(target.FullRef, "refs/heads/"))
	case git_commands.CheckoutByNameRemoteBranch:
		// The model supplies the RemoteBranch that s c on a remote branch uses.
		fullName := strings.TrimPrefix(target.FullRef, "refs/remotes/")
		for _, remote := range c.Model().RemoteBranches {
			if remote.FullName() == fullName {
				return flow.refs.CheckoutSelectedRemoteBranch(remote)
			}
		}
		return fmt.Errorf(c.Tr.CheckoutByNameRemoteNotLoaded, fullName)
	case git_commands.CheckoutByNameTag:
		return flow.refs.CheckoutSelectedRefDetached(target.FullRef)
	case git_commands.CheckoutByNameCommit:
		return flow.refs.CheckoutCommitDetached(target.OID)
	case git_commands.CheckoutByNameAmbiguous:
		return fmt.Errorf(c.Tr.CheckoutByNameAmbiguous, input, strings.Join(target.Refs, ", "))
	}
	// Not found. A full ref is never offered as a new branch name.
	if strings.HasPrefix(input, "refs/") {
		return fmt.Errorf(c.Tr.CheckoutByNameNoRef, input)
	}
	return flow.confirmCreate(input)
}

// confirmCreate offers to create the branch from the selected branch, as before.
func (flow *CheckoutByNameFlow) confirmCreate(name string) error {
	c := flow.refs.c
	c.Confirm(types.ConfirmOpts{
		Title:       c.Tr.BranchNotFoundTitle,
		Prompt:      fmt.Sprintf("%s %s?", c.Tr.BranchNotFoundPrompt, name),
		HandleClose: flow.CheckRepository,
		HandleConfirm: func() error {
			if err := flow.CheckRepository(); err != nil {
				return err
			}
			if flow.sourceRef == "" {
				return errors.New(c.Tr.NoBranchesThisRepo)
			}
			return flow.refs.NewBranchWithNameFromRef(flow.sourceRef, flow.sourceLabel, name)
		},
	})
	return nil
}
