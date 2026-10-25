package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: commit-like panels share one flat `y` copy menu with consistent second
// keys (see the copy table in new-features/Roadmap.md, item 12). These are the
// commit actions; the nested copy-attribute menu keeps its binding for the
// palette but has no default route.
func (self *BasicCommitsController) commitCopyBindings() []*types.Binding {
	selectedCommit := func() (*models.Commit, error) {
		commit := self.getSelectedItem()
		if commit == nil {
			return nil, errors.New(self.c.Tr.NoItemSelected)
		}
		return commit, nil
	}
	bindings := commitCopyBindingList(self.c, self, selectedCommit,
		self.require(self.singleItemSelected()),
		self.require(self.singleItemSelected(func(commit *models.Commit) *types.DisabledReason {
			if len(commit.Tags) == 0 {
				return &types.DisabledReason{Text: self.c.Tr.CommitHasNoTags}
			}
			return nil
		})),
	)

	// Fork: keybindings.go already owns some commit-hash actions for these
	// views; a view must not offer the same command ID twice.
	owned := forkCommitCopyOwnedIDs(self.context.GetKey())
	result := make([]*types.Binding, 0, len(bindings))
	for _, binding := range bindings {
		if !owned[binding.CommandID] {
			result = append(result, binding)
		}
	}
	return result
}

// commitCopyBindingList builds the flat copy menu's commit actions. Commit
// panels and ref panels share it: getCommit resolves the commit when an action
// runs, and disabled and tagsDisabled decide when the menu offers the actions.
//
// copier supplies the copy methods, which read only copier.c. Ref panels may
// therefore pass a bare &BasicCommitsController{c: c}; a method that starts to
// use other fields must not be called that way.
func commitCopyBindingList(
	c *ControllerCommon,
	copier *BasicCommitsController,
	getCommit func() (*models.Commit, error),
	disabled func() *types.DisabledReason,
	tagsDisabled func() *types.DisabledReason,
) []*types.Binding {
	handle := func(action func(*models.Commit) error) func() error {
		return func() error {
			commit, err := getCommit()
			if err != nil {
				return err
			}
			return action(commit)
		}
	}
	// Ref panels cannot check tags cheaply, so the tags action checks them at
	// press time. Commit panels never reach this check with an empty tag list
	// because tagsDisabled already blocks it.
	copyTags := func(commit *models.Commit) error {
		if len(commit.Tags) == 0 {
			return errors.New(c.Tr.CommitHasNoTags)
		}
		return copier.copyCommitTagsToClipboard(commit)
	}

	return []*types.Binding{
		{
			CommandID:         "clipboard.copy-commit-hash",
			Verb:              "copy",
			Handler:           handle(copier.copyCommitHashToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyCommitHash,
		},
		{
			CommandID:         "clipboard.copy-abbreviated-commit-hash",
			Verb:              "copy",
			Handler:           handle(copier.copyAbbreviatedCommitHashToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyAbbreviatedCommitHash,
		},
		{
			CommandID:         "clipboard.copy-commit-subject",
			Verb:              "copy",
			Handler:           handle(copier.copyCommitSubjectToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyCommitSubject,
		},
		{
			CommandID:         "clipboard.copy-commit-message",
			Verb:              "copy",
			Handler:           handle(copier.copyCommitMessageToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyCommitMessage,
		},
		{
			CommandID: "clipboard.copy-commit-body",
			Verb:      "copy",
			// The body is read at press time, not in GetDisabledReason, so
			// menus do not run git for every render.
			Handler:           handle(copier.copyCommitBodyToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyCommitMessageBody,
		},
		{
			CommandID:         "clipboard.copy-commit-url",
			Verb:              "copy",
			Handler:           handle(copier.copyCommitURLToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyCommitURL,
		},
		{
			CommandID:         "clipboard.copy-commit-diff",
			Verb:              "copy",
			Handler:           handle(copier.copyCommitDiffToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyCommitDiff,
		},
		{
			CommandID:         "clipboard.copy-commit-author",
			Verb:              "copy",
			Handler:           handle(copier.copyAuthorToClipboard),
			GetDisabledReason: disabled,
			Description:       c.Tr.CopyCommitAuthor,
		},
		{
			CommandID:         "clipboard.copy-commit-tags",
			Verb:              "copy",
			Handler:           handle(copyTags),
			GetDisabledReason: tagsDisabled,
			Description:       c.Tr.CopyCommitTags,
		},
	}
}

// addCommitCopyBindings appends the flat copy actions to the upstream list.
func (self *BasicCommitsController) addCommitCopyBindings(bindings []*types.Binding) []*types.Binding {
	return append(bindings, self.commitCopyBindings()...)
}

// forkCommitCopyOwnedIDs lists the copy command IDs that keybindings.go binds
// in each commit view. The copy-hash handlers there are view-specific: the
// commits and sub-commits handlers truncate, while the reflog handler copies
// the full hash.
func forkCommitCopyOwnedIDs(key types.ContextKey) map[string]bool {
	switch key {
	case context.LOCAL_COMMITS_CONTEXT_KEY, context.SUB_COMMITS_CONTEXT_KEY:
		return map[string]bool{"clipboard.copy-abbreviated-commit-hash": true}
	case context.REFLOG_COMMITS_CONTEXT_KEY:
		return map[string]bool{"clipboard.copy-commit-hash": true}
	}
	return nil
}

// copyAbbreviatedCommitHashToClipboard truncates exactly like the commits and
// sub-commits handler in keybindings.go (Git.TruncateCopiedCommitHashesTo).
func (self *BasicCommitsController) copyAbbreviatedCommitHashToClipboard(commit *models.Commit) error {
	hash := commit.Hash()
	if maxWidth := self.c.UserConfig().Git.TruncateCopiedCommitHashesTo; maxWidth > 0 {
		hash = hash[:min(len(hash), maxWidth)]
	}

	self.c.LogAction(self.c.Tr.Actions.CopyCommitHashToClipboard)
	if err := self.c.OS().CopyToClipboard(hash); err != nil {
		return err
	}

	self.c.Toast(fmt.Sprintf("'%s' %s", hash, self.c.Tr.CopiedToClipboard))
	return nil
}

func (self *BasicCommitsController) copyCommitBodyToClipboard(commit *models.Commit) error {
	body := self.getCommitMessageBody(commit.Hash())
	if body == "" {
		return errors.New(self.c.Tr.CommitHasNoMessageBody)
	}
	return self.copyCommitMessageBodyToClipboard(body)
}
