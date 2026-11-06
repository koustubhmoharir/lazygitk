package helpers

import (
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/cherrypicking"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type CherryPickHelper struct {
	c *HelperCommon

	rebaseHelper *MergeAndRebaseHelper
}

// I'm using the analogy of copy+paste in the terminology here because it's intuitively what's going on,
// even if in truth we're running git cherry-pick

func NewCherryPickHelper(
	c *HelperCommon,
	rebaseHelper *MergeAndRebaseHelper,
) *CherryPickHelper {
	return &CherryPickHelper{
		c:            c,
		rebaseHelper: rebaseHelper,
	}
}

func (self *CherryPickHelper) getData() *cherrypicking.CherryPicking {
	return self.c.Modes().CherryPicking
}

// Fork: unreachable, replaced by the captured cherry-pick flow (controllers/fork_cherry_pick.go). Do not reconnect: it stashes implicitly.
// HandlePasteCommits begins a cherry-pick rebase with the commits the user has copied.
// Only to be called from the branch commits controller
func (self *CherryPickHelper) Paste() error {
	self.c.Confirm(types.ConfirmOpts{
		Title: self.c.Tr.CherryPick,
		Prompt: utils.ResolvePlaceholderString(
			self.c.Tr.SureCherryPick,
			map[string]string{
				"numCommits": strconv.Itoa(len(self.getData().CherryPickedCommits)),
			}),
		HandleConfirm: func() error {
			mustStash := IsWorkingTreeDirtyExceptSubmodules(self.c.Model().Files, self.c.Model().Submodules)
			cherryPickedCommits := self.getData().CherryPickedCommits
			return self.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{
				Message:              self.c.Tr.CherryPickingStatus,
				HideWorkingTreeState: true,
			}, func(gocui.Task) error {
				self.c.LogAction(self.c.Tr.Actions.CherryPick)

				if mustStash {
					if err := self.c.Git().Stash.Push(self.c.Tr.AutoStashForCherryPicking); err != nil {
						return err
					}
				}

				result := self.c.Git().Rebase.CherryPickCommits(cherryPickedCommits)
				err := self.rebaseHelper.CheckMergeOrRebaseWithRefreshOptions(result,
					types.RefreshOptions{BatchUIUpdates: true})
				if err != nil {
					return result
				}

				// If we're in the cherry-picking state at this point, it must
				// be because there were conflicts. Don't clear the copied
				// commits in this case, since we might want to abort and try
				// pasting them again.
				isInCherryPick, result := self.c.Git().Status.IsInCherryPick()
				if result != nil {
					return result
				}
				if !isInCherryPick {
					// DidPaste and the re-render touch mode state and contexts,
					// so run them on the UI thread.
					self.c.OnUIThread(func() error {
						self.getData().DidPaste = true
						self.rerender()
						return nil
					})

					if mustStash {
						if err := self.c.Git().Stash.Pop(0); err != nil {
							return err
						}
						self.c.RefreshFromWorker(types.RefreshOptions{
							Scope: []types.RefreshableView{types.STASH, types.FILES},
						})
					}
				}

				return nil
			})
		},
	})

	return nil
}

func (self *CherryPickHelper) CanPaste() bool {
	return self.getData().CanPaste()
}

func (self *CherryPickHelper) Reset() error {
	self.getData().ContextKey = ""
	self.getData().CherryPickedCommits = nil

	self.rerender()
	return nil
}

func (self *CherryPickHelper) rerender() {
	for _, context := range []types.Context{
		self.c.Contexts().LocalCommits,
		self.c.Contexts().ReflogCommits,
		self.c.Contexts().SubCommits,
	} {
		self.c.PostRefreshUpdate(context)
	}
}
