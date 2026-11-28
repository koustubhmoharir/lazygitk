package context

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils/promptmark"
	"github.com/samber/lo"
)

// Fork: the prefix-menu parts of MenuContext.GetKeybindings. They were moved
// here out of menu_context.go to ease upstream merges.

// Fork prefixes reserve physical navigation; configured letter aliases must not
// consume a continuation before its command accelerator.
func (self *MenuContext) applyForkPrefixKeys(opts types.KeybindingsOpts) types.KeybindingsOpts {
	if len(self.blockUnmatchedKeys) > 0 {
		opts.Config.Universal.PrevItem = []string{"<up>"}
		opts.Config.Universal.NextItem = []string{"<down>"}
		opts.Config.Universal.ConfirmMenu = []string{"<enter>"}
		opts.Config.Universal.Return = []string{"<esc>"}
	}
	return opts
}

// A no-op default-command binding for each key that a prefix menu blocks.
func (self *MenuContext) forkBlockedKeyBindings() []*types.Binding {
	return lo.Map(self.blockUnmatchedKeys, func(key gocui.Key, _ int) *types.Binding {
		return &types.Binding{Keys: []gocui.Key{key}, DefaultCommand: true, Handler: func() error { return nil }}
	})
}

// Fork: the arrow column that points from an item's key to its label. Items
// without a key get a blank cell, and a menu where no item has a key has no
// visible column at all (RenderDisplayStrings drops blank columns).
func forkKeyArrow(hasKey bool) string {
	if !hasKey {
		return ""
	}
	return style.FgCyan.Sprint("→")
}

// Fork: the label columns of a menu item as they are displayed. Role markers in
// a label (see pkg/utils/promptmark) resolve to the prompt styles. The first
// column of a disabled item is styled as a whole (struck through or dimmed), so
// its markers are stripped instead, which keeps inner colours from resetting
// that style.
func forkMenuDisplayColumns(item *types.MenuItem) []string {
	columns := make([]string, len(item.LabelColumns))
	for i, column := range item.LabelColumns {
		if i == 0 && item.DisabledReason != nil {
			columns[i] = promptmark.Strip(column)
		} else {
			columns[i] = theme.ResolvePromptMarks(column)
		}
	}
	return columns
}

// Fork: the plain text of label columns, for matching a filter against. Role
// markers are not text, so a filter matches what the user reads.
func forkPlainColumns(columns []string) []string {
	return lo.Map(columns, func(column string, _ int) string {
		return promptmark.Strip(column)
	})
}
