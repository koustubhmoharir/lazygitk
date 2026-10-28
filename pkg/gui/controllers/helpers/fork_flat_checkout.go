package helpers

import (
	"errors"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: the flat checkout keys under s (docs-master/fork/Tracker.md). These
// replace upstream's second-level checkout menus on the default routes. Each
// action captures its commit and repository when it starts, uses the safe named
// checkout or the captured detached checkout, and never offers a stash.

// CheckoutSelectedBranch is s c on a local branch: the safe named checkout of
// that branch, with no menu.
func (self *RefsHelper) CheckoutSelectedBranch(branch *models.Branch) error {
	if branch.DetachedHead {
		// The detached entry names no branch, so there is no name to check out.
		return errors.New(self.c.Tr.NamedCheckoutSelectCommit)
	}
	return self.checkoutLocalBranchNamed(branch.Name)
}

// CheckoutSelectedRemoteBranch is s c on a remote branch. An existing local
// branch of the same name is checked out like CheckoutSelectedBranch; otherwise
// a tracking branch is created first, then checked out the same way.
func (self *RefsHelper) CheckoutSelectedRemoteBranch(remote *models.RemoteBranch) error {
	localName := remote.Name
	if lo.ContainsBy(self.c.Model().Branches, func(branch *models.Branch) bool {
		return branch.Name == localName
	}) {
		return self.checkoutLocalBranchNamed(localName)
	}

	repository := captureActionRepository(self.c)
	if err := repository.checkRepository(); err != nil {
		return err
	}
	if err := repository.git.Branch.CreateWithUpstream(localName, remote.FullName()); err != nil {
		return err
	}
	// Refresh first so the new branch is in the model for the named checkout,
	// which re-checks the repository before it starts.
	self.c.Refresh(types.RefreshOptions{
		Scope: []types.RefreshableView{types.BRANCHES},
		Then: func() error {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			return self.checkoutLocalBranchNamed(localName)
		},
	})
	return nil
}

// NameCheckoutAtSelectedRef is s c on a tag: the named-checkout prompt for the
// tag's commit.
func (self *RefsHelper) NameCheckoutAtSelectedRef(fullRef string) error {
	return self.withCapturedRefCommit(fullRef, func(oid string, repository *capturedActionRepository) error {
		return self.namedCheckoutFlowAt(oid, repository).begin()
	})
}

// NameCheckoutAtCommit is s c on a commit in Commits, Reflog or Sub-commits: the
// named-checkout prompt for that commit.
func (self *RefsHelper) NameCheckoutAtCommit(oid string) error {
	return self.namedCheckoutFlowAt(oid, captureActionRepository(self.c)).begin()
}

// CheckoutSelectedRefDetached is s d on a branch, remote branch or tag: detached
// HEAD at the commit the ref names when the action starts.
func (self *RefsHelper) CheckoutSelectedRefDetached(fullRef string) error {
	return self.withCapturedRefCommit(fullRef, func(oid string, repository *capturedActionRepository) error {
		return self.checkoutCapturedDetached(oid, repository)
	})
}

// CheckoutCommitDetached is s d on a commit: detached HEAD at that commit.
func (self *RefsHelper) CheckoutCommitDetached(oid string) error {
	return self.checkoutCapturedDetached(oid, captureActionRepository(self.c))
}

// CheckoutPreviousFlat is s p. A previous local branch gets the safe named
// checkout; any other previous checkout (a detached commit, a tag or a remote
// branch) is checked out detached at its commit. Upstream's CheckoutRef is never
// used here, because it offers an autostash.
func (self *RefsHelper) CheckoutPreviousFlat() error {
	repository := captureActionRepository(self.c)
	return self.c.WithWaitingStatus(self.c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		previousRef, err := repository.git.Branch.PreviousRef()
		if err != nil {
			return err
		}
		if name, isBranch := strings.CutPrefix(previousRef, "refs/heads/"); isBranch {
			oid, err := repository.git.Branch.ResolveBranchActionCommit(previousRef)
			if err != nil {
				return err
			}
			self.c.OnUIThread(func() error {
				return self.namedCheckoutFlowAt(oid, repository).prepare(name)
			})
			return nil
		}
		oid, err := repository.git.Branch.PreviousCheckoutCommit()
		if err != nil {
			return err
		}
		self.c.OnUIThread(func() error {
			return self.checkoutCapturedDetached(oid, repository)
		})
		return nil
	})
}

func (self *RefsHelper) checkoutLocalBranchNamed(name string) error {
	return self.withCapturedRefCommit("refs/heads/"+name, func(oid string, repository *capturedActionRepository) error {
		return self.namedCheckoutFlowAt(oid, repository).prepare(name)
	})
}

// withCapturedRefCommit resolves fullRef to its commit on a worker, then runs
// use on the UI thread. Both steps re-check the captured repository.
func (self *RefsHelper) withCapturedRefCommit(fullRef string, use func(oid string, repository *capturedActionRepository) error) error {
	repository := captureActionRepository(self.c)
	return self.c.WithWaitingStatus(self.c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		oid, err := repository.git.Branch.ResolveBranchActionCommit(fullRef)
		if err != nil {
			return err
		}
		self.c.OnUIThread(func() error {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			return use(oid, repository)
		})
		return nil
	})
}

// namedCheckoutFlowAt binds the safe named checkout to one captured commit. The
// name prompt lists branches already at that commit first.
func (self *RefsHelper) namedCheckoutFlowAt(oid string, repository *capturedActionRepository) *namedCheckoutFlow {
	return &namedCheckoutFlow{
		c:                  self.c,
		git:                repository.git,
		oid:                oid,
		validateRepository: repository.checkRepository,
		nameSuggestions: func() func(string) []*types.Suggestion {
			return NewSuggestionsHelper(self.c).GetBranchNameSuggestionsAtCommitFunc(oid)
		},
	}
}

// checkoutCapturedDetached detaches HEAD at oid, using the same captured
// checkout as the detached item of the upstream menus.
func (self *RefsHelper) checkoutCapturedDetached(oid string, repository *capturedActionRepository) error {
	if err := repository.checkRepository(); err != nil {
		return err
	}
	return self.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: self.c.Tr.CheckingOutStatus}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		err := repository.git.Branch.CheckoutCapturedCommit(oid)
		self.c.RefreshFromWorker(namedCheckoutFailureRefreshOptions())
		return err
	})
}
