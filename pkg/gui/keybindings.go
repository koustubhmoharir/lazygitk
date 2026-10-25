package gui

import (
	"errors"
	"log"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (gui *Gui) noPopupPanel(f func() error) func() error {
	return func() error {
		if gui.helpers.Confirmation.IsPopupPanelFocused() {
			return nil
		}

		return f()
	}
}

func (gui *Gui) outsideFilterMode(f func() error) func() error {
	return func() error {
		if !gui.validateNotInFilterMode() {
			return nil
		}

		return f()
	}
}

func (gui *Gui) validateNotInFilterMode() bool {
	if gui.State.Modes.Filtering.Active() {
		gui.c.Confirm(types.ConfirmOpts{
			Title:         gui.c.Tr.MustExitFilterModeTitle,
			Prompt:        gui.c.Tr.MustExitFilterModePrompt,
			HandleConfirm: gui.helpers.Mode.ExitFilterMode,
		})

		return false
	}
	return true
}

// only to be called from the cheatsheet generate script. This mutates the Gui struct.
func (gui *Gui) GetCheatsheetKeybindings() []*types.Binding {
	gui.g = &gocui.Gui{}
	if err := gui.createAllViews(); err != nil {
		panic(err)
	}
	// need to instantiate views
	gui.helpers = helpers.NewStubHelpers()
	gui.State = &GuiRepoState{}
	gui.State.Contexts = gui.contextTree()
	gui.State.ContextMgr = NewContextMgr(gui, gui.State.Contexts)
	// Fork command projection needs a canonical focused source even offline.
	gui.State.ContextMgr.ContextStack = []types.Context{gui.State.Contexts.Files}
	gui.resetHelpersAndControllers()
	bindings, _ := gui.GetInitialKeybindings()
	return bindings
}

func (gui *Gui) keybindingOpts() types.KeybindingsOpts {
	keybindingConfig := gui.c.UserConfig().Keybinding

	guards := types.KeybindingGuards{
		OutsideFilterMode: gui.outsideFilterMode,
		NoPopupPanel:      gui.noPopupPanel,
	}

	return types.KeybindingsOpts{
		GetKeys: config.GetValidatedKeyBindingKeys,
		Config:  keybindingConfig,
		Guards:  guards,
	}
}

func (gui *Gui) GetInitialKeybindings() ([]*types.Binding, []*gocui.ViewMouseBinding) {
	opts := gui.c.KeybindingsOpts()

	bindings := []*types.Binding{
		{
			ViewName:    "",
			Keys:        opts.GetKeys(opts.Config.Universal.OpenRecentRepos),
			CommandID:   "repo.switch-menu",
			Verb:        "switch-repository",
			Handler:     opts.Guards.NoPopupPanel(gui.helpers.Repos.CreateRecentReposMenu),
			Description: gui.c.Tr.SwitchRepo,
		},
		{
			// Fork: `s n` opens a repository by path (see fork_open_repo_path.go).
			ViewName:    "",
			CommandID:   "repo.open-path",
			Verb:        "switch-repository",
			Handler:     opts.Guards.NoPopupPanel(gui.helpers.Repos.OpenRepoPathPrompt),
			Description: gui.c.Tr.OpenRepoByPath,
		},
		{
			ViewName:    "",
			Keys:        opts.GetKeys(opts.Config.Universal.ScrollUpMain),
			CommandID:   "navigation.main.scroll-up",
			Verb:        "scroll-up",
			Handler:     gui.scrollUpMain,
			Alternative: "fn+up/shift+k",
			Description: gui.c.Tr.ScrollUpMainWindow,
		},
		{
			ViewName:    "",
			Keys:        opts.GetKeys(opts.Config.Universal.ScrollDownMain),
			CommandID:   "navigation.main.scroll-down",
			Verb:        "scroll-down",
			Handler:     gui.scrollDownMain,
			Alternative: "fn+down/shift+j",
			Description: gui.c.Tr.ScrollDownMainWindow,
		},
		{
			ViewName:          "files",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-file-path",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyPathToClipboard,
		},
		{
			ViewName:          "localBranches",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-branch-name",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyBranchNameToClipboard,
		},
		{
			ViewName:          "remoteBranches",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-branch-name",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyBranchNameToClipboard,
		},
		{
			ViewName:          "tags",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-tag-name",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyTagToClipboard,
		},
		// Fork: this handler truncates to TruncateCopiedCommitHashesTo; the
		// full hash has its own command (clipboard.copy-commit-hash).
		{
			ViewName:          "commits",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-abbreviated-commit-hash",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemCommitHashToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyAbbreviatedCommitHash,
		},
		{
			ViewName:    "commits",
			Keys:        opts.GetKeys(opts.Config.Commits.ResetCherryPick),
			CommandID:   "commit.clear-cherry-pick-selection",
			Verb:        "clear-cherry-pick-selection",
			Handler:     gui.helpers.CherryPick.Reset,
			Description: gui.c.Tr.ResetCherryPick,
		},
		{
			ViewName:          "reflogCommits",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-commit-hash",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyCommitHash,
		},
		// Fork: this handler truncates to TruncateCopiedCommitHashesTo; the
		// full hash has its own command (clipboard.copy-commit-hash).
		{
			ViewName:          "subCommits",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-abbreviated-commit-hash",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemCommitHashToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyAbbreviatedCommitHash,
		},
		{
			ViewName:  "information",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseLeft)},
			CommandID: "ui.information.activate",
			Verb:      "activate-link",
			Handler:   gui.handleInfoClick,
		},
		{
			ViewName:          "commitFiles",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-file-path",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopyPathToClipboard,
		},
		{
			ViewName:    "",
			Keys:        opts.GetKeys(opts.Config.Universal.ExtrasMenu),
			CommandID:   "command-log.options-menu",
			Verb:        "log-options",
			Handler:     opts.Guards.NoPopupPanel(gui.handleCreateExtrasMenuPanel),
			Description: gui.c.Tr.OpenCommandLogMenu,
			Tooltip:     gui.c.Tr.OpenCommandLogMenuTooltip,
			OpensMenu:   true,
		},
		{
			ViewName:    "main",
			Keys:        []gocui.Key{gocui.NewKeyName(gocui.MouseWheelDown)},
			CommandID:   "navigation.main.scroll-down",
			Verb:        "scroll-down",
			Handler:     gui.scrollDownMain,
			Description: gui.c.Tr.ScrollDown,
			Alternative: "fn+up",
		},
		{
			ViewName:    "main",
			Keys:        []gocui.Key{gocui.NewKeyName(gocui.MouseWheelUp)},
			CommandID:   "navigation.main.scroll-up",
			Verb:        "scroll-up",
			Handler:     gui.scrollUpMain,
			Description: gui.c.Tr.ScrollUp,
			Alternative: "fn+down",
		},
		{
			ViewName:  "secondary",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseWheelDown)},
			CommandID: "navigation.secondary.scroll-down",
			Verb:      "scroll-down",
			Handler:   gui.scrollDownSecondary,
		},
		{
			ViewName:  "secondary",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseWheelUp)},
			CommandID: "navigation.secondary.scroll-up",
			Verb:      "scroll-up",
			Handler:   gui.scrollUpSecondary,
		},
		{
			ViewName:  "confirmation",
			Keys:      opts.GetKeys(opts.Config.Universal.PrevItem),
			CommandID: "navigation.confirmation.scroll-up",
			Verb:      "scroll-up",
			Handler:   gui.scrollUpConfirmationPanel,
		},
		{
			ViewName:  "confirmation",
			Keys:      opts.GetKeys(opts.Config.Universal.NextItem),
			CommandID: "navigation.confirmation.scroll-down",
			Verb:      "scroll-down",
			Handler:   gui.scrollDownConfirmationPanel,
		},
		{
			ViewName:  "confirmation",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseWheelUp)},
			CommandID: "navigation.confirmation.scroll-up",
			Verb:      "scroll-up",
			Handler:   gui.scrollUpConfirmationPanel,
		},
		{
			ViewName:  "confirmation",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseWheelDown)},
			CommandID: "navigation.confirmation.scroll-down",
			Verb:      "scroll-down",
			Handler:   gui.scrollDownConfirmationPanel,
		},
		{
			ViewName:  "confirmation",
			Keys:      opts.GetKeys(opts.Config.Universal.NextPage),
			CommandID: "navigation.confirmation.page-down",
			Verb:      "next-page",
			Handler:   gui.pageDownConfirmationPanel,
		},
		{
			ViewName:  "confirmation",
			Keys:      opts.GetKeys(opts.Config.Universal.PrevPage),
			CommandID: "navigation.confirmation.page-up",
			Verb:      "previous-page",
			Handler:   gui.pageUpConfirmationPanel,
		},
		{
			ViewName:  "confirmation",
			Keys:      opts.GetKeys(opts.Config.Universal.GotoTop),
			CommandID: "navigation.confirmation.first",
			Verb:      "first-item",
			Handler:   gui.goToConfirmationPanelTop,
		},
		{
			ViewName:  "confirmation",
			Keys:      opts.GetKeys(opts.Config.Universal.GotoBottom),
			CommandID: "navigation.confirmation.last",
			Verb:      "last-item",
			Handler:   gui.goToConfirmationPanelBottom,
		},
		{
			ViewName:          "submodules",
			Keys:              opts.GetKeys(opts.Config.Universal.CopyToClipboard),
			CommandID:         "clipboard.copy-submodule-name",
			Verb:              "copy",
			Handler:           gui.handleCopySelectedSideContextItemToClipboard,
			GetDisabledReason: gui.getCopySelectedSideContextItemToClipboardDisabledReason,
			Description:       gui.c.Tr.CopySubmoduleNameToClipboard,
		},
		{
			ViewName:  "extras",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseWheelUp)},
			CommandID: "navigation.command-log.scroll-up",
			Verb:      "scroll-up",
			Handler:   gui.scrollUpExtra,
		},
		{
			ViewName:  "extras",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseWheelDown)},
			CommandID: "navigation.command-log.scroll-down",
			Verb:      "scroll-down",
			Handler:   gui.scrollDownExtra,
		},
		{
			ViewName:  "extras",
			Tag:       "navigation",
			Keys:      opts.GetKeys(opts.Config.Universal.PrevItem),
			CommandID: "navigation.command-log.scroll-up",
			Verb:      "scroll-up",
			Handler:   gui.scrollUpExtra,
		},
		{
			ViewName:  "extras",
			Tag:       "navigation",
			Keys:      opts.GetKeys(opts.Config.Universal.NextItem),
			CommandID: "navigation.command-log.scroll-down",
			Verb:      "scroll-down",
			Handler:   gui.scrollDownExtra,
		},
		{
			ViewName:  "extras",
			Keys:      opts.GetKeys(opts.Config.Universal.NextPage),
			CommandID: "navigation.command-log.page-down",
			Verb:      "next-page",
			Handler:   gui.pageDownExtrasPanel,
		},
		{
			ViewName:  "extras",
			Keys:      opts.GetKeys(opts.Config.Universal.PrevPage),
			CommandID: "navigation.command-log.page-up",
			Verb:      "previous-page",
			Handler:   gui.pageUpExtrasPanel,
		},
		{
			ViewName:  "extras",
			Keys:      opts.GetKeys(opts.Config.Universal.GotoTop),
			CommandID: "navigation.command-log.first",
			Verb:      "first-item",
			Handler:   gui.goToExtrasPanelTop,
		},
		{
			ViewName:  "extras",
			Keys:      opts.GetKeys(opts.Config.Universal.GotoBottom),
			CommandID: "navigation.command-log.last",
			Verb:      "last-item",
			Handler:   gui.goToExtrasPanelBottom,
		},
		{
			ViewName:  "extras",
			Tag:       "navigation",
			Keys:      []gocui.Key{gocui.NewKeyName(gocui.MouseLeft)},
			CommandID: "navigation.focus-command-log",
			Verb:      "focus-command-log",
			Handler:   gui.handleFocusCommandLog,
		},
	}

	contexts := gui.State.Contexts.Flatten()
	mouseKeybindings := make([]*gocui.ViewMouseBinding, 0, len(contexts))
	for _, c := range contexts {
		viewName := c.GetViewName()
		for _, binding := range c.GetKeybindings(opts) {
			// TODO: move all mouse keybindings into the mouse keybindings approach below
			binding.ViewName = viewName
			bindings = append(bindings, binding)
		}

		mouseKeybindings = append(mouseKeybindings, c.GetMouseKeybindings(opts)...)
	}

	bindings = append(bindings, []*types.Binding{
		{
			ViewName:    "",
			Keys:        opts.GetKeys(opts.Config.Universal.NextTab),
			CommandID:   "navigation.tab.next",
			Verb:        "next-tab",
			Handler:     opts.Guards.NoPopupPanel(gui.handleNextTab),
			Description: gui.c.Tr.NextTab,
			Tag:         "navigation",
		},
		{
			ViewName:    "",
			Keys:        opts.GetKeys(opts.Config.Universal.PrevTab),
			CommandID:   "navigation.tab.previous",
			Verb:        "previous-tab",
			Handler:     opts.Guards.NoPopupPanel(gui.handlePrevTab),
			Description: gui.c.Tr.PrevTab,
			Tag:         "navigation",
		},
	}...)

	// Fork: project semantic defaults once at the source shared by dispatch,
	// palettes and generated cheatsheets, replacing upstream's pane-key table.
	projected, err := gui.projectForkBindings(bindings)
	if err != nil {
		panic(err) // User configuration paths have already been validated.
	}
	return projected, mouseKeybindings
}

