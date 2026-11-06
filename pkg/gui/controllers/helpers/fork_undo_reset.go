package helpers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: undo and redo of a commit or rebase (controllers/undo_controller.go)
// replace upstream's hardResetWithAutoStash, which stashed and popped on the
// user's behalf. This runs the captured hard reset to the reflog target instead.
// Tracked changes are refused and never stashed. Otherwise the reset goes through
// the same review as s h, with one confirmation titled Undo or Redo.

// ReflogReplay says whether a reflog target is reached by an undo or by a redo.
type ReflogReplay int

const (
	ReflogUndo ReflogReplay = iota
	ReflogRedo
)

// ResetReflogTarget hard-resets the current branch to target, the reflog text of
// an undo or redo. envVars are set on the reset command, so its reflog entry keeps
// the undo or redo marker that the reflog parser reads.
func (self *RefsHelper) ResetReflogTarget(target string, replay ReflogReplay, envVars []string) error {
	title, effect, refusal := self.c.Tr.Actions.Undo, self.c.Tr.UndoResetEffect, self.c.Tr.UndoRefuseTrackedChanges
	if replay == ReflogRedo {
		title, effect, refusal = self.c.Tr.Actions.Redo, self.c.Tr.RedoResetEffect, self.c.Tr.RedoRefuseTrackedChanges
	}
	return resetCurrentToCapturedSource(self.c, utils.ShortHash(target), target, capturedResetOptions{
		mode: "hard", title: title, effect: effect, envVars: envVars,
		preflight: func(branch *git_commands.BranchCommands) error {
			dirty, err := branch.TrackedChangesPresent()
			if err != nil {
				return err
			}
			if dirty {
				return errors.New(refusal)
			}
			return nil
		},
	})
}
