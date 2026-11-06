package helpers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: runMergeTool runs the external merge tool and refreshes Files whether the tool
// succeeds or fails. A tool can resolve and stage some files before it exits with an
// error, and upstream RunSubprocessAndRefresh returns on the error before refreshing.
// The error is still returned, so the caller shows it. Used by OpenMergeTool.
func (self *WorkingTreeHelper) runMergeTool(cmd *oscommands.CmdObj) error {
	_, err := self.c.RunSubprocess(cmd)
	self.c.Refresh(types.RefreshOptions{DontBlockRepoSwitch: true})
	return err
}
