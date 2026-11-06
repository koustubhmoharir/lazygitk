package helpers

import (
	"os"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: feedback for a repo switch (docs-master/fork/Tracker.md). A switch
// blanks every side pane and the main area to "Loading…" before its
// synchronous work, so no old content shows under the new repo. If the switch
// fails, the repo we stayed in is rendered again from its model.

// repoSwitchPaneContexts returns the contexts whose views a repo switch blanks:
// every side context (all tabs of each side panel, not just the visible ones)
// and the main contexts. Popups are not included.
func repoSwitchPaneContexts(contexts []types.Context) []types.Context {
	return lo.Filter(contexts, func(ctx types.Context, _ int) bool {
		kind := ctx.GetKind()
		return kind == types.SIDE_CONTEXT || kind == types.MAIN_CONTEXT
	})
}

// repoSwitchPaneViewNames returns the names of the views that repoSwitchPaneContexts
// selects, each once.
func repoSwitchPaneViewNames(contexts []types.Context) []string {
	return lo.Uniq(lo.FilterMap(repoSwitchPaneContexts(contexts), func(ctx types.Context, _ int) (string, bool) {
		name := ctx.GetViewName()
		return name, name != ""
	}))
}

// showRepoSwitchLoading shows "Loading…" in every pane and draws it at once,
// because the switch that follows holds the UI thread until the new repo is
// set up, and the main loop cannot draw during that time. Keys pressed meanwhile
// queue as usual.
func (self *ReposHelper) showRepoSwitchLoading() {
	gui := self.c.GocuiGui()
	var views []*gocui.View
	for _, name := range repoSwitchPaneViewNames(self.c.Contexts().Flatten()) {
		view, err := gui.View(name)
		if err != nil {
			continue
		}
		view.SetContent(self.c.Tr.RepoSwitchLoading)
		views = append(views, view)
	}
	gui.ForceFlushViewsContentOnly(views)
}

// notAGitRepoError is what a switch returns when git cannot open the directory
// as a repository. The repository path prompt keeps its text for this error;
// any other failure of a switch is shown as an error and the prompt closes.
type notAGitRepoError struct{ err error }

func (e notAGitRepoError) Error() string { return e.err.Error() }
func (e notAGitRepoError) Unwrap() error { return e.err }

// restoreCurrentRepo puts back what a failed switch changed before the new repo
// took effect: the working directory and the process environment (which the
// switch's direnv load and location variables set), then the panes. The
// models still hold the repo we stayed in, so its panes are rendered again.
func (self *ReposHelper) restoreCurrentRepo(originalDir string, originalEnv []string) {
	if err := os.Chdir(originalDir); err != nil {
		self.c.Log.WithError(err).Error("could not return to the current repository's directory")
	}
	restoreProcessEnv(originalEnv)

	for _, ctx := range repoSwitchPaneContexts(self.c.Contexts().Flatten()) {
		ctx.HandleRender()
	}
	self.c.Context().CurrentSide().HandleRenderToMain()
	// The status pane has no context to render it: refresh draws it from the
	// models, as the refresh of any other repo state does.
	self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.STATUS}})
}

// restoreProcessEnv makes the process environment match a snapshot taken with
// os.Environ: variables added since are removed, and changed ones set back.
func restoreProcessEnv(snapshot []string) {
	saved := make(map[string]string, len(snapshot))
	for _, entry := range snapshot {
		if name, value, ok := strings.Cut(entry, "="); ok {
			saved[name] = value
		}
	}
	for _, entry := range os.Environ() {
		if name, _, ok := strings.Cut(entry, "="); ok {
			if _, wasSet := saved[name]; !wasSet {
				_ = os.Unsetenv(name)
			}
		}
	}
	for name, value := range saved {
		if current, ok := os.LookupEnv(name); !ok || current != value {
			_ = os.Setenv(name, value)
		}
	}
}
