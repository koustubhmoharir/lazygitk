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
	bindings := []*types.Binding{
		{
			CommandID:         "clipboard.copy-commit-hash",
			Verb:              "copy",
			Handler:           self.withItem(self.copyCommitHashToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitHash,
		},
		{
			CommandID:         "clipboard.copy-abbreviated-commit-hash",
			Verb:              "copy",
			Handler:           self.withItem(self.copyAbbreviatedCommitHashToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyAbbreviatedCommitHash,
		},
		{
			CommandID:         "clipboard.copy-commit-subject",
			Verb:              "copy",
			Handler:           self.withItem(self.copyCommitSubjectToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitSubject,
		},
		{
			CommandID:         "clipboard.copy-commit-message",
			Verb:              "copy",
			Handler:           self.withItem(self.copyCommitMessageToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitMessage,
		},
		{
			CommandID: "clipboard.copy-commit-body",
			Verb:      "copy",
			// The body is read at press time, not in GetDisabledReason, so
			// menus do not run git for every render.
			Handler:           self.withItem(self.copyCommitBodyToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitMessageBody,
		},
		{
			CommandID:         "clipboard.copy-commit-url",
			Verb:              "copy",
			Handler:           self.withItem(self.copyCommitURLToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitURL,
		},
		{
			CommandID:         "clipboard.copy-commit-diff",
			Verb:              "copy",
			Handler:           self.withItem(self.copyCommitDiffToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitDiff,
		},
		{
			CommandID:         "clipboard.copy-commit-author",
			Verb:              "copy",
			Handler:           self.withItem(self.copyAuthorToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyCommitAuthor,
		},
		{
			CommandID: "clipboard.copy-commit-tags",
			Verb:      "copy",
			Handler:   self.withItem(self.copyCommitTagsToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected(func(commit *models.Commit) *types.DisabledReason {
				if len(commit.Tags) == 0 {
					return &types.DisabledReason{Text: self.c.Tr.CommitHasNoTags}
				}
				return nil
			})),
			Description: self.c.Tr.CopyCommitTags,
		},
	}

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
