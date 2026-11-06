package helpers

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Fork: the `s n` prompt opens a repository by typed path (roadmap item 14).
// Switching goes through DispatchSwitchTo, as the recent-repositories menu
// does. The prompt's suggestions and Tab completion are computed here, and the
// prompt is reopened with the typed text when a switch is refused.

// repoPathSuggestionLimit caps the directory suggestions under the prompt.
const repoPathSuggestionLimit = 15

// repoPathSeparators end a directory segment in typed input. On Linux the
// second one is already the separator, so LastIndexAny finds either.
var repoPathSeparators = string(filepath.Separator) + "/"

// OpenRepoPathPrompt opens the "Repository path" prompt, prefilled with the
// parent directory of the current repository.
func (self *ReposHelper) OpenRepoPathPrompt() error {
	repoRoot := self.c.Git().RepoPaths.WorktreePath()
	self.openRepoPathPrompt(repoRoot, repoPathDefaultInput(repoRoot))
	return nil
}

func (self *ReposHelper) openRepoPathPrompt(repoRoot string, initialContent string) {
	self.repoPathSeq++ // Fork: a Tab completion from an earlier opening is dropped (see repoPathTab)
	self.c.Prompt(types.PromptOpts{
		Title:               self.c.Tr.RepoPathPrompt,
		InitialContent:      initialContent,
		FindSuggestionsFunc: repoPathSuggestionsFunc(repoRoot),
		HandleConfirm:       func(input string) error { return self.confirmRepoPath(repoRoot, input) },
		HandleTab:           func() (bool, error) { return self.completeRepoPathInput(repoRoot) },
	})
}

// reopenRepoPathPrompt shows errMsg (when set) and reopens the prompt with the
// text the user typed, so they can correct it.
func (self *ReposHelper) reopenRepoPathPrompt(repoRoot string, input string, errMsg string) error {
	if errMsg != "" {
		self.c.ErrorToast(errMsg)
	}
	self.openRepoPathPrompt(repoRoot, input)
	return nil
}

func (self *ReposHelper) confirmRepoPath(repoRoot string, input string) error {
	path := resolveRepoPath(input, repoRoot, userHomeDir())
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return self.reopenRepoPathPrompt(repoRoot, input,
			utils.ResolvePlaceholderString(self.c.Tr.ErrRepoPathNotDirectory, map[string]string{"path": path}))
	}

	// A refused switch leaves the prompt open with its text. The busy check
	// comes first: the refusal is then shown here, and the prompt keeps its text.
	if self.switchRefusedBecauseBusy() {
		return self.reopenRepoPathPrompt(repoRoot, input, "")
	}

	if err := self.DispatchSwitchTo(path, self.c.Tr.ErrRepositoryMovedOrDeleted, context.NO_CONTEXT); err != nil {
		// Fork: only a directory git cannot open keeps the prompt open with its
		// text. Other failures, such as a repo config that does not load, are
		// shown as errors (notAGitRepoError, fork_repo_switch.go).
		var notRepo notAGitRepoError
		if errors.As(err, &notRepo) {
			self.c.Log.WithError(err).Warn("repository path is not a Git repository")
			return self.reopenRepoPathPrompt(repoRoot, input,
				utils.ResolvePlaceholderString(self.c.Tr.ErrRepoPathNotGitRepo, map[string]string{"path": path}))
		}
		return err
	}
	return nil
}

// repoPathTab is one Tab press in the repository path prompt. Its completion
// is computed on a worker and applied on the UI thread only while the press is
// still the latest, the prompt is still open and its input is unchanged.
type repoPathTab struct {
	seq   uint64 // the value of ReposHelper.repoPathSeq when Tab was pressed
	input string // the prompt's input when Tab was pressed
}

// repoPathTabIsCurrent reports whether a completion for tab may still be
// applied: no later Tab or opening of the prompt has happened, the prompt is
// still open, and the input is what it was when Tab was pressed.
func repoPathTabIsCurrent(tab repoPathTab, latestSeq uint64, promptOpen bool, currentInput string) bool {
	return tab.seq == latestSeq && promptOpen && tab.input == currentInput
}

// completeRepoPathInput is the prompt's Tab hook. The directory reads run on a
// worker, so Tab does not hold the UI thread on a slow or large directory. Tab
// is always handled here: when the completion arrives and finds nothing to
// complete, focus moves to the suggestions, as a Tab that cannot complete
// always did.
func (self *ReposHelper) completeRepoPathInput(repoRoot string) (bool, error) {
	self.repoPathSeq++
	tab := repoPathTab{seq: self.repoPathSeq, input: self.c.GetPromptInput()}
	input := normalizeRepoPathInput(tab.input, repoRoot)
	home := userHomeDir()
	self.c.OnWorker(func(_ gocui.Task) error {
		completed, ok := completeRepoPathText(input, repoRoot, home)
		self.c.OnUIThread(func() error {
			self.applyRepoPathCompletion(tab, completed, ok)
			return nil
		})
		return nil
	})
	return true, nil
}

