package controllers

import (
	"errors"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: repo.fetch is a global binding so f fetches every remote from any
// normal pane (roadmap item 19). Remotes keeps f for remote.fetch, which
// wins as a view-local binding. The handler is the one upstream Files used.
func forkFetchBinding(c *ControllerCommon, opts types.KeybindingsOpts) *types.Binding {
	return &types.Binding{
		CommandID:   "repo.fetch",
		Verb:        "fetch",
		Keys:        opts.GetKeys(opts.Config.Files.Fetch),
		Handler:     func() error { return forkFetch(c) },
		Description: c.Tr.Fetch,
		Tooltip:     c.Tr.FetchTooltip,
	}
}

func forkFetch(c *ControllerCommon) error {
	fetchGeneration := c.State().GetRepoGeneration()
	return c.WithWaitingStatus(c.Tr.FetchingStatus, func(task gocui.Task) error {
		c.LogAction("Fetch")
		err := c.Git().Sync.Fetch(task)

		if err != nil && strings.Contains(err.Error(), "exit status 128") {
			return errors.New(c.Tr.PassUnameWrong)
		}

		return c.Helpers().BranchesHelper.PostFetchRefresh(err, false, fetchGeneration)
	})
}