func (gui *Gui) GetInitialKeybindingsWithCustomCommands() ([]*types.Binding, []*gocui.ViewMouseBinding) {
	bindings, custom, mouse := gui.initialBindingsWithCustomCommands()
	all := append(custom, bindings...)
	view := gui.State.ContextMgr.CurrentStatic().GetViewName()
	if forkCommandView(view) {
		all = effectiveForkDisplayBindings(all, view)
	}
	return all, mouse
}

func (gui *Gui) initialBindingsWithCustomCommands() ([]*types.Binding, []*types.Binding, []*gocui.ViewMouseBinding) {
	// if the search or filter prompt is open, we only want the keybindings for
	// that context. It shouldn't be possible, for example, to open a menu while
	// the prompt is showing; you first need to confirm or cancel the search/filter.
	if currentContext := gui.State.ContextMgr.Current(); currentContext.GetKey() == context.SEARCH_CONTEXT_KEY {
		bindings := currentContext.GetKeybindings(gui.c.KeybindingsOpts())
		viewName := currentContext.GetViewName()
		for _, binding := range bindings {
			binding.ViewName = viewName
		}
		return bindings, nil, nil
	}

	bindings, mouseBindings := gui.GetInitialKeybindings()
	customBindings, err := gui.CustomCommandsClient.GetCustomCommandKeybindings()
	if err != nil {
		log.Fatal(err)
	}
	return bindings, customBindings, mouseBindings
}

