package gui

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/samber/lo"
)

// Fork: automatic fetch never moves local branches on its own (U1). A remote
// whose fetch refspecs write into refs/heads/ is skipped, because automatic
// fetch cannot ask first. The skip is shown once per session.

// forkBackgroundFetchGuard holds the once-per-session state of the notice.
type forkBackgroundFetchGuard struct {
	noticeShown atomic.Bool
}

// forkSkipBackgroundFetch reports whether this automatic fetch must not run: one
// of the remotes it would fetch has a fetch refspec that writes into local
// branches. The first such skip of the session also shows a notice naming the
// remotes. If the refspecs cannot be read, the fetch is skipped as well, and
// the error is logged.
func (self *BackgroundRoutineMgr) forkSkipBackgroundFetch(git *commands.GitCommand) bool {
	risks, err := git.Sync.FetchBackgroundRefspecRisks()
	if err != nil {
		self.gui.c.Log.Warnf("Automatic fetch skipped: could not read the fetch refspecs: %v", err)
		return true
	}
	if len(risks) == 0 {
		return false
	}

	if self.forkFetchGuard.noticeShown.CompareAndSwap(false, true) {
		remotes := strings.Join(lo.Uniq(lo.Map(risks, func(risk git_commands.FetchRefspecRisk, _ int) string {
			return risk.Remote
		})), ", ")
		message := fmt.Sprintf(self.gui.Tr.ForkFetchBackgroundSkippedToast, remotes)
		self.gui.c.OnUIThread(func() error {
			self.gui.c.Toast(message)
			return nil
		})
	}
	return true
}
