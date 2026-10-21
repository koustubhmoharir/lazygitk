package controllers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context/traits"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// This controller is for all contexts that contain a list of commits.

var _ types.IController = &BasicCommitsController{}

type ContainsCommits interface {
	types.Context
	types.IListContext
	GetSelected() *models.Commit
	GetSelectedItems() ([]*models.Commit, int, int)
	GetCommits() []*models.Commit
	GetSelectedLineIdx() int
	GetSelectionRangeAndMode() (int, int, traits.RangeSelectMode)
	SetSelectionRangeAndMode(int, int, traits.RangeSelectMode)
}

type BasicCommitsController struct {
	baseController
	*ListControllerTrait[*models.Commit]
	c       *ControllerCommon
	context ContainsCommits
}

func NewBasicCommitsController(c *ControllerCommon, context ContainsCommits) *BasicCommitsController {
	return &BasicCommitsController{
		baseController: baseController{},
		c:              c,
		context:        context,
		ListControllerTrait: NewListControllerTrait(
			c,
			context,
			context.GetSelected,
			context.GetSelectedItems,
		),
	}
}

func (self *BasicCommitsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		// Fork: ref-only placement is separate from upstream creation/checkout.
		self.branchPlacementBinding(opts),
		{
			CommandID:         "commit.checkout-menu",
			Verb:              "checkout",
			Keys:              opts.GetKeys(opts.Config.Commits.CheckoutCommit),
			Handler:           self.withItem(self.checkout),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.Checkout,
			Tooltip:           self.c.Tr.CheckoutCommitTooltip,
			DisplayOnScreen:   true,
		},
		{
			CommandID:         "commit.copy-attribute-menu",
			Verb:              "copy",
			Keys:              opts.GetKeys(opts.Config.Commits.CopyCommitAttributeToClipboard),
			Handler:           self.withItem(self.copyCommitAttribute),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitAttributeToClipboard,
			Tooltip:           self.c.Tr.CopyCommitAttributeToClipboardTooltip,
			OpensMenu:         true,
		},
		{
			CommandID:         "commit.open-browser",
			Verb:              "open-in-browser",
			Keys:              opts.GetKeys(opts.Config.Commits.OpenInBrowser),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.OpenCommitInBrowser,
		},
		{
			CommandID:         "branch.create-and-checkout",
			Verb:              "create",
			Keys:              opts.GetKeys(opts.Config.Universal.New),
			Handler:           self.withItem(self.newBranch),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CreateBranchAndCheckout,
		},
		{
			// Putting this in BasicCommitsController even though we really only want it in the commits
			// panel. But I find it important that this ends up next to "New Branch", and I couldn't
			// find another way to achieve this. It's not such a big deal to have it in subcommits and
			// reflog too, I'd say.
			CommandID:         "branch.move-commits-to-new",
			Verb:              "move-commits",
			Keys:              opts.GetKeys(opts.Config.Branches.MoveCommitsToNewBranch),
			Handler:           self.c.Helpers().Refs.MoveCommitsToNewBranch,
			GetDisabledReason: self.c.Helpers().Refs.CanMoveCommitsToNewBranch,
			Description:       self.c.Tr.MoveCommitsToNewBranch,
			Tooltip:           self.c.Tr.MoveCommitsToNewBranchTooltip,
		},
		{
			CommandID:   "worktree.create-menu",
			Verb:        "create-worktree",
			Keys:        opts.GetKeys(opts.Config.Universal.NewWorktree),
			Handler:     self.withItem(self.c.Helpers().Worktree.NewWorktreeMenuForCommit),
			Description: self.c.Tr.NewWorktree,
			OpensMenu:   true,
		},
		{
			CommandID:         "branch.reset-menu",
			Verb:              "reset",
			Keys:              opts.GetKeys(opts.Config.Commits.ViewResetOptions),
			Handler:           self.withItem(self.createResetMenu),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.ViewResetOptions,
			Tooltip:           self.c.Tr.ResetTooltip,
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			CommandID:         "commit.copy-for-cherry-pick",
			Verb:              "copy-for-cherry-pick",
			Keys:              opts.GetKeys(opts.Config.Commits.CherryPickCopy),
			Handler:           self.withItem(self.copyRange),
			GetDisabledReason: self.require(self.itemRangeSelected(self.canCopyCommits)),
			Description:       self.c.Tr.CherryPickCopy,
			// Fork: unbound paste is discoverable in the palette, not an empty key hint.
			Tooltip:         cherryPickCopyTooltip(self.c.Tr, opts.Config),
			DisplayOnScreen: true,
		},
		{
			CommandID:   "commit.clear-cherry-pick-selection",
			Verb:        "clear-cherry-pick-selection",
			Keys:        opts.GetKeys(opts.Config.Commits.ResetCherryPick),
			Handler:     self.c.Helpers().CherryPick.Reset,
			Description: self.c.Tr.ResetCherryPick,
		},
		{
			CommandID:         "diff.commit.external",
			Verb:              "external-diff-context",
			Keys:              opts.GetKeys(opts.Config.Universal.OpenDiffTool),
			Handler:           self.withItem(self.openDiffTool),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.HistoricalDiffStickyTool,
		},
		{
			CommandID:         "commit.select-current-branch-history",
			Verb:              "select-branch-commits",
			Keys:              opts.GetKeys(opts.Config.Commits.SelectCommitsOfCurrentBranch),
			Handler:           self.selectCommitsOfCurrentBranch,
			GetDisabledReason: self.require(self.canSelectCommitsOfCurrentBranch),
			Description:       self.c.Tr.SelectCommitsOfCurrentBranch,
		},
	}

	// Fork direct-diff actions are additive; the existing sticky comparison remains available.
	return append(self.historicalDiffBindings(opts), bindings...)
}

