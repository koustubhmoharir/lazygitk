package git_commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
// The diff does not say which of its files are submodules. A changed file is one
// when the commit's tree says so (treeFiles), and a deleted file when the parent's
// tree does (fromTreeFiles, which may be nil when there are no deleted files to look
// up).
func MergeTreePathsIntoCommitFiles(diffFiles []*models.CommitFile, treeFiles []*models.CommitFile, fromTreeFiles []*models.CommitFile) []*models.CommitFile {
	result := make([]*models.CommitFile, 0, len(diffFiles)+len(treeFiles))

	toSubmodules := submodulePaths(treeFiles)
	fromSubmodules := submodulePaths(fromTreeFiles)

	inDiff := make(map[string]bool, len(diffFiles))
	for _, file := range diffFiles {
		inDiff[file.Path] = true
		isSubmodule := toSubmodules[file.Path]
		if file.Deleted() {
			isSubmodule = fromSubmodules[file.Path]
		}
		if isSubmodule && !file.Submodule {
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

// TreeEntry is an entry of a tree, as ls-tree -l lists it.
type TreeEntry struct {
	// Path is the entry's path from the repository root.
	Path string
	// Type is "blob" for a file, "tree" for a folder and "commit" for a submodule.
	Type string
	// Object is the hash of the entry's object: the blob, the tree or, for a
	// submodule, the commit it records.
	Object string
	// Size is the size of a file in bytes, and -1 for a folder or a submodule.
	Size int64
}

// treeEntriesFromLongListing parses the output of ls-tree -l -z: a NUL-terminated
// record per entry, each of the form "<mode> SP <type> SP <object> SP+ <size> TAB <path>",
// where the size is "-" for a tree or a commit.
func treeEntriesFromLongListing(output string) ([]TreeEntry, error) {
	records := strings.Split(output, "\x00")
	entries := make([]TreeEntry, 0, len(records))
	for _, record := range records {
		if record == "" {
			continue
		}
		meta, path, found := strings.Cut(record, "\t")
		fields := strings.Fields(meta)
		if !found || len(fields) != 4 {
			return nil, fmt.Errorf("malformed git tree entry: %q", record)
		}
		size := int64(-1)
		if fields[3] != "-" {
			parsed, err := strconv.ParseInt(fields[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("malformed git tree entry size: %q", record)
			}
			size = parsed
		}
		entries = append(entries, TreeEntry{Path: path, Type: fields[1], Object: fields[2], Size: size})
	}
	return entries, nil
}

// GetTreeEntries lists the entries of the tree of ref that pathspec names: the
// entries of a folder when pathspec ends in a slash, the one entry it names
// otherwise, and the root's entries when pathspec is empty. The pathspec is literal,
// so a name with glob characters is not a pattern.
func (self *CommitFileLoader) GetTreeEntries(ref string, pathspec string) ([]TreeEntry, error) {
	cmdArgs := NewGitCmd("--literal-pathspecs").
		Arg("ls-tree", "-z", "-l", "--full-tree", ref).
		ArgIf(pathspec != "", "--", pathspec).
		ToArgv()

	output, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}

	return treeEntriesFromLongListing(output)
}

// SubmoduleSubject returns the subject of the commit hash in the checkout of a
// submodule at dir, or "" when there is no checkout there or the commit is not in it.
// Git is not run in dir unless it has a .git of its own, so that an empty directory
// does not read the superproject instead.
func (self *CommitFileLoader) SubmoduleSubject(dir string, hash string) string {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return ""
	}

	cmdArgs := NewGitCmd("log").
		Arg("-1", "--format=%s", hash).
		ToArgv()

	output, err := self.cmd.New(cmdArgs).DontLog().SetWd(dir).RunWithOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

// submodulePaths is the set of the paths of the submodules among files.
func submodulePaths(files []*models.CommitFile) map[string]bool {
	paths := make(map[string]bool, len(files))
	for _, file := range files {
		if file.Submodule {
			paths[file.Path] = true
		}
	}
	return paths
}
