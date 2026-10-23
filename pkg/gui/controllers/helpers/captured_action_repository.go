package helpers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands"
)

// Retain both backend identity and UI repository generation. Returning to the
// same backend after visiting another repository must not revive old callbacks.
type capturedActionRepository struct {
	c          *HelperCommon
	git        *commands.GitCommand
	generation int
}

func captureActionRepository(c *HelperCommon) *capturedActionRepository {
	return &capturedActionRepository{c: c, git: c.Git(), generation: c.State().GetRepoGeneration()}
}

func (target *capturedActionRepository) checkRepository() error {
	if target.c.Git() != target.git || target.c.State().GetRepoGeneration() != target.generation {
		return errors.New(target.c.Tr.BranchSyncRepositoryChanged)
	}
	return nil
}
