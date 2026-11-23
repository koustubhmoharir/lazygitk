package git_commands

import "github.com/jesseduffield/lazygit/pkg/commands/models"

// Fork: the full listing of a commit's files (item 1 of "browse a commit's
// files"). The changed files come from the diff as before, and the rest of the
// commit's tree is added as unchanged files.

// MergeTreePathsIntoCommitFiles returns every changed file as the diff gives it,
// followed by each path of the tree that the diff does not list, as an unchanged
// file with an empty ChangeStatus. A renamed file is listed once, under its new
// path, because the diff lists that path. A deleted file keeps its D status.
func MergeTreePathsIntoCommitFiles(diffFiles []*models.CommitFile, treePaths []string) []*models.CommitFile {
	result := make([]*models.CommitFile, 0, len(diffFiles)+len(treePaths))
	inDiff := make(map[string]bool, len(diffFiles))
	for _, file := range diffFiles {
		inDiff[file.Path] = true
		result = append(result, file)
	}
	for _, path := range treePaths {
		if !inDiff[path] {
			result = append(result, &models.CommitFile{Path: path})
		}
	}
	return result
}