func (gui *Gui) resetKeybindings() {
	gui.g.DeleteAllKeybindings()

	bindings, customBindings, mouseBindings := gui.initialBindingsWithCustomCommands()
	// Fork: normal panes use compiled command paths and no-op reservations;
	// custom, modal, editor and mouse routes retain native registration.
	gui.registerForkCommands(bindings, customBindings)
	// Fork: command key labels follow the same bindings as the registrations
	// above, except while the search prompt narrows them to its own controls.
	if gui.State.ContextMgr.Current().GetKey() != context.SEARCH_CONTEXT_KEY {
		gui.setForkCommandBindings(bindings, customBindings)
	}
	gui.updateForkPanelJumpLabels(bindings, customBindings)

	for _, binding := range mouseBindings {
		gui.SetMouseKeybinding(binding)
	}

	for _, values := range gui.viewTabMap() {
		for _, value := range values {
			viewName := value.ViewName
			tabClickCallback := func(tabIndex int) error {
				return gui.onViewTabClick(gui.helpers.Window.WindowForView(viewName), tabIndex)
			}

			gui.g.SetTabClickBinding(viewName, tabClickCallback)
		}
	}
}

func (gui *Gui) SetKeybinding(binding *types.Binding) {
	handler := func(g *gocui.Gui, v *gocui.View) error {
		return gui.callKeybindingHandler(binding)
	}

	for _, key := range binding.Keys {
		if binding.DefaultCommand && !forkMouseKey(key) {
			gui.g.SetCommandKeybinding(binding.ViewName, key, handler)
		} else {
			gui.g.SetKeybinding(binding.ViewName, key, handler)
		}
	}
}

