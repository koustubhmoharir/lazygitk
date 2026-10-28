package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// CreateSelectedRefCheckoutMenu adds naming choices to the default branch/tag
// switch action. Explicit legacy shortcuts retain their upstream handlers.
func (self *RefsHelper) CreateSelectedRefCheckoutMenu(ref models.Ref) error {
	git, generation := self.c.Git(), self.c.State().GetRepoGeneration()
	source := ref.FullRefName()
	localName := ""
	if branch, ok := ref.(*models.Branch); ok && !branch.DetachedHead {
		localName = branch.Name
	}
	valid := func() error {
		if self.c.Git() != git || self.c.State().GetRepoGeneration() != generation {
			return errors.New(self.c.Tr.NamedCheckoutRepositoryChanged)
		}
		return nil
	}
	return self.c.WithWaitingStatus(self.c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		if err := valid(); err != nil {
			return err
		}
		oid, err := git.Branch.ResolveBranchActionCommit(source)
		if err != nil {
			return err
		}
		self.c.OnUIThread(func() error {
			if err := valid(); err != nil {
				return err
			}
			flow := &namedCheckoutFlow{c: self.c, git: git, oid: oid, validateRepository: valid}
			items := make([]*types.MenuItem, 0, 3)
			if localName != "" {
				items = append(items, &types.MenuItem{
					CommandID: "branch.checkout", Verb: "checkout", Keys: menuKey('s'),
					Label: fmt.Sprintf(self.c.Tr.Actions.CheckoutBranchAtCommit, localName),
					OnPress: func() error {
						if err := valid(); err != nil {
							return err
						}
						return flow.prepare(localName)
					},
				})
			}
			named := self.namedCheckoutItem(oid)
			named.OnPress = flow.begin
			items = append(items, named, &types.MenuItem{
				CommandID: "commit.checkout-detached", Verb: "checkout", Keys: menuKey('d'),
				Label: fmt.Sprintf(self.c.Tr.Actions.CheckoutCommitAsDetachedHead, utils.ShortHash(oid)),
				OnPress: func() error {
					if err := valid(); err != nil {
						return err
					}
					return self.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: self.c.Tr.CheckingOutStatus}, func(gocui.Task) error {
						if err := valid(); err != nil {
							return err
						}
						err := git.Branch.CheckoutCapturedCommit(oid)
						self.c.RefreshFromWorker(namedCheckoutFailureRefreshOptions())
						return err
					})
				},
			})
			return self.c.Menu(types.CreateMenuOptions{Title: self.c.Tr.Actions.CheckoutBranchOrCommit, Items: items})
		})
		return nil
	})
}
