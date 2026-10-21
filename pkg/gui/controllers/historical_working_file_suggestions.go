package controllers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// historicalWorkingFileSuggestions browses the filesystem, not the index: an
// ignored or untracked file is just as useful a comparison target. Directories
// are editable suggestions for further browsing, not valid file destinations.
func historicalWorkingFileSuggestions(worktreeRoot string, gitDirs ...string) func(string) []*types.Suggestion {
	metadataDirs := make([]string, 0, len(gitDirs))
	for _, gitDir := range gitDirs {
		if gitDir == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(gitDir); err == nil {
			gitDir = resolved
		}
		if absolute, err := filepath.Abs(gitDir); err == nil {
			metadataDirs = append(metadataDirs, absolute)
		}
	}
	insideGitMetadata := func(candidate string) bool {
		for _, gitDir := range metadataDirs {
			if rel, err := filepath.Rel(gitDir, candidate); err == nil && filepath.IsLocal(rel) {
				return true
			}
		}
		return false
	}
	return func(input string) []*types.Suggestion {
		root, err := filepath.EvalSymlinks(worktreeRoot)
		if err != nil {
			return nil
		}
		root, err = filepath.Abs(root)
		if err != nil || strings.ContainsRune(input, '\x00') {
			return nil
		}
		dir, prefix := filepath.Split(input)
		searchDir := dir
		if !filepath.IsAbs(searchDir) {
			searchDir = filepath.Join(root, searchDir)
		}
		rel, err := filepath.Rel(root, searchDir)
		if err != nil || !filepath.IsLocal(rel) {
			return nil
		}
		if insideGitMetadata(searchDir) {
			return nil
		}
		current := root
		if rel != "." {
			for _, part := range strings.Split(rel, string(filepath.Separator)) {
				if strings.EqualFold(part, ".git") {
					return nil
				}
				current = filepath.Join(current, part)
				info, err := os.Lstat(current)
				if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					return nil
				}
			}
		}
		entries, err := os.ReadDir(searchDir)
		if err != nil {
			return nil
		}
		var suggestions []*types.Suggestion
		for _, entry := range entries {
			if insideGitMetadata(filepath.Join(searchDir, entry.Name())) {
				continue
			}
			if strings.EqualFold(entry.Name(), ".git") || !strings.HasPrefix(entry.Name(), prefix) || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			info, err := entry.Info()
			if err != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
				continue
			}
			value := filepath.Join(dir, entry.Name())
			if info.IsDir() {
				value += string(filepath.Separator)
			}
			suggestions = append(suggestions, &types.Suggestion{Value: value, Label: value})
		}
		return suggestions
	}
}
