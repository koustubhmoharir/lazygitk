package controllers

import (
	"fmt"
	"regexp"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: branch.checkout-previous is a global binding so s p switches to the
// previous checkout from any pane (roadmap item 22). Upstream declared it in
// BranchesController. The handler uses the safe checkout paths, never upstream's
// CheckoutPreviousRef, which offers an autostash.
func forkCheckoutPreviousBinding(c *ControllerCommon, opts types.KeybindingsOpts) *types.Binding {
	return &types.Binding{
		CommandID: "branch.checkout-previous",
		Verb:      "checkout-previous",
		Keys:      opts.GetKeys(opts.Config.Branches.CheckoutPreviousBranch),
		Handler: func() error {
			c.LogAction(c.Tr.Actions.CheckoutBranch)
			return c.Helpers().Refs.CheckoutPreviousFlat()
		},
		DescriptionFunc: func() string {
			if previous := forkPreviousCheckout(c.Model().ReflogCommits); previous != "" {
				return fmt.Sprintf(c.Tr.CheckoutPreviousBranchNamed, previous)
			}
			return c.Tr.CheckoutPreviousBranch
		},
		GetDisabledReason: func() *types.DisabledReason {
			statusFocused := c.Context().Current().GetKey() == context.STATUS_CONTEXT_KEY
			return forkCheckoutPreviousDisabledReason(c.Tr, statusFocused, forkPreviousCheckout(c.Model().ReflogCommits))
		},
	}
}

// forkCheckoutPreviousDisabledReason hides s p in Status, where s is for
// repositories, and otherwise reports a missing previous checkout.
func forkCheckoutPreviousDisabledReason(tr *i18n.TranslationSet, statusFocused bool, previous string) *types.DisabledReason {
	if statusFocused {
		return &types.DisabledReason{Text: tr.NoApplicableCommandsInThisContext, Inapplicable: true}
	}
	if previous == "" {
		return &types.DisabledReason{Text: tr.NoPreviousBranch}
	}
	return nil
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
