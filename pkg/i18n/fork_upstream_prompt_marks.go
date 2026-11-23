package i18n

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/utils/promptmark"
)

// Fork-owned role marks for upstream English prompts.
//
// Upstream call sites are not changed. Instead, markUpstreamPrompts wraps the
// substituted values and the warnings of selected upstream prompts in role
// markers (see pkg/utils/promptmark) when the English set is built. The
// visible text does not change.
//
// Each entry names a field, a substring of its upstream English text, and the
// role to mark it with. The substring must occur exactly once in the field.
// If it does not (for example, after an upstream wording change), the entry is
// skipped and the field is left unmarked. fork_upstream_prompt_marks_test.go
// checks every entry, so drift is caught when upstream is merged.
//
// Placeholders such as "{{.name}}" and format verbs such as "%s" are valid
// substrings: promptmark markers survive substitution.

type upstreamPromptMark struct {
	field func(*TranslationSet) *string
	text  string
	mark  func(string) string
}

var upstreamPromptMarks = []upstreamPromptMark{
	{func(s *TranslationSet) *string { return &s.AlreadyCheckedOutByWorktree }, "{{.worktreeName}}", promptmark.Path},

	{func(s *TranslationSet) *string { return &s.SureForceCheckout }, "You will lose all local changes", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.NukeTreeConfirmation }, "which is not undoable", promptmark.Warn},
	{func(s *TranslationSet) *string { return &s.ResetHardConfirmation }, "which is not undoable", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.SoftResetPrompt }, "'%s'", promptmark.Hash},

	{func(s *TranslationSet) *string { return &s.DiscardFileChangesPromptResetPatch }, "This will reset the active custom patch", promptmark.Warn},
	{func(s *TranslationSet) *string { return &s.DiscardLinesFromCommitPromptWithReset }, "This will reset the active custom patch", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "{{.branches}}", promptmark.Branch},
	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "{{.cancelKey}}", promptmark.Key},
	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "{{.confirmKey}}", promptmark.Key},
	{func(s *TranslationSet) *string { return &s.ForcePushBranchesPrompt }, "force push", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "{{.cancelKey}}", promptmark.Key},
	{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "{{.confirmKey}}", promptmark.Key},
	{func(s *TranslationSet) *string { return &s.ForcePushPrompt }, "force push", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.DiscardNestedRepositoriesPrompt }, "Discarding this also discards the following", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.DropUpdateRefPrompt }, "This is irreversible", promptmark.Warn},
	{func(s *TranslationSet) *string { return &s.DropMergeCommitPrompt }, "it will also drop all the commits that were merged in by it", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.ConfirmRevertCommit }, "{{.selectedCommit}}", promptmark.Hash},

	{func(s *TranslationSet) *string { return &s.RemoveSubmodulePrompt }, "'%s'", promptmark.Path},
	{func(s *TranslationSet) *string { return &s.RemoveSubmodulePrompt }, "This is irreversible", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "{{.tagName}}", promptmark.Tag},
	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "{{.cancelKey}}", promptmark.Key},
	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "{{.confirmKey}}", promptmark.Key},
	{func(s *TranslationSet) *string { return &s.ForceTagPrompt }, "overwrite", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.SurePasteCommitMessage }, "Pasting will overwrite the current commit message", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.ForceRemoveWorktreePrompt }, "{{.worktreeName}}", promptmark.Path},

	{func(s *TranslationSet) *string { return &s.DirenvApprovalPrompt }, "{{.confirmKey}}", promptmark.Key},
	{func(s *TranslationSet) *string { return &s.DirenvApprovalPrompt }, "{{.cancelKey}}", promptmark.Key},

	{func(s *TranslationSet) *string { return &s.SureCherryPick }, "{{.numCommits}}", promptmark.Count},

	{func(s *TranslationSet) *string { return &s.DeleteRemoteBranchPrompt }, "{{.selectedBranchName}}", promptmark.RemoteBranch},
	{func(s *TranslationSet) *string { return &s.DeleteRemoteBranchPrompt }, "{{.upstream}}", promptmark.Remote},

	{func(s *TranslationSet) *string { return &s.DeleteLocalAndRemoteBranchPrompt }, "{{.localBranchName}}", promptmark.Branch},
	{func(s *TranslationSet) *string { return &s.DeleteLocalAndRemoteBranchPrompt }, "{{.remoteBranchName}}", promptmark.RemoteBranch},
	{func(s *TranslationSet) *string { return &s.DeleteLocalAndRemoteBranchPrompt }, "{{.remoteName}}", promptmark.Remote},

	{func(s *TranslationSet) *string { return &s.ForceDeleteBranchMessage }, "{{.selectedBranchName}}", promptmark.Branch},
	{func(s *TranslationSet) *string { return &s.ForceDeleteBranchMessage }, "is not fully merged", promptmark.Warn},
	{func(s *TranslationSet) *string { return &s.ForceDeleteBranchesMessage }, "are not fully merged", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.SubmoduleMergeConflictDescription }, "{{.path}}", promptmark.Path},

	{func(s *TranslationSet) *string { return &s.AmendCommitWithConflictsMenuPrompt }, "amend the last finished commit", promptmark.Warn},

	{func(s *TranslationSet) *string { return &s.MoveCommitsToNewBranchMenuPrompt }, "{{.baseBranchName}}", promptmark.Branch},

	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptCheckout }, "{{.branchName}}", promptmark.Branch},
	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptTrackingBranch }, "{{.name}}", promptmark.Branch},
	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptTrackingBranch }, "{{.ref}}", promptmark.RemoteBranch},
	{func(s *TranslationSet) *string { return &s.WorktreeLocationPromptNewBranch }, "{{.name}}", promptmark.Branch},
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
		*field = strings.Replace(*field, entry.text, entry.mark(entry.text), 1)
	}
}
