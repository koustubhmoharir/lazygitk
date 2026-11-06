package helpers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
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
// branch of the same name is checked out like CheckoutSelectedBranch. Otherwise
// the remote branch's commit is captured by its full ref, checked against the
// same guards as named checkout, and then created and checked out as one
// tracking-branch checkout. A refusal creates no branch.
func (self *RefsHelper) CheckoutSelectedRemoteBranch(remote *models.RemoteBranch) error {
	localName := remote.Name
	if lo.ContainsBy(self.c.Model().Branches, func(branch *models.Branch) bool {
		return branch.Name == localName
	}) {
		return self.checkoutLocalBranchNamed(localName)
	}

	remoteRef := "refs/remotes/" + remote.FullName()
	repository := captureActionRepository(self.c)
	return self.c.WithWaitingStatus(self.c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		oid, err := repository.git.Branch.ResolveBranchActionCommit(remoteRef)
		if err != nil {
			return err
		}
		plan, err := repository.git.Branch.PrepareRemoteBranchCheckout(localName, remoteRef, oid)
		if err != nil {
			return self.remoteCheckoutError(err)
		}
		self.c.OnUIThread(func() error {
			return self.applyRemoteBranchCheckout(plan, repository)
		})
		return nil
	})
}

// applyRemoteBranchCheckout runs the one guarded create-and-checkout. Git
// revalidates the captured remote branch just before it runs. Input is buffered
// until the refresh shows the result, as in named checkout.
func (self *RefsHelper) applyRemoteBranchCheckout(plan git_commands.RemoteBranchCheckoutPlan, repository *capturedActionRepository) error {
	if err := repository.checkRepository(); err != nil {
		return err
	}
	return self.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: self.c.Tr.NamedCheckoutApplyingStatus}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		err := repository.git.Branch.ApplyRemoteBranchCheckout(plan)
		opts := namedCheckoutFailureRefreshOptions()
		if err == nil {
			opts = namedCheckoutRefreshOptions()
			opts.Then = func() error {
				if self.c.Context().Current() != self.c.Contexts().Branches {
					self.c.Context().Push(self.c.Contexts().Branches, types.OnFocusOpts{})
				}
				return nil
			}
		}
		self.c.RefreshFromWorker(opts)
		return self.remoteCheckoutError(err)
	})
}

// remoteCheckoutError translates the remote-moved refusal into the fork's wording.
// Other errors are Git's own and are shown unchanged.
func (self *RefsHelper) remoteCheckoutError(err error) error {
	if errors.Is(err, git_commands.ErrRemoteBranchChanged) {
		return errors.New(self.c.Tr.CheckoutRemoteBranchChanged)
	}
	return err
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

// CheckoutPreviousFlat is s p. The previous checkout is read from HEAD's reflog,
// not from @{-1}, which is ambiguous when a branch and a tag share a name. A
// previous local branch gets the safe named checkout; a previous commit is
// checked out detached. Anything else is refused. Upstream's CheckoutRef is
// never used here, because it offers an autostash.
func (self *RefsHelper) CheckoutPreviousFlat() error {
	repository := captureActionRepository(self.c)
	return self.c.WithWaitingStatus(self.c.Tr.NamedCheckoutWorkingStatus, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		target, err := repository.git.Branch.ClassifyPreviousCheckout()
		if err != nil {
			return err
		}
		switch target.Kind {
		case git_commands.PreviousCheckoutBranch:
			oid, err := repository.git.Branch.ResolveBranchActionCommit(target.FullRef)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(target.FullRef, "refs/heads/")
			self.c.OnUIThread(func() error {
				return self.namedCheckoutFlowAt(oid, repository).prepare(name)
			})
		case git_commands.PreviousCheckoutCommit:
			oid := target.OID
			self.c.OnUIThread(func() error {
				return self.checkoutCapturedDetached(oid, repository)
			})
		case git_commands.PreviousCheckoutGone:
			return fmt.Errorf(self.c.Tr.PreviousCheckoutUnavailable, target.Source)
		default:
			return errors.New(self.c.Tr.NoPreviousBranch)
		}
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
// checkout as the detached item of the upstream menus. envVars are set on the
// Git command only (undo and redo use them for the reflog action).
func (self *RefsHelper) checkoutCapturedDetached(oid string, repository *capturedActionRepository, envVars ...string) error {
	if err := repository.checkRepository(); err != nil {
		return err
	}
	return self.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: self.c.Tr.CheckingOutStatus}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		err := repository.git.Branch.CheckoutCapturedCommit(oid, envVars...)
		self.c.RefreshFromWorker(namedCheckoutFailureRefreshOptions())
		return err
	})
}
