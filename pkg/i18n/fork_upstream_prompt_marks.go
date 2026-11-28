package i18n

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/utils/promptmark"
)

// Fork-owned role marks for upstream English texts.
//
// Upstream call sites are not changed. Instead, markUpstreamPrompts wraps the
// substituted values and the warnings of selected upstream texts in role
// markers (see pkg/utils/promptmark) when the English set is built. Most
// entries leave the visible text as it is. An entry with unquote drops the
// quotes around its value, as the menu labels in the fork's style do; the
// visible text then differs from upstream's by those quotes only.
//
// Each entry names a field, a substring of its upstream English text, and the
// role to mark it with. The substring must occur exactly once in the field.
// If it does not (for example, after an upstream wording change), the entry is
// skipped and the field is left unmarked. fork_upstream_prompt_marks_test.go
// checks every entry, so drift is caught when upstream is merged.
//
// Placeholders such as "{{.name}}" and format verbs such as "%s" are valid
// substrings: promptmark markers survive substitution.
//
// By default the whole text is marked. The value field, when set, names the
// part of text to mark instead, so that the surrounding words keep their
// place, as in "Mark %s as" or "(%s)". The value must occur exactly once in
// text. With unquote, text is the value between single or double quotes, and
// the quotes are dropped. A nil mark drops the quotes without marking the
// value, for values whose role depends on the value: the call site marks them
// with promptmark.RefAs.

type upstreamPromptMark struct {
	field func(*TranslationSet) *string
	text  string
	mark  func(string) string
	// value is the part of text to mark, when it is not all of text.
	value string
	// unquote drops the quotes around value, so text is quote+value+quote.
	unquote bool
}

// replacement returns the text that replaces entry.text in the field, and
// whether the entry is well formed for its settings.
func (entry upstreamPromptMark) replacement() (string, bool) {
	marked := func(s string) string {
		if entry.mark == nil {
			return s
		}
		return entry.mark(s)
	}
	switch {
	case entry.value == "":
		if entry.mark == nil {
			return "", false
		}
		return entry.mark(entry.text), true
	case entry.unquote:
		if entry.text != "'"+entry.value+"'" && entry.text != `"`+entry.value+`"` {
			return "", false
		}
		return marked(entry.value), true
	default:
		if strings.Count(entry.text, entry.value) != 1 {
			return "", false
		}
		return strings.Replace(entry.text, entry.value, marked(entry.value), 1), true
	}
}

// visible returns the text that entry.text displays as once it is replaced.
func (entry upstreamPromptMark) visible() string {
	switch {
	case entry.value == "":
		return entry.text
	case entry.unquote:
		return entry.value
	default:
		return entry.text
	}
}

