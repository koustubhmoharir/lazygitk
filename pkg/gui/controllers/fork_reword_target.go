package controllers

import (
	"errors"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: a reword captures its target when it starts: the repository, the commit's
// ID, a copy of the commits and the selected index, whether the commit was HEAD, and
// HEAD's ref and ID. Both submit paths verify that capture before any write, and then
// act only on the captured values. Upstream's handleReword and
// switchFromCommitMessagePanelToEditor read the live commits and selection when the
// message was submitted, so a commit that moved while the box was open was rewritten
// instead. The integration point is LocalCommitsController.reword.
type rewordTarget struct {
	lc         *LocalCommitsController
	git        *commands.GitCommand
	generation int
	commits    []*models.Commit
	index      int
	commitOID  string
	wasHead    bool
	headRef    string
	headOID    string
}

func (self *LocalCommitsController) captureRewordTarget(commit *models.Commit, index int) (*rewordTarget, error) {
	headRef, headOID, err := self.c.Git().Branch.CurrentHead()
	if err != nil {
		return nil, err
	}
	commits := self.c.Model().Commits
	// The captured commits drive the rebase, so they must describe the HEAD
	// captured above. A model that is already stale is refused before the box opens.
	if headIdx := models.HeadCommitIdx(commits); headIdx < 0 || commits[headIdx].Hash() != headOID {
		self.c.Refresh(types.RefreshOptions{})
		return nil, errors.New(self.c.Tr.RewordListStale)
	}
	return &rewordTarget{
		lc:         self,
		git:        self.c.Git(),
		generation: self.c.State().GetRepoGeneration(),
		commits:    slices.Clone(commits),
		index:      index,
		commitOID:  commit.Hash(),
		wasHead:    models.IsHeadCommit(commits, index),
		headRef:    headRef,
		headOID:    headOID,
	}, nil
}

// verify reports an error unless the repository, HEAD's ref and ID, and (for a HEAD
// target) HEAD's ID are still the captured ones. It writes nothing.
func (target *rewordTarget) verify() error {
	tr := target.lc.c.Tr
	if target.lc.c.Git() != target.git || target.lc.c.State().GetRepoGeneration() != target.generation {
		return errors.New(tr.RewordRepositoryChanged)
	}
	ref, oid, err := target.git.Branch.CurrentHead()
	if err != nil {
		return err
	}
	if ref != target.headRef || oid != target.headOID {
		return errors.New(tr.RewordTargetChanged)
	}
	if target.wasHead && oid != target.commitOID {
		return errors.New(tr.RewordTargetChanged)
	}
	return nil
}

// refuse returns a failed verification. Nothing was written, so the submit paths have
// not reported success and the text stays a draft. The panes refresh to the actual state.
func (target *rewordTarget) refuse(err error) error {
	target.lc.c.Refresh(types.RefreshOptions{})
	return err
}

// submit is the panel's OnConfirm.
func (target *rewordTarget) submit(summary string, description string) error {
	// Fork: see TakeCommitMessageSubmitReporter. It is called only after a success.
	reportSuccess := target.lc.c.Helpers().Commits.TakeCommitMessageSubmitReporter()
	if err := target.verify(); err != nil {
		return target.refuse(err)
	}
	if target.wasHead {
		return target.lc.c.Helpers().GPG.WithGpgHandling(target.git.Commit.RewordLastCommit(summary, description),
			git_commands.CommitGpgSign,
			target.lc.c.Tr.RewordingStatus,
			// Fork: the reword has succeeded.
			func() error {
				reportSuccess()
				return nil
			}, nil)
	}

	return target.lc.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{
		Message:              target.lc.c.Tr.RewordingStatus,
		HideWorkingTreeState: true,
	}, func(gocui.Task) error {
		// The rebase is the write, so verify again on the worker just before it.
		if err := target.verify(); err != nil {
			target.lc.c.RefreshFromWorker(types.RefreshOptions{})
			return err
		}
		err := target.git.Rebase.RewordCommit(target.commits, target.index, summary, description)
		if err != nil {
			// Fork: the rebase may be left stopped, and the text is kept as a draft. Refresh
			// first, as on success, so that the model shows the stopped rebase before input
			// is released; see fork_reword_failure.go.
			target.lc.c.RefreshFromWorker(types.RefreshOptions{})
			return target.lc.rewordFailure(err)
		}
		// Fork: the reword has succeeded.
		reportSuccess()
		target.lc.c.RefreshFromWorker(types.RefreshOptions{})
		return nil
	})
}

// switchToEditor is the panel's OnSwitchToEditor. The text it is given is the file the
// editor opens for the message.
func (target *rewordTarget) switchToEditor(filepath string) error {
	// Fork: see TakeCommitMessageSubmitReporter. Called only once the editor is about
	// to get the text, so a refusal or a failed rebase start keeps the draft.
	reportSuccess := target.lc.c.Helpers().Commits.TakeCommitMessageSubmitReporter()
	if err := target.verify(); err != nil {
		return target.refuse(err)
	}
	if target.wasHead {
		reportSuccess()
		return target.lc.c.RunSubprocessAndRefresh(
			target.git.Commit.RewordLastCommitInEditorWithMessageFileCmdObj(filepath))
	}

	err := target.git.Rebase.BeginInteractiveRebaseForCommit(target.commits, target.index, false)
	if err != nil {
		return err
	}
	reportSuccess()

	// now the selected commit should be our head so we'll amend it with the new message
	err = target.lc.c.RunSubprocessAndRefresh(
		target.git.Commit.RewordLastCommitInEditorWithMessageFileCmdObj(filepath))
	if err != nil {
		return err
	}

	err = target.git.Rebase.ContinueRebase()
	if err != nil {
		return err
	}

	target.lc.c.Refresh(types.RefreshOptions{})
	return nil
}