func (self *BasicCommitsController) getCommitMessageBody(hash string) string {
	commitMessageBody, err := self.c.Git().Commit.GetCommitMessage(hash)
	if err != nil {
		return ""
	}
	_, body := self.c.Helpers().Commits.SplitCommitMessageAndDescription(commitMessageBody)
	return body
}

func (self *BasicCommitsController) copyCommitAttribute(commit *models.Commit) error {
	commitMessageBody := self.getCommitMessageBody(commit.Hash())
	var commitMessageBodyDisabled *types.DisabledReason
	if commitMessageBody == "" {
		commitMessageBodyDisabled = &types.DisabledReason{
			Text: self.c.Tr.CommitHasNoMessageBody,
		}
	}

	commitTagsItem := &types.MenuItem{
		CommandID: "clipboard.copy-commit-tags",
		Verb:      "copy",
		Label:     self.c.Tr.CommitTags,
		OnPress: func() error {
			return self.copyCommitTagsToClipboard(commit)
		},
		Keys: menuKey('t'),
	}

	if len(commit.Tags) == 0 {
		commitTagsItem.DisabledReason = &types.DisabledReason{Text: self.c.Tr.CommitHasNoTags}
	}

	items := []*types.MenuItem{
		{
			CommandID: "clipboard.copy-commit-hash",
			Verb:      "copy",
			Label:     self.c.Tr.CommitHash,
			OnPress: func() error {
				return self.copyCommitHashToClipboard(commit)
			},
		},
		{
			CommandID: "clipboard.copy-commit-subject",
			Verb:      "copy",
			Label:     self.c.Tr.CommitSubject,
			OnPress: func() error {
				return self.copyCommitSubjectToClipboard(commit)
			},
			Keys: menuKey('s'),
		},
		{
			CommandID: "clipboard.copy-commit-message",
			Verb:      "copy",
			Label:     self.c.Tr.CommitMessage,
			OnPress: func() error {
				return self.copyCommitMessageToClipboard(commit)
			},
			Keys: menuKey('m'),
		},
		{
			CommandID:      "clipboard.copy-commit-body",
			Verb:           "copy",
			Label:          self.c.Tr.CommitMessageBody,
			DisabledReason: commitMessageBodyDisabled,
			OnPress: func() error {
				return self.copyCommitMessageBodyToClipboard(commitMessageBody)
			},
			Keys: menuKey('b'),
		},
		{
			CommandID: "clipboard.copy-commit-url",
			Verb:      "copy",
			Label:     self.c.Tr.CommitURL,
			OnPress: func() error {
				return self.copyCommitURLToClipboard(commit)
			},
			Keys: menuKey('u'),
		},
		{
			CommandID: "clipboard.copy-commit-diff",
			Verb:      "copy",
			Label:     self.c.Tr.CommitDiff,
			OnPress: func() error {
				return self.copyCommitDiffToClipboard(commit)
			},
			Keys: menuKey('d'),
		},
		{
			CommandID: "clipboard.copy-commit-author",
			Verb:      "copy",
			Label:     self.c.Tr.CommitAuthor,
			OnPress: func() error {
				return self.copyAuthorToClipboard(commit)
			},
			Keys: menuKey('a'),
		},
		commitTagsItem,
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.Actions.CopyCommitAttributeToClipboard,
		Items: items,
	})
}

