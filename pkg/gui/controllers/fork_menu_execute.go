package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the keys that run a menu's selected item. Upstream also runs it on the
// universal select key (Space), which is not the fork's way to act on an item:
// Enter is. So a regular menu keeps Space for nothing, unless the user has set
// universal.select explicitly. Filtering menus keep the key, where it types into
// the filter (see MenuContext.GetKeybindings). Prefix menus block Space themselves
// (forkPrefixBlockedKeys), so they need no key here. The caller passes the select
// field itself, so the legacy-field generator still maps it to menu.execute.
func forkMenuExecuteKeys(c *ControllerCommon, opts types.KeybindingsOpts, selectKey config.Keybinding, filtering bool) []gocui.Key {
	if filtering || c.UserConfig().KeybindingIsExplicit("universal.select") {
		return opts.GetKeys(selectKey)
	}
	return nil
}
