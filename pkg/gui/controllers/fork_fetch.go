package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: repo.fetch is a global binding so f fetches from any normal pane
// (roadmap item 19). It fetches the primary remote: origin, or the repository's
// only remote. With several remotes and no origin it asks which one to fetch.
// Remotes keeps f for remote.fetch, which wins as a view-local binding. The
// fetch ignores git.fetchAll (U1).
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

// Fork: palette-only "Fetch all remotes" (U1). It has no key path of its own.
func forkFetchAllBinding(c *ControllerCommon) *types.Binding {
	return &types.Binding{
		CommandID:   "repo.fetch-all",
		Verb:        "fetch",
		Description: c.Tr.ForkFetchAllRemotes,
		Handler:     func() error { return forkFetchAll(c) },
	}
}

func forkFetch(c *ControllerCommon) error {
	remotes, err := c.Git().Sync.RemoteNames()
	if err != nil {
		return err
	}
	if len(remotes) == 0 {
		return errors.New(c.Tr.ForkFetchNoRemotes)
	}
	if remote, ok := git_commands.PrimaryFetchRemote(remotes); ok {
		return forkFetchRemote(c, remote)
	}

	items := lo.Map(remotes, func(remote string, _ int) *types.MenuItem {
		return &types.MenuItem{
			CommandID: "repo.fetch-remote",
			Verb:      "fetch",
			Label:     remote,
			OnPress:   func() error { return forkFetchRemote(c, remote) },
		}
	})
	items = append(items, &types.MenuItem{
		CommandID: "repo.fetch-all",
		Verb:      "fetch",
		Label:     c.Tr.ForkFetchAllRemotes,
		OnPress:   func() error { return forkFetchAll(c) },
	})
	return c.Menu(types.CreateMenuOptions{
		Title:  c.Tr.Fetch,
		Prompt: c.Tr.ForkFetchMenuPrompt,
		Items:  items,
	})
}

// Fork: forkFetchRemote fetches one remote, as f does for the primary remote and
// the menu does for a chosen one. Only that remote is fetched.
func forkFetchRemote(c *ControllerCommon, remote string) error {
	return forkReviewFetchRefspecs(c, []string{remote}, func() error {
		return forkRunFetch(c, func(task gocui.Task) error {
			return c.Git().Sync.FetchRemote(task, remote)
		})
	})
}

// Fork: forkFetchAll fetches every remote, as the palette and the menu do.
func forkFetchAll(c *ControllerCommon) error {
	remotes, err := c.Git().Sync.RemoteNames()
	if err != nil {
		return err
	}
	if len(remotes) == 0 {
		return errors.New(c.Tr.ForkFetchNoRemotes)
	}
	return forkReviewFetchRefspecs(c, remotes, func() error {
		return forkRunFetch(c, c.Git().Sync.FetchAllRemotes)
	})
}

// Fork: forkReviewFetchRefspecs runs fetch once the user has seen any fetch
// refspec of the given remotes that writes into local branches. Without one,
// fetch runs at once. The confirmation's default is to fetch; Cancel and Escape
// stop it. Remotes and automatic fetch use the same check (see
// gui/fork_background_fetch.go for automatic fetch).
func forkReviewFetchRefspecs(c *ControllerCommon, remotes []string, fetch func() error) error {
	risks, err := c.Git().Sync.FetchRefspecRisks(remotes)
	if err != nil {
		return err
	}
	if len(risks) == 0 {
		return fetch()
	}
	c.Confirm(types.ConfirmOpts{
		Title:         c.Tr.ForkFetchRefspecTitle,
		Prompt:        forkFetchRefspecPrompt(c, risks),
		HandleConfirm: fetch,
	})
	return nil
}

func forkFetchRefspecPrompt(c *ControllerCommon, risks []git_commands.FetchRefspecRisk) string {
	lines := lo.Map(risks, func(risk git_commands.FetchRefspecRisk, _ int) string {
		return fmt.Sprintf("- %s: %s", risk.Remote, risk.Refspec)
	})
	return fmt.Sprintf(c.Tr.ForkFetchRefspecPrompt, strings.Join(lines, "\n"))
}

// Fork: forkRunFetch runs a fetch on a worker and then refreshes the branches.
// The credential prompt, the error mapping and PostFetchRefresh are the ones the
// upstream fetch used.
func forkRunFetch(c *ControllerCommon, fetch func(gocui.Task) error) error {
	fetchGeneration := c.State().GetRepoGeneration()
	return c.WithWaitingStatus(c.Tr.FetchingStatus, func(task gocui.Task) error {
		c.LogAction("Fetch")
		err := fetch(task)

		if err != nil && strings.Contains(err.Error(), "exit status 128") {
			return errors.New(c.Tr.PassUnameWrong)
		}

		return c.Helpers().BranchesHelper.PostFetchRefresh(err, false, fetchGeneration)
	})
}