var upstreamPromptMarks = []upstreamPromptMark{
	{func(s *TranslationSet) *string { return &s.AlreadyCheckedOutByWorktree }, "{{.worktreeName}}", promptmark.Path, "", false},

	{func(s *TranslationSet) *string { return &s.SureForceCheckout }, "You will lose all local changes", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.NukeTreeConfirmation }, "which is not undoable", promptmark.Warn, "", false},
	{func(s *TranslationSet) *string { return &s.ResetHardConfirmation }, "which is not undoable", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.SoftResetPrompt }, "'%s'", promptmark.Hash, "", false},

	{func(s *TranslationSet) *string { return &s.DiscardFileChangesPromptResetPatch }, "This will reset the active custom patch", promptmark.Warn, "", false},
	{func(s *TranslationSet) *string { return &s.DiscardLinesFromCommitPromptWithReset }, "This will reset the active custom patch", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "{{.branches}}", promptmark.Branch, "", false},
	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "{{.cancelKey}}", promptmark.Key, "", false},
	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "{{.confirmKey}}", promptmark.Key, "", false},
	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "force push", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "{{.cancelKey}}", promptmark.Key, "", false},
	{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "{{.confirmKey}}", promptmark.Key, "", false},
	{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "force push", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.DiscardNestedRepositoriesPrompt }, "Discarding this also discards the following", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.DropUpdateRefPrompt }, "This is irreversible", promptmark.Warn, "", false},
	{func(s *TranslationSet) *string { return &s.DropMergeCommitPrompt }, "it will also drop all the commits that were merged in by it", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.ConfirmRevertCommit }, "{{.selectedCommit}}", promptmark.Hash, "", false},

	{func(s *TranslationSet) *string { return &s.RemoveSubmodulePrompt }, "'%s'", promptmark.Path, "", false},
	{func(s *TranslationSet) *string { return &s.RemoveSubmodulePrompt }, "This is irreversible", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "{{.tagName}}", promptmark.Tag, "", false},
	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "{{.cancelKey}}", promptmark.Key, "", false},
	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "{{.confirmKey}}", promptmark.Key, "", false},
	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "overwrite", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.SurePasteCommitMessage }, "Pasting will overwrite the current commit message", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.ForceRemoveWorktreePrompt }, "{{.worktreeName}}", promptmark.Path, "", false},

	{func(s *TranslationSet) *string { return &s.DirenvApprovalPrompt }, "{{.confirmKey}}", promptmark.Key, "", false},
	{func(s *TranslationSet) *string { return &s.DirenvApprovalPrompt }, "{{.cancelKey}}", promptmark.Key, "", false},

	{func(s *TranslationSet) *string { return &s.SureCherryPick }, "{{.numCommits}}", promptmark.Count, "", false},

	{func(s *TranslationSet) *string { return &s.DeleteRemoteBranchPrompt }, "{{.selectedBranchName}}", promptmark.RemoteBranch, "", false},
	{func(s *TranslationSet) *string { return &s.DeleteRemoteBranchPrompt }, "{{.upstream}}", promptmark.Remote, "", false},

	{func(s *TranslationSet) *string { return &s.DeleteLocalAndRemoteBranchPrompt }, "{{.localBranchName}}", promptmark.Branch, "", false},
	{func(s *TranslationSet) *string { return &s.DeleteLocalAndRemoteBranchPrompt }, "{{.remoteBranchName}}", promptmark.RemoteBranch, "", false},
	{func(s *TranslationSet) *string { return &s.DeleteLocalAndRemoteBranchPrompt }, "{{.remoteName}}", promptmark.Remote, "", false},

	{func(s *TranslationSet) *string { return &s.ForceDeleteBranchMessage }, "{{.selectedBranchName}}", promptmark.Branch, "", false},
	{func(s *TranslationSet) *string { return &s.ForceDeleteBranchMessage }, "is not fully merged", promptmark.Warn, "", false},
	{func(s *TranslationSet) *string { return &s.ForceDeleteBranchesMessage }, "are not fully merged", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.SubmoduleMergeConflictDescription }, "{{.path}}", promptmark.Path, "", false},

	{func(s *TranslationSet) *string { return &s.AmendCommitWithConflictsMenuPrompt }, "amend the last finished commit", promptmark.Warn, "", false},

	{func(s *TranslationSet) *string { return &s.MoveCommitsToNewBranchMenuPrompt }, "{{.baseBranchName}}", promptmark.Branch, "", false},

	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptCheckout }, "{{.branchName}}", promptmark.Branch, "", false},
	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptTrackingBranch }, "{{.name}}", promptmark.Branch, "", false},
	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptTrackingBranch }, "{{.ref}}", promptmark.RemoteBranch, "", false},
	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptNewBranch }, "{{.name}}", promptmark.Branch, "", false},

	// Menu labels: values are marked and their quotes dropped (see the header).
	{field: func(s *TranslationSet) *string { return &s.Bisect.Mark }, text: "current commit (%s)", mark: promptmark.Hash, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.Bisect.SkipCurrent }, text: "(%s)", mark: promptmark.Hash, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.Bisect.SkipSelected }, text: "(%s)", mark: promptmark.Hash, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.Bisect.MarkStart }, text: "Mark %s as", mark: promptmark.Hash, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.MoveCommitsToNewBranchFromBaseItem }, text: "(%s)", mark: promptmark.Branch, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.MoveCommitsToNewBranchStackedItem }, text: "(%s)", mark: promptmark.Branch, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.RemovePatchFromOriginalCommit }, text: "(%s)", mark: promptmark.Hash, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.MovePatchToSelectedCommit }, text: "(%s)", mark: promptmark.Hash, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.Actions.CheckoutCommitAsDetachedHead }, text: "commit %s as", mark: promptmark.Hash, value: "%s"},
	{field: func(s *TranslationSet) *string { return &s.Actions.CheckoutBranchAtCommit }, text: "'%s'", mark: promptmark.Branch, value: "%s", unquote: true},
	{field: func(s *TranslationSet) *string { return &s.PushOnlyCurrentBranch }, text: "'{{.branchName}}'", mark: promptmark.Branch, value: "{{.branchName}}", unquote: true},
	{field: func(s *TranslationSet) *string { return &s.PullOnlyCurrentBranch }, text: "'{{.branchName}}'", mark: promptmark.Branch, value: "{{.branchName}}", unquote: true},
	{field: func(s *TranslationSet) *string { return &s.WorktreeForRef }, text: "'{{.ref}}'", mark: promptmark.Branch, value: "{{.ref}}", unquote: true},
	{field: func(s *TranslationSet) *string { return &s.NewLocalBranchAndWorktreeFromRef }, text: "'{{.ref}}'", mark: promptmark.RemoteBranch, value: "{{.ref}}", unquote: true},
	{field: func(s *TranslationSet) *string { return &s.SimpleRebase }, text: "'{{.ref}}'", mark: nil, value: "{{.ref}}", unquote: true},
	{field: func(s *TranslationSet) *string { return &s.InteractiveRebase }, text: "'{{.ref}}'", mark: nil, value: "{{.ref}}", unquote: true},
}

// markUpstreamPrompts applies upstreamPromptMarks to set. An entry whose
// substring does not occur exactly once in its field is skipped, so a wording
// change upstream leaves the text unmarked rather than breaking it.
func markUpstreamPrompts(set *TranslationSet) {
	markUpstreamPromptEntries(set, upstreamPromptMarks)
}

func markUpstreamPromptEntries(set *TranslationSet, entries []upstreamPromptMark) {
	for _, entry := range entries {
		field := entry.field(set)
		if strings.Count(*field, entry.text) != 1 {
			continue
		}
		replacement, ok := entry.replacement()
		if !ok {
			continue
		}
		*field = strings.Replace(*field, entry.text, replacement, 1)
	}
}
