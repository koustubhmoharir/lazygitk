package git_commands

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

// Fork: the commit-index entry points of the rebase and patch commands refuse an index
// outside the commits list with this error, rather than panicking on it. A patch whose
// source commit is missing from the list passes -1 (see the GUI's getPatchCommitIndex).
// Upstream checked only the upper bound, and a bad lower bound panicked.
func requireCommitIndex(commits []*models.Commit, index int) error {
	if index < 0 || index >= len(commits) {
		return errors.New("index outside of range of commits")
	}
	return nil
}
