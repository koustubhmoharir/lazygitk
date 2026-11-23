package git_commands

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

// Fork: the full listing of a commit's files (item 1 of "browse a commit's
// files"). The changed files come from the diff as before, and the rest of the
// commit's tree is added as unchanged files.

// treeFilesFromListing parses the output of ls-tree -r -z: a NUL-terminated record per
// entry, each of the form "<mode> SP <type> SP <object> TAB <path>". The path is what
// follows the first TAB, so it may contain anything but a NUL. An entry of type commit
// is a submodule.
func treeFilesFromListing(output string) ([]*models.CommitFile, error) {
	records := strings.Split(output, "\x00")
	files := make([]*models.CommitFile, 0, len(records))
	for _, record := range records {
		if record == "" {
			continue
		}
		meta, path, found := strings.Cut(record, "\t")
		fields := strings.Fields(meta)
		if !found || len(fields) != 3 {
			return nil, fmt.Errorf("malformed git tree entry: %q", record)
		}
		files = append(files, &models.CommitFile{Path: path, Submodule: fields[1] == "commit"})
	}
	return files, nil
}

// MergeTreePathsIntoCommitFiles returns every changed file as the diff gives it,
// followed by each file of the tree that the diff does not list, as an unchanged
// file with an empty ChangeStatus. A renamed file is listed once, under its new
// path, because the diff lists that path. A deleted file keeps its D status.
//
// A deleted file is a submodule when the parent's tree says so (fromTreeFiles, which
// may be nil when there are no deleted files to look up); the diff does not say.
func MergeTreePathsIntoCommitFiles(diffFiles []*models.CommitFile, treeFiles []*models.CommitFile, fromTreeFiles []*models.CommitFile) []*models.CommitFile {
	result := make([]*models.CommitFile, 0, len(diffFiles)+len(treeFiles))

	fromSubmodules := make(map[string]bool, len(fromTreeFiles))
	for _, file := range fromTreeFiles {
		if file.Submodule {
			fromSubmodules[file.Path] = true
		}
	}

	inDiff := make(map[string]bool, len(diffFiles))
	for _, file := range diffFiles {
		inDiff[file.Path] = true
		if file.Deleted() && fromSubmodules[file.Path] {
			marked := *file
			marked.Submodule = true
			file = &marked
		}
		result = append(result, file)
	}
	for _, file := range treeFiles {
		if !inDiff[file.Path] {
			result = append(result, file)
		}
	}
	return result
}
