package controllers

import (
	"fmt"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: the text of the folder listing and of the submodule view in the full listing of
// a commit's files (see fork_commit_folder.go). The functions here take what git said,
// as values, so that the layout is tested without git.

// folderRow is a line of a folder listing.
type folderRow struct {
	// name is the entry's name, with a trailing slash for a folder.
	name string
	// status is the status letter of a file the commit changed or deleted, or "".
	status string
	// size is the file's size, the submodule marker for a submodule, or "" for a folder.
	size string
}

// folderListingRows returns the immediate entries of the folder whose path prefix is
// prefix (with a trailing slash, or "" for the root): folders first, then files, each
// sorted by name. entries are the folder's entries at the commit. changed are the
// commit's changed files under the folder, with their statuses; a file that is deleted
// (status D) and not among entries is listed as deleted. fromEntries are the folder's
// entries at the commit's parent, for the sizes of the deleted files; it may be nil.
// submoduleMarker is what a submodule's size column shows.
func folderListingRows(prefix string, entries []git_commands.TreeEntry, changed []*models.CommitFile, fromEntries []git_commands.TreeEntry, submoduleMarker string) []folderRow {
	statuses := make(map[string]string, len(changed))
	for _, file := range changed {
		statuses[file.Path] = file.ChangeStatus
	}
	fromSizes := make(map[string]git_commands.TreeEntry, len(fromEntries))
	for _, entry := range fromEntries {
		fromSizes[entry.Path] = entry
	}

	var folders, files []folderRow
	present := make(map[string]bool, len(entries))
	for _, entry := range entries {
		name := path.Base(entry.Path)
		present[name] = true
		switch entry.Type {
		case "tree":
			folders = append(folders, folderRow{name: name + "/"})
		case "commit":
			files = append(files, folderRow{name: name, status: statuses[entry.Path], size: submoduleMarker})
		default:
			files = append(files, folderRow{name: name, status: statuses[entry.Path], size: formatFileSize(entry.Size)})
		}
	}

	// A deleted entry is absent from the commit's tree, so it is taken from the changed
	// files. A deleted folder is listed once, as deleted, however many of its files the
	// commit deleted.
	deletedFolders := make(map[string]bool)
	for _, file := range changed {
		if file.ChangeStatus != "D" {
			continue
		}
		rest, ok := strings.CutPrefix(file.Path, prefix)
		if !ok {
			continue
		}
		name, _, isBelow := strings.Cut(rest, "/")
		if present[name] {
			continue
		}
		if isBelow {
			if !deletedFolders[name] {
				deletedFolders[name] = true
				folders = append(folders, folderRow{name: name + "/", status: "D"})
			}
			continue
		}
		size := ""
		if file.Submodule {
			size = submoduleMarker
		} else if entry, found := fromSizes[file.Path]; found && entry.Size >= 0 {
			size = formatFileSize(entry.Size)
		}
		files = append(files, folderRow{name: name, status: "D", size: size})
	}

	sortRowsByName(folders)
	sortRowsByName(files)
	return append(folders, files...)
}

func sortRowsByName(rows []folderRow) {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
}

// formatFolderRows lays the rows out one per line, with the status letter in a column on
// the left, the names padded to a common width, and the sizes right-aligned in a column
// after them. A folder has neither a status nor a size, so its line holds only its name.
func formatFolderRows(rows []folderRow) string {
	nameWidth, sizeWidth := 0, 0
	for _, row := range rows {
		nameWidth = max(nameWidth, utils.StringWidth(row.name))
		sizeWidth = max(sizeWidth, utils.StringWidth(row.size))
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		status := " "
		if row.status != "" {
			status = row.status
		}
		line := status + " " + row.name + strings.Repeat(" ", nameWidth-utils.StringWidth(row.name))
		if sizeWidth > 0 {
			line += "  " + strings.Repeat(" ", sizeWidth-utils.StringWidth(row.size)) + row.size
		}
		lines = append(lines, strings.TrimRight(line, " "))
	}
	return strings.Join(lines, "\n")
}

// formatFileSize is the size of a file in bytes, in the units people read: bytes below
// 1 KB, and then KB, MB, GB or TB with one decimal place, in powers of 1024.
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	value := float64(size) / 1024
	units := []string{"KB", "MB", "GB", "TB"}
	for i, unit := range units {
		// The value is rounded to the decimal place shown before the unit is chosen, so
		// that 1023.97 KB reads as 1.0 MB rather than 1024.0 KB.
		if math.Round(value*10)/10 < 1024 || i == len(units)-1 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
		value /= 1024
	}
	return ""
}

// folderEntriesText says how many entries a folder listing lists: many is the template
// for any count but one, which is one.
func folderEntriesText(many string, one string, count int) string {
	if count == 1 {
		return one
	}
	return utils.ResolvePlaceholderString(many, map[string]string{"count": strconv.Itoa(count)})
}

// folderListingTitle is the title of a folder listing with the number of entries it
// lists: the folder's title, then the count.
func folderListingTitle(folderTitle string, entriesText string) string {
	return folderTitle + " " + entriesText
}

// submoduleLabels are the labels of the submodule view's lines.
type submoduleLabels struct {
	commit   string
	previous string
	removed  string
}

// submoduleListingText is the text of a submodule's view. current is the entry the commit
// records for the submodule, nil when the commit removed it; previous is the entry the
// commit's parent records, nil when the commit did not change it. subject gives the
// subject of a commit in the submodule, "" when it has none to show.
func submoduleListingText(labels submoduleLabels, current *git_commands.TreeEntry, previous *git_commands.TreeEntry, subject func(hash string) string) string {
	var lines []string
	if current != nil {
		lines = append(lines, submoduleLine(labels.commit, current.Object, subject))
	} else {
		lines = append(lines, fmt.Sprintf("%-11s%s", labels.commit, labels.removed))
	}
	if previous != nil {
		lines = append(lines, submoduleLine(labels.previous, previous.Object, subject))
	}
	return strings.Join(lines, "\n")
}

// submoduleLine is a line of the submodule view: the label, then the full hash and the
// commit's subject, when there is one.
func submoduleLine(label string, hash string, subject func(hash string) string) string {
	value := hash
	if s := subject(hash); s != "" {
		value += " " + s
	}
	return fmt.Sprintf("%-11s%s", label, value)
}
