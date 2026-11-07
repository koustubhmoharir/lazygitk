package context

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
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
