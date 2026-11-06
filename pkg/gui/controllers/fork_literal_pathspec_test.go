package controllers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// Fork: real-Git regressions for the Files and commit-file copy and difftool
// actions. A selected name such as a*.txt is a pathspec pattern unless the
// command runs with literal pathspecs, so it must not also select a2.txt.

// literalPathspecFixture returns a Files fixture whose repository has a*.txt,
// a2.txt and dir/x.txt. The last commit changes all three, and the working tree
// then has unstaged changes to all three.
func literalPathspecFixture(t *testing.T) *workingFileDiffFixture {
	t.Helper()
	fixture := newWorkingFileDiffFixture(t, []*models.File{
		{Path: "a*.txt", Tracked: true, HasUnstagedChanges: true},
		{Path: "a2.txt", Tracked: true, HasUnstagedChanges: true},
	}, workingFileTestBranches(""))
	writeAll := func(content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(fixture.root, "dir"), 0o700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"a*.txt", "a2.txt", "dir/x.txt"} {
			if err := os.WriteFile(filepath.Join(fixture.root, name), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	writeAll("one\n")
	literalGit(t, fixture.root, "add", "-A")
	literalGit(t, fixture.root, "commit", "-qm", "base")
	writeAll("two\n")
	literalGit(t, fixture.root, "commit", "-qam", "change")
	writeAll("three\n")
	fixture.ui.model.Branches = workingFileTestBranches(literalGit(t, fixture.root, "rev-parse", "HEAD"))
	fixture.ui.model.CommitFiles = []*models.CommitFile{{Path: "a*.txt"}, {Path: "a2.txt"}, {Path: "dir/x.txt"}}
	// A harmless tool that prints the file it was given, so the output shows
	// exactly which files git selected.
	literalGit(t, fixture.root, "config", "difftool.fake.cmd", `printf '%s\n' "$MERGED"`)
	return fixture
}

func literalGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// runRecordedDifftool runs the one difftool command the action recorded, in the
// repository, and returns what the tool printed.
func runRecordedDifftool(t *testing.T, fixture *workingFileDiffFixture) string {
	t.Helper()
	if len(fixture.ui.commands) != 1 {
		t.Fatalf("subprocesses = %d, want 1", len(fixture.ui.commands))
	}
	cmd := fixture.ui.commands[0]
	cmd.SetWd(fixture.root)
	out, err := cmd.RunWithOutput()
	if err != nil {
		t.Fatalf("difftool: %v\n%s", err, out)
	}
	return out
}

func TestWorkingFileDiffIndexLiteralPathspec(t *testing.T) {
	fixture := literalPathspecFixture(t)
	if err := fixture.controller.diffWorkingFileWithStaged(fixture.node(t, "a*.txt")); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "a*.txt\n", runRecordedDifftool(t, fixture), "the difftool must open only the selected file")
}

func TestFilesOpenDiffToolLiteralPathspec(t *testing.T) {
	fixture := literalPathspecFixture(t)
	// Diff mode is off, so the route compares the working file with the index.
	if err := fixture.controller.openDiffTool(fixture.node(t, "a*.txt")); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "a*.txt\n", runRecordedDifftool(t, fixture), "the difftool must open only the selected file")
}

func TestCopyWorkingFileDiffLiteralPathspec(t *testing.T) {
	fixture := literalPathspecFixture(t)
	fixture.node(t, "a*.txt")
	item, err := findMenuItem(fixture.controller.copyMenuItems(), "clipboard.copy-working-file-diff")
	if err != nil {
		t.Fatal(err)
	}
	if err := item.OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(fixture.ui.clipboard) != 1 {
		t.Fatalf("clipboard writes = %d, want 1", len(fixture.ui.clipboard))
	}
	diff := fixture.ui.clipboard[0]
	assert.Contains(t, diff, "diff --git a/a*.txt b/a*.txt")
	assert.NotContains(t, diff, "a2.txt", "the copied diff must not include a2.txt")
}

