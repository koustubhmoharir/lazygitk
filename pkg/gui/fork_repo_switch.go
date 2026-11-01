package gui

import "github.com/jesseduffield/lazygit/pkg/utils"

// Fork: forkRepoSwitchOpenedToast returns the completion callback of a repo
// switch's refresh (RefreshOptions.ThenIfRepoUnchanged). It toasts the name of
// the repo that has loaded. The name is taken when the refresh starts, as the
// Status pane shows it (RepoPaths.RepoName).
func (gui *Gui) forkRepoSwitchOpenedToast() func() {
	repo := gui.git.RepoPaths.RepoName()
	return func() {
		gui.c.Toast(utils.ResolvePlaceholderString(gui.Tr.RepoSwitchOpenedToast, map[string]string{"repo": repo}))
	}
}