func (self *BasicCommitsController) copyCommitHashToClipboard(commit *models.Commit) error {
	self.c.LogAction(self.c.Tr.Actions.CopyCommitHashToClipboard)
	if err := self.c.OS().CopyToClipboard(commit.Hash()); err != nil {
		return err
	}

	self.c.Toast(fmt.Sprintf("'%s' %s", commit.Hash(), self.c.Tr.CopiedToClipboard))
	return nil
}

func (self *BasicCommitsController) copyCommitURLToClipboard(commit *models.Commit) error {
	url, err := self.c.Helpers().Host.GetCommitURL(commit.Hash())
	if err != nil {
		return err
	}

	self.c.LogAction(self.c.Tr.Actions.CopyCommitURLToClipboard)
	if err := self.c.OS().CopyToClipboard(url); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.CommitURLCopiedToClipboard)
	return nil
}

func (self *BasicCommitsController) copyCommitDiffToClipboard(commit *models.Commit) error {
	diff, err := self.c.Git().Commit.GetCommitDiff(commit.Hash())
	if err != nil {
		return err
	}

	self.c.LogAction(self.c.Tr.Actions.CopyCommitDiffToClipboard)
	if err := self.c.OS().CopyToClipboard(diff); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.CommitDiffCopiedToClipboard)
	return nil
}

func (self *BasicCommitsController) copyAuthorToClipboard(commit *models.Commit) error {
	author, err := self.c.Git().Commit.GetCommitAuthor(commit.Hash())
	if err != nil {
		return err
	}

	formattedAuthor := fmt.Sprintf("%s <%s>", author.Name, author.Email)

	self.c.LogAction(self.c.Tr.Actions.CopyCommitAuthorToClipboard)
	if err := self.c.OS().CopyToClipboard(formattedAuthor); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.CommitAuthorCopiedToClipboard)
	return nil
}

func (self *BasicCommitsController) copyCommitMessageToClipboard(commit *models.Commit) error {
	message, err := self.c.Git().Commit.GetCommitMessage(commit.Hash())
	if err != nil {
		return err
	}

	self.c.LogAction(self.c.Tr.Actions.CopyCommitMessageToClipboard)
	if err := self.c.OS().CopyToClipboard(message); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.CommitMessageCopiedToClipboard)
	return nil
}

func (self *BasicCommitsController) copyCommitMessageBodyToClipboard(commitMessageBody string) error {
	self.c.LogAction(self.c.Tr.Actions.CopyCommitMessageBodyToClipboard)
	if err := self.c.OS().CopyToClipboard(commitMessageBody); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.CommitMessageBodyCopiedToClipboard)
	return nil
}