// applyRepoPathCompletion runs on the UI thread. It drops the completion unless
// repoPathTabIsCurrent holds for it.
func (self *ReposHelper) applyRepoPathCompletion(tab repoPathTab, completed string, ok bool) {
	// The prompt's Tab hook is set only while this prompt is open.
	promptOpen := self.c.Contexts().Prompt.State.HandleTab != nil
	if !repoPathTabIsCurrent(tab, self.repoPathSeq, promptOpen, self.c.GetPromptInput()) {
		return
	}

	if !ok {
		// Same as the prompt's own Tab fallback: focus the suggestions, if any.
		// The repo path prompt has no delete action, so the subtitle is empty.
		if len(self.c.Contexts().Suggestions.State.Suggestions) > 0 {
			self.c.Views().Suggestions.Subtitle = ""
			self.c.Context().Replace(self.c.Contexts().Suggestions)
		}
		return
	}

	textArea := self.c.Views().Prompt.TextArea
	textArea.Clear()
	textArea.TypeString(completed)
	self.c.Views().Prompt.RenderTextArea()
	self.c.Contexts().Suggestions.RefreshSuggestions()
}

// repoPathDefaultInput is the initial prompt text: the repository's parent
// directory with a trailing separator.
func repoPathDefaultInput(repoRoot string) string {
	parent := filepath.Dir(repoRoot)
	if strings.HasSuffix(parent, string(filepath.Separator)) {
		return parent
	}
	return parent + string(filepath.Separator)
}

// normalizeRepoPathInput treats empty input as the initial content. The
// suggestions for "" then match the prefilled prompt, so the list is correct
// before the user types, without changing shared prompt code.
func normalizeRepoPathInput(input string, repoRoot string) string {
	if input == "" {
		return repoPathDefaultInput(repoRoot)
	}
	return input
}

// splitRepoPathInput splits typed input at its last separator. dirText keeps
// the separator and prefix is the partly typed name.
func splitRepoPathInput(input string) (dirText string, prefix string) {
	i := strings.LastIndexAny(input, repoPathSeparators)
	return input[:i+1], input[i+1:]
}

// resolveRepoPath expands a leading ~ and resolves a relative path against the
// current repository's root.
func resolveRepoPath(input string, repoRoot string, home string) string {
	path := input
	if home != "" && (path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~"+string(filepath.Separator))) {
		path = filepath.Join(home, path[1:])
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	return filepath.Clean(path)
}

func userHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// matchRepoPathDirs returns the sorted names of the subdirectories of dir that
// start with prefix. Hidden directories appear only when the prefix starts with
// a dot. An unreadable dir gives no names.
func matchRepoPathDirs(dir string, prefix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	showHidden := strings.HasPrefix(prefix, ".")
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !showHidden {
			continue
		}
		if !repoPathNameHasPrefix(name, prefix) || !isRepoPathDir(dir, entry) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func repoPathNameHasPrefix(name string, prefix string) bool {
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix))
	}
	return strings.HasPrefix(name, prefix)
}

// isRepoPathDir reports whether entry is a directory, following a symlink.
func isRepoPathDir(dir string, entry fs.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&fs.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, entry.Name()))
	return err == nil && info.IsDir()
}

// repoPathSuggestionsFunc returns the suggestion function for the prompt. It
// reads the directory again for each input.
func repoPathSuggestionsFunc(repoRoot string) func(string) []*types.Suggestion {
	home := userHomeDir()
	return func(input string) []*types.Suggestion {
		return repoPathSuggestions(input, repoRoot, home)
	}
}

// completeRepoPathText returns the input after one Tab press, and whether Tab
// can make progress. It reads every matching directory, not just the ones
// suggested, because the common prefix must cover all of them.
func completeRepoPathText(input string, repoRoot string, home string) (string, bool) {
	dirText, prefix := splitRepoPathInput(input)
	names := matchRepoPathDirs(resolveRepoPath(dirText, repoRoot, home), prefix)
	return completeRepoPathMatches(dirText, prefix, names)
}

// repoPathSuggestions lists up to repoPathSuggestionLimit subdirectories of the
// typed directory whose names start with the typed segment. Each value is the
// full input with that name completed and a separator added.
func repoPathSuggestions(input string, repoRoot string, home string) []*types.Suggestion {
	dirText, prefix := splitRepoPathInput(normalizeRepoPathInput(input, repoRoot))
	names := matchRepoPathDirs(resolveRepoPath(dirText, repoRoot, home), prefix)
	names = names[:min(len(names), repoPathSuggestionLimit)]
	return lo.Map(names, func(name string, _ int) *types.Suggestion {
		return &types.Suggestion{
			Value: dirText + name + string(filepath.Separator),
			Label: name + "/",
		}
	})
}

// completeRepoPathMatches returns the input after one shell-style Tab press
// for the given matches. It inserts the longest common prefix of the names when
// that extends the typed prefix, and adds a separator when there is exactly one
// match. ok is false when Tab can't make progress.
func completeRepoPathMatches(dirText string, prefix string, names []string) (completed string, ok bool) {
	if len(names) == 0 {
		return "", false
	}
	common := longestCommonRunePrefix(names)
	if utf8.RuneCountInString(common) > utf8.RuneCountInString(prefix) {
		completed = dirText + common
	} else if len(names) == 1 {
		completed = dirText + names[0]
	} else {
		return "", false
	}
	if len(names) == 1 {
		completed += string(filepath.Separator)
	}
	return completed, completed != dirText+prefix
}

// longestCommonRunePrefix compares whole runes, so the result is valid UTF-8.
func longestCommonRunePrefix(names []string) string {
	common := []rune(names[0])
	for _, name := range names[1:] {
		runes := []rune(name)
		n := 0
		for n < len(common) && n < len(runes) && common[n] == runes[n] {
			n++
		}
		common = common[:n]
	}
	return string(common)
}
