package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestForkCheckoutPreviousDisabledReason(t *testing.T) {
	tr := i18n.EnglishTranslationSet()

	t.Run("hidden while Status is focused", func(t *testing.T) {
		reason := forkCheckoutPreviousDisabledReason(tr, true, "main")
		if assert.NotNil(t, reason) {
			assert.True(t, reason.Inapplicable)
			assert.Equal(t, tr.NoApplicableCommandsInThisContext, reason.Text)
		}
	})

	t.Run("available with a previous checkout", func(t *testing.T) {
		assert.Nil(t, forkCheckoutPreviousDisabledReason(tr, false, "main"))
	})

	t.Run("reports a missing previous checkout elsewhere", func(t *testing.T) {
		reason := forkCheckoutPreviousDisabledReason(tr, false, "")
		if assert.NotNil(t, reason) {
			assert.False(t, reason.Inapplicable)
			assert.Equal(t, tr.NoPreviousBranch, reason.Text)
		}
	})
}