// newLiteralCommitFiles returns a commit-files controller and context for the
// commit, whose parent comes from the repository, with the commit's files
// loaded into the tree.
func newLiteralCommitFiles(t *testing.T, fixture *workingFileDiffFixture, commit string) (*CommitFilesController, *guicontext.CommitFilesContext) {
	t.Helper()
	parent := literalGit(t, fixture.root, "rev-parse", commit+"^")
	cmn := common.NewDummyCommon()
	contextCommon := &guicontext.ContextCommon{Common: cmn, IGuiCommon: fixture.ui}
	commitFiles := guicontext.NewCommitFilesContext(contextCommon)
	// The commit is set directly: ReInit also titles the view, which these
	// tests do not have.
	commitFiles.SetRef(models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: commit, Parents: []string{parent}}))
	commitFiles.SetRefRange(nil)
	commitFiles.SetTree()
	helperCommon := &helpers.HelperCommon{
		Common:       cmn,
		IGuiCommon:   fixture.ui,
		IGetContexts: historicalDiffContextsStub{tree: &guicontext.ContextTree{CommitFiles: commitFiles}},
	}
	helpersStub := historicalTestHelpers{value: &helpers.Helpers{ExternalTool: helpers.NewExternalToolHelper(helperCommon)}}
	return NewCommitFilesController(NewControllerCommon(helperCommon, helpersStub)), commitFiles
}

func TestCopyCommitFileDiffLiteralPathspec(t *testing.T) {
	fixture := literalPathspecFixture(t)
	controller, _ := newLiteralCommitFiles(t, fixture, literalGit(t, fixture.root, "rev-parse", "HEAD"))

	tests := []struct {
		name    string
		paths   []string
		include string
		exclude string
	}{
		{name: "pattern-like file", paths: []string{"a*.txt"}, include: "diff --git a/a*.txt b/a*.txt", exclude: "a2.txt"},
		{name: "directory", paths: []string{"dir"}, include: "diff --git a/dir/x.txt", exclude: "a*.txt"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture.ui.clipboard = nil
			if err := controller.copyDiffToClipboard(tc.paths, "copied"); err != nil {
				t.Fatal(err)
			}
			if len(fixture.ui.clipboard) != 1 {
				t.Fatalf("clipboard writes = %d, want 1", len(fixture.ui.clipboard))
			}
			diff := fixture.ui.clipboard[0]
			assert.Contains(t, diff, tc.include)
			assert.NotContains(t, diff, tc.exclude)
		})
	}

	// Rename endpoint: a*.txt is renamed to renamed.txt, and a2.txt changes in
	// the same commit. The old path is a pattern, so a literal path must copy
	// only the rename. The commit first records the content so that Git pairs
	// the two ends of the rename.
	t.Run("rename endpoint", func(t *testing.T) {
		literalGit(t, fixture.root, "commit", "-qam", "three")
		if err := os.Rename(filepath.Join(fixture.root, "a*.txt"), filepath.Join(fixture.root, "renamed.txt")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fixture.root, "a2.txt"), []byte("four\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		literalGit(t, fixture.root, "add", "-A")
		literalGit(t, fixture.root, "commit", "-qm", "rename")
		controller, _ := newLiteralCommitFiles(t, fixture, literalGit(t, fixture.root, "rev-parse", "HEAD"))

		fixture.ui.clipboard = nil
		if err := controller.copyDiffToClipboard([]string{"renamed.txt", "a*.txt"}, "copied"); err != nil {
			t.Fatal(err)
		}
		if len(fixture.ui.clipboard) != 1 {
			t.Fatalf("clipboard writes = %d, want 1", len(fixture.ui.clipboard))
		}
		diff := fixture.ui.clipboard[0]
		assert.Contains(t, diff, "rename from a*.txt")
		assert.NotContains(t, diff, "a2.txt", "the copied diff must not include a2.txt")
	})
}

func TestCommitFilesOpenDiffToolLiteralPathspec(t *testing.T) {
	fixture := literalPathspecFixture(t)
	controller, commitFiles := newLiteralCommitFiles(t, fixture, literalGit(t, fixture.root, "rev-parse", "HEAD"))
	var node *filetree.CommitFileNode
	for i := range commitFiles.Len() {
		commitFiles.SetSelectedLineIdx(i)
		if candidate := commitFiles.GetSelected(); candidate != nil && candidate.GetPath() == "a*.txt" {
			node = candidate
			break
		}
	}
	if node == nil {
		t.Fatal(`no commit file "a*.txt"`)
	}
	if err := controller.openDiffTool(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "a*.txt\n", runRecordedDifftool(t, fixture), "the difftool must open only the selected file")
}
