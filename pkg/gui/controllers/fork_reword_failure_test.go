package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestRewordFailureMessage(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	err := errors.New("exit status 1")

	tests := []struct {
		name     string
		rebasing bool
		key      string
		expected string
	}{
		{
			name:     "rebasing with a key",
			rebasing: true,
			key:      "g o",
			expected: "Rewording failed: exit status 1\n\n" +
				"The rebase is still stopped. Press g o to abort or continue it before you try again.\n\n" +
				"Your message is saved. Press Up in the message box to get it back.",
		},
		{
			name:     "rebasing without a key",
			rebasing: true,
			key:      "",
			expected: "Rewording failed: exit status 1\n\n" +
				"The rebase is still stopped. Abort or continue it before you try again.\n\n" +
				"Your message is saved. Press Up in the message box to get it back.",
		},
		{
			name:     "not rebasing",
			rebasing: false,
			key:      "g o",
			expected: "Rewording failed: exit status 1\n\n" +
				"Your message is saved. Press Up in the message box to get it back.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, rewordFailureMessage(tr, err, test.rebasing, test.key))
		})
	}
}

// rewordFailureUI answers the key label call that the failure explanation makes, on top
// of the recording UI that the selected-commit rebase tests use.
type rewordFailureUI struct {
	*selectedRebaseFlowUI
	tree *guicontext.ContextTree
}

func (ui *rewordFailureUI) CommandKeyLabel(string, string) string {
	return "g o"
}

// rewordFailureFixture is a repository whose older commit cannot be reworded: a
// pre-commit hook rejects the amend that runs while the reword rebase is stopped at it.
func rewordFailureFixture(t *testing.T) (*LocalCommitsController, *rewordFailureUI, string) {
	t.Helper()
	root := t.TempDir()
	selectedRebaseFlowGit(t, root, "init", "-q")
	selectedRebaseFlowGit(t, root, "config", "user.name", "Test")
	selectedRebaseFlowGit(t, root, "config", "user.email", "test@example.invalid")
	ids := []string{}
	for _, name := range []string{"first", "second", "third"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		selectedRebaseFlowGit(t, root, "add", name)
		selectedRebaseFlowGit(t, root, "commit", "-qm", name)
		ids = append(ids, selectedRebaseFlowGit(t, root, "rev-parse", "HEAD"))
	}
	// Written after the commits, so that they are not rejected.
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho rejected by hook >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Moves into the test repository, so that a command which runs in the working
	// directory acts on it, not on the package's repository.
	t.Chdir(root)

	cmn := common.NewDummyCommon()
	git := commands.NewGitCommandAux(cmn, nil, oscommands.NewDummyOSCommand(), git_config.NewFakeGitConfig(nil), git_commands.MockRepoPaths(root), nil)
	git.Version = &git_commands.GitVersion{Major: 2, Minor: 51, Patch: 1}
	git.Config = git_commands.NewConfigCommands(cmn, git_config.NewFakeGitConfig(map[string]string{"rebase.updateRefs": "true"}))
	commits := []*models.Commit{historicalTestCommit(ids[2]), historicalTestCommit(ids[1]), historicalTestCommit(ids[0])}
	view := &guicontext.LocalCommitsViewModel{ListViewModel: guicontext.NewListViewModel(func() []*models.Commit { return commits })}
	view.SetSelection(1)
	tree := &guicontext.ContextTree{LocalCommits: &guicontext.LocalCommitsContext{LocalCommitsViewModel: view}}
	ui := &rewordFailureUI{
		selectedRebaseFlowUI: &selectedRebaseFlowUI{git: git, modes: &types.Modes{}, state: &selectedRebaseFlowState{repo: &selectedRebaseFlowRepoState{}}},
		tree:                 tree,
	}
	ui.model.Commits = commits

	basic := historicalTestBasicController(commits, 1, 1)
	basic.c.IGuiCommon = ui
	basic.c.HelperCommon.IGetContexts = historicalDiffContextsStub{tree: tree}
	basic.c.IGetHelpers = selectedRebaseFlowHelpers{value: &helpers.Helpers{Commits: &helpers.CommitsHelper{}}}
	return &LocalCommitsController{c: basic.c}, ui, root
}

// A failed reword of an older commit leaves the rebase stopped at that commit. The
// repository state must be refreshed before input is released, so that the model shows
// the stopped rebase, and the error explaining it is still returned.
func TestRewordOlderCommitFailureRefreshesRepositoryState(t *testing.T) {
	controller, ui, root := rewordFailureFixture(t)

	if err := controller.handleReword("reworded", ""); err != nil {
		t.Fatal(err)
	}
	err := ui.runWorker(t)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Rewording failed")
		assert.Contains(t, err.Error(), "rejected by hook")
	}
	_, statErr := os.Stat(filepath.Join(root, ".git", "rebase-merge"))
	assert.NoError(t, statErr, "the reword rebase should still be stopped")
	assert.Len(t, ui.refreshes, 1, "the repository state must be refreshed after the failure")
}
