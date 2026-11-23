package i18n

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/utils/promptmark"
	"github.com/stretchr/testify/assert"
)

// TestUpstreamPromptMarksApplied checks every entry against the upstream
// English text. A missing or repeated substring means upstream wording has
// drifted; the entry is then skipped at runtime, so this test is what reports
// it.
func TestUpstreamPromptMarksApplied(t *testing.T) {
	set := EnglishTranslationSet()

	// Group entries by field, so each field is checked against all its marks.
	entriesByField := map[*string][]upstreamPromptMark{}
	var fields []*string
	for _, entry := range upstreamPromptMarks {
		field := entry.field(set)
		if _, ok := entriesByField[field]; !ok {
			fields = append(fields, field)
		}
		entriesByField[field] = append(entriesByField[field], entry)
	}

	for _, field := range fields {
		entries := entriesByField[field]
		marked := *field

		// Removing the marks this table adds gives the upstream text.
		unmarked := marked
		for _, entry := range entries {
			span := entry.mark(entry.text)
			assert.Equal(t, 1, strings.Count(marked, span),
				"expected the marked span %q exactly once", span)
			unmarked = strings.ReplaceAll(unmarked, span, entry.text)
		}

		for _, entry := range entries {
			assert.Equal(t, 1, strings.Count(unmarked, entry.text),
				"upstream text no longer contains %q exactly once", entry.text)
		}
		assert.Equal(t, unmarked, promptmark.Strip(marked),
			"stripping the markers changed the visible text")
	}
}

// TestMarkUpstreamPromptEntriesSkipsUnmatched checks that an entry whose
// substring is missing or repeated leaves its field unchanged, that a matching
// entry is still applied, and that an empty field does not panic.
func TestMarkUpstreamPromptEntriesSkipsUnmatched(t *testing.T) {
	set := &TranslationSet{}
	set.SureForceCheckout = "one two one"
	set.ForcePushPrompt = "no match here"
	set.SurePasteCommitMessage = ""

	markUpstreamPromptEntries(set, []upstreamPromptMark{
		{func(s *TranslationSet) *string { return &s.SureForceCheckout }, "one", promptmark.Warn},
		{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "absent", promptmark.Warn},
		{func(s *TranslationSet) *string { return &s.SurePasteCommitMessage }, "anything", promptmark.Warn},
		{func(s *TranslationSet) *string { return &s.SureForceCheckout }, "two", promptmark.Warn},
	})

	assert.Equal(t, "one "+promptmark.Warn("two")+" one", set.SureForceCheckout)
	assert.Equal(t, "no match here", set.ForcePushPrompt)
	assert.Equal(t, "", set.SurePasteCommitMessage)
}
