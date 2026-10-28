package controllers

import (
	"fmt"
	"regexp"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: branch.checkout-previous is a global binding so s p switches to the
// previous branch from any pane (roadmap item 22). Upstream declared it in
// BranchesController; the handler is the same RefsHelper.CheckoutPreviousRef.
func forkCheckoutPreviousBinding(c *ControllerCommon, opts types.KeybindingsOpts) *types.Binding {
	return &types.Binding{
		CommandID: "branch.checkout-previous",
		Verb:      "checkout-previous",
		Keys:      opts.GetKeys(opts.Config.Branches.CheckoutPreviousBranch),
		Handler: func() error {
			c.LogAction(c.Tr.Actions.CheckoutBranch)
			return c.Helpers().Refs.CheckoutPreviousRef()
		},
		DescriptionFunc: func() string {
			if previous := forkPreviousCheckout(c.Model().ReflogCommits); previous != "" {
				return fmt.Sprintf(c.Tr.CheckoutPreviousBranchNamed, previous)
			}
			return c.Tr.CheckoutPreviousBranch
		},
		GetDisabledReason: func() *types.DisabledReason {
			if forkPreviousCheckout(c.Model().ReflogCommits) == "" {
				return &types.DisabledReason{Text: c.Tr.NoPreviousBranch}
			}
			return nil
		},
	}
}

var (
	forkCheckoutReflogPattern = regexp.MustCompile(`^checkout: moving from (\S+) to \S+`)
	forkFullHashPattern       = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
)

// forkPreviousCheckout names what `git checkout -` returns to, from the loaded
// reflog (newest first): the source of the latest checkout entry, the same
// entry Git reads for @{-1}. A detached source is shown as a short hash.
func forkPreviousCheckout(reflog []*models.Commit) string {
	for _, commit := range reflog {
		match := forkCheckoutReflogPattern.FindStringSubmatch(commit.Name)
		if match == nil {
			continue
		}
		if forkFullHashPattern.MatchString(match[1]) {
			return utils.ShortHash(match[1])
		}
		return match[1]
	}
	return ""
}
