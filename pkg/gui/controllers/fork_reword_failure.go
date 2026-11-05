package controllers

import (
	"strings"

	"github.com/go-errors/errors"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: a reword of an older commit runs a rebase, and when it fails the rebase is
// left stopped, so the reword box will not open until it is aborted or continued.
// The submitted text is kept as a draft, so the error says both. The integration
// point is the rebase branch of LocalCommitsController.handleReword.
func (self *LocalCommitsController) rewordFailure(err error) error {
	rebasing := self.c.Git().Status.WorkingTreeState().Rebasing
	key := self.c.CommandKeyLabel("branch.merge-rebase-menu", "")
	return errors.New(rewordFailureMessage(self.c.Tr, err, rebasing, key))
}

// rewordFailureMessage builds the paragraphs of the reword failure popup.
func rewordFailureMessage(tr *i18n.TranslationSet, err error, rebasing bool, key string) string {
	paragraphs := []string{
		utils.ResolvePlaceholderString(tr.RewordFailed, map[string]string{
			"error": err.Error(),
		}),
	}
	if rebasing && key != "" {
		paragraphs = append(paragraphs, utils.ResolvePlaceholderString(tr.RewordFailedRebaseStopped, map[string]string{
			"key": key,
		}))
	} else if rebasing {
		paragraphs = append(paragraphs, tr.RewordFailedRebaseStoppedNoKey)
	}
	paragraphs = append(paragraphs, tr.RewordFailedMessageKept)
	return strings.Join(paragraphs, "\n\n")
}
