package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestFileCommitOrContinueAction(t *testing.T) {
	rebasing := models.WorkingTreeState{Rebasing: true}
	merging := models.WorkingTreeState{Merging: true}
	cherryPicking := models.WorkingTreeState{CherryPicking: true}
	reverting := models.WorkingTreeState{Reverting: true}
	rebasingAndCherryPicking := models.WorkingTreeState{Rebasing: true, CherryPicking: true}

	tests := []struct {
		name         string
		state        models.WorkingTreeState
		stoppedEdit  bool
		anyStaged    bool
		wantedAction fileCommitOrContinue
	}{
		{"nothing in progress commits", models.WorkingTreeState{}, false, false, fileActionCommit},
		{"nothing in progress commits with staged changes", models.WorkingTreeState{}, false, true, fileActionCommit},
		{"edit stop with staged changes commits", rebasing, true, true, fileActionCommit},
		{"edit stop without staged changes continues", rebasing, true, false, fileActionContinue},
		{"conflict stop continues even with staged changes", rebasing, false, true, fileActionContinue},
		{"conflict stop without staged changes continues", rebasing, false, false, fileActionContinue},
		{"merge continues", merging, false, true, fileActionContinue},
		{"cherry-pick continues", cherryPicking, false, true, fileActionContinue},
		{"revert continues", reverting, false, true, fileActionContinue},
		{"cherry-pick inside a rebase continues the cherry-pick", rebasingAndCherryPicking, true, true, fileActionContinue},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.wantedAction, fileCommitOrContinueAction(test.state, test.stoppedEdit, test.anyStaged))
		})
	}
}
