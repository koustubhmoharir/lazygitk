package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: palette-only command that clears command usage counts (roadmap item 9).
// It has no default key, and it is not counted as a command use (see
// forkUsageRecordable).
func forkResetCommandUsageBinding(c *ControllerCommon) *types.Binding {
	return &types.Binding{
		CommandID:   "app.reset-command-usage",
		Verb:        "reset-usage",
		Description: c.Tr.ResetCommandUsageCounts,
		OpensMenu:   true,
		Handler: func() error {
			c.Confirm(types.ConfirmOpts{
				Title:         c.Tr.ResetCommandUsageCounts,
				Prompt:        c.Tr.ResetCommandUsageCountsPrompt,
				HandleConfirm: c.ResetCommandUsage,
			})
			return nil
		},
	}
}
