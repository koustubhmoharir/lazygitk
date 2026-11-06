package controllers

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestIgnoredRebaseErrorsMapToForkStrings(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	assert.NoError(t, mapIgnoredRebaseError(tr, nil))

	afterCommit := mapIgnoredRebaseError(tr, git_commands.ErrRebaseRiskAfterCommit)
	assert.Equal(t, tr.RebaseRiskAfterCommit, afterCommit.Error())
	wrapped := mapIgnoredRebaseError(tr, errors.Join(errors.New("context"), git_commands.ErrRebaseRiskAfterCommit))
	assert.Equal(t, tr.RebaseRiskAfterCommit, wrapped.Error())

	approval := mapIgnoredRebaseError(tr, git_commands.ErrRebaseIgnoredApprovalRequired)
	assert.Equal(t, tr.RebaseIgnoredApprovalRequired, approval.Error())

	other := errors.New("unrelated")
	assert.Same(t, other, mapIgnoredRebaseError(tr, other))
}
