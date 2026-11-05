package controllers

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestRewordFailureMessage(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	err := errors.New("exit status 1")

	tests := []struct {
		name     string
		rebasing bool
		key      string
		expected string
	}{
		{
			name:     "rebasing with a key",
			rebasing: true,
			key:      "g o",
			expected: "Rewording failed: exit status 1\n\n" +
				"The rebase is still stopped. Press g o to abort or continue it before you try again.\n\n" +
				"Your message is saved. Press Up in the message box to get it back.",
		},
		{
			name:     "rebasing without a key",
			rebasing: true,
			key:      "",
			expected: "Rewording failed: exit status 1\n\n" +
				"The rebase is still stopped. Abort or continue it before you try again.\n\n" +
				"Your message is saved. Press Up in the message box to get it back.",
		},
		{
			name:     "not rebasing",
			rebasing: false,
			key:      "g o",
			expected: "Rewording failed: exit status 1\n\n" +
				"Your message is saved. Press Up in the message box to get it back.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, rewordFailureMessage(tr, err, test.rebasing, test.key))
		})
	}
}