func (self *BasicCommitsController) copyCommitSubjectToClipboard(commit *models.Commit) error {
	message, err := self.c.Git().Commit.GetCommitSubject(commit.Hash())
	if err != nil {
		return err
	}

	self.c.LogAction(self.c.Tr.Actions.CopyCommitSubjectToClipboard)
	if err := self.c.OS().CopyToClipboard(message); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.CommitSubjectCopiedToClipboard)
	return nil
}

func (self *BasicCommitsController) copyCommitTagsToClipboard(commit *models.Commit) error {
	message := strings.Join(commit.Tags, "\n")

	self.c.LogAction(self.c.Tr.Actions.CopyCommitTagsToClipboard)
	if err := self.c.OS().CopyToClipboard(message); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.CommitTagsCopiedToClipboard)
	return nil
}

func (self *BasicCommitsController) openInBrowser(commit *models.Commit) error {
	url, err := self.c.Helpers().Host.GetCommitURL(commit.Hash())
	if err != nil {
		return err
	}

	self.c.LogAction(self.c.Tr.Actions.OpenCommitInBrowser)
	if err := self.c.OS().OpenLink(url); err != nil {
		return err
	}

	return nil
}

func (self *BasicCommitsController) newBranch(commit *models.Commit) error {
	return self.c.Helpers().Refs.NewBranch(commit.RefName(), commit.Description(), "")
}

func (self *BasicCommitsController) createResetMenu(commit *models.Commit) error {
	return self.c.Helpers().Refs.CreateGitResetMenu(commit.Hash(), commit.Hash())
}

func (self *BasicCommitsController) checkout(commit *models.Commit) error {
	return self.c.Helpers().Refs.CreateCheckoutMenu(commit)
}

func (self *BasicCommitsController) copyRange(*models.Commit) error {
	return self.c.Helpers().CherryPick.CopyRange(self.context.GetCommits(), self.context)
}

func (self *BasicCommitsController) canCopyCommits(selectedCommits []*models.Commit, startIdx int, endIdx int) *types.DisabledReason {
	for _, commit := range selectedCommits {
		if commit.Hash() == "" {
			return &types.DisabledReason{Text: self.c.Tr.CannotCherryPickNonCommit, ShowErrorInPanel: true}
		}
	}

	return nil
}

func (self *BasicCommitsController) openDiffTool(commit *models.Commit) error {
	to := commit.RefName()
	from, reverse := self.c.Modes().Diffing.GetFromAndReverseArgsForDiff(commit.ParentRefName())
	_, err := self.c.RunSubprocess(self.c.Git().Diff.OpenDiffToolCmdObj(
		git_commands.DiffToolCmdOptions{
			Filepath:    ".",
			FromCommit:  from,
			ToCommit:    to,
			Reverse:     reverse,
			IsDirectory: true,
			Staged:      false,
		}))
	return err
}

func (self *BasicCommitsController) canSelectCommitsOfCurrentBranch() *types.DisabledReason {
	if index := self.findFirstCommitAfterCurrentBranch(); index <= 0 {
		return &types.DisabledReason{Text: self.c.Tr.NoCommitsThisBranch}
	}

	return nil
}

func (self *BasicCommitsController) findFirstCommitAfterCurrentBranch() int {
	_, index, ok := lo.FindIndexOf(self.context.GetCommits(), func(c *models.Commit) bool {
		return c.IsMerge() || c.Status == models.StatusMerged
	})

	if !ok {
		return 0
	}

	return index
}

func (self *BasicCommitsController) selectCommitsOfCurrentBranch() error {
	index := self.findFirstCommitAfterCurrentBranch()
	if index <= 0 {
		return nil
	}

	_, _, mode := self.context.GetSelectionRangeAndMode()
	if mode != traits.RangeSelectModeSticky {
		// If we are in sticky range mode already, keep that; otherwise, open a non-sticky range
		mode = traits.RangeSelectModeNonSticky
	}
	// Create the range from bottom to top, so that when you cancel the range,
	// the head commit is selected
	self.context.SetSelectionRangeAndMode(0, index-1, mode)
	self.context.HandleFocus(types.OnFocusOpts{})
	return nil
}