func (gui *Gui) SetMouseKeybinding(binding *gocui.ViewMouseBinding) {
	gui.g.SetViewClickBinding(binding)
}

// Fork: shortcuts and prefix-menu leaves call this, so their uses subtract from
// the command's usage count (see fork_command_usage.go).
func (gui *Gui) callKeybindingHandler(binding *types.Binding) error {
	return gui.callKeybindingHandlerWithUsage(binding, forkShortcutUsageDelta)
}

// Fork: delta is the command-usage change recorded if the handler is accepted.
func (gui *Gui) callKeybindingHandlerWithUsage(binding *types.Binding, delta int) error {
	if binding.CommandActive != nil && !binding.CommandActive() {
		return nil
	}
	if binding.GetDisabledReason != nil {
		if disabledReason := binding.GetDisabledReason(); disabledReason != nil {
			// Fork defaults consume inapplicable actions instead of falling
			// through to a different pane/global meaning. Explicit overrides
			// retain upstream disabled/error dispatch behavior.
			if binding.DefaultCommand && (disabledReason.Inapplicable || disabledReason.AllowFurtherDispatching) {
				return nil
			}
			if disabledReason.AllowFurtherDispatching {
				return &types.ErrKeybindingNotHandled{DisabledReason: disabledReason}
			}

			if disabledReason.ShowErrorInPanel {
				return errors.New(disabledReason.Text)
			}

			if len(disabledReason.Text) > 0 {
				gui.c.ErrorToast(gui.Tr.DisabledMenuItemPrefix + disabledReason.Text)
			}
			return nil
		}
	}

	// Fork: the use stays pending while the handler runs, and is committed
	// unless a popup it opens claims it (see forkUsageTracker).
	pending := gui.forkUsage.begin(binding, delta)
	err := binding.Handler()
	gui.forkUsage.end(pending)
	return err
}
