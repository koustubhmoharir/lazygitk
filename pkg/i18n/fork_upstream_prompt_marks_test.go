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

		// Undoing the replacements gives the upstream text.
		unmarked := marked
		for _, entry := range entries {
			replacement, ok := entry.replacement()
			assert.True(t, ok, "malformed entry for %q", entry.text)
			assert.Equal(t, 1, strings.Count(marked, replacement),
				"expected the marked span %q exactly once", replacement)
			unmarked = strings.Replace(unmarked, replacement, entry.text, 1)
		}

		for _, entry := range entries {
			assert.Equal(t, 1, strings.Count(unmarked, entry.text),
				"upstream text no longer contains %q exactly once", entry.text)
		}

		// The visible text is the upstream text, less the quotes that the
		// unquoting entries drop.
		visible := unmarked
		for _, entry := range entries {
			visible = strings.Replace(visible, entry.text, entry.visible(), 1)
		}
		assert.Equal(t, visible, promptmark.Strip(marked),
			"stripping the markers changed the visible text")
	}
}

// TestUpstreamPromptMarkReplacement checks how each kind of entry replaces its
// text, and that malformed settings are rejected rather than applied.
func TestUpstreamPromptMarkReplacement(t *testing.T) {
	hash := promptmark.Hash
	branch := promptmark.Branch
	tests := []struct {
		name        string
		entry       upstreamPromptMark
		replacement string
		visible     string
		ok          bool
	}{
		{
			name:        "whole text",
			entry:       upstreamPromptMark{text: "which is not undoable", mark: promptmark.Warn},
			replacement: promptmark.Warn("which is not undoable"),
			visible:     "which is not undoable",
			ok:          true,
		},
		{
			name:        "value inside parentheses keeps them",
			entry:       upstreamPromptMark{text: "(%s)", mark: hash, value: "%s"},
			replacement: "(" + hash("%s") + ")",
			visible:     "(%s)",
			ok:          true,
		},
		{
			name:        "value inside a longer text keeps the rest",
			entry:       upstreamPromptMark{text: "Mark %s as", mark: hash, value: "%s"},
			replacement: "Mark " + hash("%s") + " as",
			visible:     "Mark %s as",
			ok:          true,
		},
		{
			name:        "unquote drops the quotes",
			entry:       upstreamPromptMark{text: "'%s'", mark: branch, value: "%s", unquote: true},
			replacement: branch("%s"),
			visible:     "%s",
			ok:          true,
		},
		{
			name:        "unquote accepts double quotes",
			entry:       upstreamPromptMark{text: `"%s"`, mark: branch, value: "%s", unquote: true},
			replacement: branch("%s"),
			visible:     "%s",
			ok:          true,
		},
		{
			name:        "nil mark drops the quotes and marks nothing",
			entry:       upstreamPromptMark{text: "'{{.ref}}'", value: "{{.ref}}", unquote: true},
			replacement: "{{.ref}}",
			visible:     "{{.ref}}",
			ok:          true,
		},
		{
			name:  "unquote needs the value between quotes",
			entry: upstreamPromptMark{text: "to '%s' now", mark: branch, value: "%s", unquote: true},
			ok:    false,
		},
		{
			name:  "value must occur once in text",
			entry: upstreamPromptMark{text: "%s and %s", mark: hash, value: "%s"},
			ok:    false,
		},
		{
			name:  "a whole-text entry needs a mark",
			entry: upstreamPromptMark{text: "'%s'"},
			ok:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			replacement, ok := test.entry.replacement()
			assert.Equal(t, test.ok, ok)
			if !test.ok {
				return
			}
			assert.Equal(t, test.replacement, replacement)
			assert.Equal(t, test.visible, test.entry.visible())
		})
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
		{func(s *TranslationSet) *string { return &s.SureForceCheckout }, "one", promptmark.Warn, "", false},
		{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "absent", promptmark.Warn, "", false},
		{func(s *TranslationSet) *string { return &s.SurePasteCommitMessage }, "anything", promptmark.Warn, "", false},
		{func(s *TranslationSet) *string { return &s.SureForceCheckout }, "two", promptmark.Warn, "", false},
	})

	assert.Equal(t, "one "+promptmark.Warn("two")+" one", set.SureForceCheckout)
	assert.Equal(t, "no match here", set.ForcePushPrompt)
	assert.Equal(t, "", set.SurePasteCommitMessage)
}
