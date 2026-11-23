package controllers

import (
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Fork: in the full listing of a commit's files, a selected folder (or the root) shows the
// listing of its entries, and a selected submodule shows the commit it records, rather
// than a diff of them (see context.CommitFileShowFor). The listing is read from git on
// the render's goroutine (see types.LoadedStringTask), from the commit's tree and, for
// the sizes of the files it deleted, its parent's. The text is laid out by
// fork_commit_folder_listing.go.

// renderFullListingFolderOrSubmodule renders the selected node as a folder listing or a
// submodule, and reports whether it did. It reports false for anything else.
func (self *CommitFilesController) renderFullListingFolderOrSubmodule(node *filetree.CommitFileNode) bool {
	from, to := self.context().GetFromAndToForDiff()
	hasParent := context.CommitFileHasParent(from)
	show := context.CommitFileShowFor(self.context().IsFullListing(), node, hasParent)

	var title, key string
	var load func() (string, string)
	switch show {
	case context.CommitFileShowFolder:
		title, key, load = self.folderListingLoad(node, from, to, hasParent)
	case context.CommitFileShowSubmodule:
		title, key, load = self.submoduleLoad(node, from, to, hasParent)
	default:
		return false
	}

	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title: title,
			Task:  types.NewLoadedStringTask(key, load),
		},
		Secondary: secondaryPatchPanelUpdateOpts(self.c),
	})
	return true
}

// folderListingLoad captures what the listing of a folder reads from the model, on the UI
// thread, and returns the view's title before the listing is loaded, its key and the
// function that loads its title and text. The loaded title also names the entry count.
func (self *CommitFilesController) folderListingLoad(node *filetree.CommitFileNode, from string, to string, hasParent bool) (string, string, func() (string, string)) {
	folder := node.GetPath()
	// The root item's path is "."; the root's entries are the tree's own.
	prefix := ""
	displayPath := "/"
	if folder != "." {
		prefix = folder + "/"
		displayPath = prefix
	}

	changed := lo.Filter(self.c.Model().CommitFiles, func(file *models.CommitFile, _ int) bool {
		return self.context().IsChangedFile(file) && strings.HasPrefix(file.Path, prefix)
	})
	hasDeleted := lo.ContainsBy(changed, (*models.CommitFile).Deleted)

	loader := self.c.Git().Loaders.CommitFileLoader
	tr := self.c.Tr
	title := utils.ResolvePlaceholderString(tr.FolderListingTitle, map[string]string{"path": displayPath})
	submoduleMarker := tr.FolderListingSubmodule

	load := func() (string, string) {
		entries, err := loader.GetTreeEntries(to, prefix)
		if err != nil {
			return title, err.Error()
		}
		var fromEntries []git_commands.TreeEntry
		if hasParent && hasDeleted {
			fromEntries, err = loader.GetTreeEntries(from, prefix)
			if err != nil {
				return title, err.Error()
			}
		}
		rows := folderListingRows(prefix, entries, changed, fromEntries, submoduleMarker)
		count := len(rows)
		entriesText := folderEntriesText(tr.FolderListingEntries, tr.FolderListingEntry, count)
		return folderListingTitle(title, entriesText), formatFolderRows(rows)
	}

	return title, "folder\x00" + from + "\x00" + to + "\x00" + prefix, load
}

// submoduleLoad captures what the view of a submodule reads from the model, on the UI
// thread, and returns the view's title, its key and the function that loads its text.
func (self *CommitFilesController) submoduleLoad(node *filetree.CommitFileNode, from string, to string, hasParent bool) (string, string, func() (string, string)) {
	submodulePath := node.GetPath()
	status := node.File.ChangeStatus

	loader := self.c.Git().Loaders.CommitFileLoader
	worktree := self.c.Git().RepoPaths.WorktreePath()
	tr := self.c.Tr
	title := utils.ResolvePlaceholderString(tr.SubmoduleTitle, map[string]string{"path": submodulePath})
	labels := submoduleLabels{commit: tr.SubmoduleCommitLabel, previous: tr.SubmodulePreviousLabel, removed: tr.SubmoduleRemoved}

	load := func() (string, string) {
		// The entry is looked up by its path alone, as a pathspec names one entry.
		current, err := submoduleEntry(loader, to, submodulePath)
		if err != nil {
			return title, err.Error()
		}
		// The previous commit is shown only when this commit changed the submodule.
		var previous *git_commands.TreeEntry
		if hasParent && status != "" {
			previous, err = submoduleEntry(loader, from, submodulePath)
			if err != nil {
				return title, err.Error()
			}
		}
		subject := func(hash string) string {
			return loader.SubmoduleSubject(filepath.Join(worktree, filepath.FromSlash(submodulePath)), hash)
		}
		return title, submoduleListingText(labels, current, previous, subject)
	}

	return title, "submodule\x00" + from + "\x00" + to + "\x00" + submodulePath, load
}

// submoduleEntry is the entry at path in the tree of ref, nil when the tree has none.
func submoduleEntry(loader *git_commands.CommitFileLoader, ref string, path string) (*git_commands.TreeEntry, error) {
	entries, err := loader.GetTreeEntries(ref, path)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Path == path {
			return &entries[i], nil
		}
	}
	return nil, nil
}
