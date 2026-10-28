package gui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// Fork: updateForkTerminalTitle sets the terminal title to the repository
// being viewed: the worktree root, or the submodule directory when viewing a
// submodule, with the home directory shown as "~". It is called from
// onNewRepo, which every repository change goes through (startup, recent
// repos, open by path, submodules, worktree switch). tcell saves the
// terminal's title on start and restores it on exit (CSI 22/23 t), and
// re-sends it after resuming, so no restore code is needed here.
func (gui *Gui) updateForkTerminalTitle() {
	if gocui.Screen == nil || gui.git == nil {
		return
	}
	home, _ := os.UserHomeDir()
	gocui.Screen.SetTitle(forkTerminalTitle(gui.git.RepoPaths.WorktreePath(), home))
}

// Fork: forkTerminalTitle replaces a leading home directory in worktreePath
// with "~". Paths that merely share a prefix with home are left unchanged.
func forkTerminalTitle(worktreePath, home string) string {
	if home == "" {
		return worktreePath
	}
	if worktreePath == home {
		return "~"
	}
	if strings.HasPrefix(worktreePath, home+string(filepath.Separator)) {
		return "~" + worktreePath[len(home):]
	}
	return worktreePath
}
