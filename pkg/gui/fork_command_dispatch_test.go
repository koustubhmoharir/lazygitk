package gui

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type forkDispatchContext struct {
	types.Context
	name string
	kind types.ContextKind
}

func (c *forkDispatchContext) GetViewName() string        { return c.name }
func (c *forkDispatchContext) GetKind() types.ContextKind { return c.kind }
func (c *forkDispatchContext) GetKey() types.ContextKey   { return types.ContextKey(c.name) }

func TestForkCommandDispatch(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	oldScreen := gocui.Screen
	var screenRestored bool
	t.Cleanup(func() {
		if screenRestored {
			gocui.Screen = oldScreen
		}
	})

	type result struct {
		err     error
		actions []string
	}
	done := make(chan result, 1)
	headless := make(chan func(), 1)
	go func() {
		heavy, err := gocui.NewGui(gocui.NewGuiOpts{Headless: true, Width: 80, Height: 24, PlayRecording: true})
		if err != nil {
			done <- result{err: err}
			return
		}
		var closeOnce sync.Once
		closeGui := func() { closeOnce.Do(heavy.Close) }
		out := result{}
		defer func() {
			closeGui()
			done <- out
		}()
		headless <- closeGui
		gui.g = heavy
		filesCtx := &forkDispatchContext{name: "files", kind: types.SIDE_CONTEXT}
		gui.State = &GuiRepoState{ContextMgr: NewContextMgr(gui, nil), Contexts: &context.ContextTree{}}
		gui.State.ContextMgr.ContextStack = []types.Context{filesCtx}
		for _, view := range []struct {
			name   string
			x0, x1 int
		}{{"files", 0, 39}, {"localBranches", 40, 79}, {"main", 0, 79}, {"menu", 0, 79}} {
			if _, err := heavy.SetView(view.name, view.x0, 0, view.x1, 23, 0); err != nil && !errors.Is(err, gocui.ErrUnknownView) {
				out.err = err
				return
			}
		}
		if _, err := heavy.SetCurrentView("files"); err != nil {
			out.err = err
			return
		}

		var actions []string
		var bindings []*types.Binding
		add := func(id, view string, key gocui.Key, disabled func() *types.DisabledReason) {
			bindings = append(bindings, &types.Binding{
				CommandID: id, ViewName: view, Keys: []gocui.Key{key}, DefaultCommand: true,
				GetDisabledReason: disabled,
				Handler:           func() error { actions = append(actions, id); return nil },
			})
		}
		add("files.uppercase", "files", gocui.NewKeyRune('x'), nil)
		add("branches.only", "localBranches", gocui.NewKeyRune('b'), nil)
		add("files.reserve-b", "files", gocui.NewKeyRune('b'), func() *types.DisabledReason {
			return &types.DisabledReason{Text: "wrong pane", Inapplicable: true}
		})
		add("global.branch-fallback", "", gocui.NewKeyRune('b'), nil)
		add("files.local-letter", "files", gocui.NewKeyRune('z'), nil)
		add("files.local", "files", gocui.NewKeyRune('l'), nil)
		add("files.reserve-v", "files", gocui.NewKeyRune('v'), func() *types.DisabledReason {
			return &types.DisabledReason{Text: "wrong pane", Inapplicable: true}
		})
		add("global.default-fallback", "", gocui.NewKeyRune('v'), nil)
		add("files.mode-inactive", "files", gocui.NewKeyRune('m'), func() *types.DisabledReason {
			return &types.DisabledReason{Text: "wrong mode", Inapplicable: true}
		})
		add("files.mode-active", "files", gocui.NewKeyRune('m'), nil)
		add("files.invalid-mode", "files", gocui.NewKeyRune('i'), func() *types.DisabledReason {
			return &types.DisabledReason{Text: "wrong mode", Inapplicable: true}
		})
		add("global.invalid-mode-fallback", "", gocui.NewKeyRune('i'), nil)
		// A local default that declines must hand the key to the global owner.
		bindings = append(bindings, &types.Binding{
			CommandID: "files.decline-e", ViewName: "files", Keys: []gocui.Key{gocui.NewKeyRune('e')}, DefaultCommand: true,
			Handler: func() error { return gocui.ErrKeybindingNotHandled },
		})
		add("global.e-fallback", "", gocui.NewKeyRune('e'), nil)
		// Global single keys must reach popups natively, but never a fork view.
		add("global.popup-key", "", gocui.NewKeyRune('g'), nil)
		custom := []*types.Binding{
			{CommandID: "custom.ctrl", Keys: []gocui.Key{gocui.NewKeyStrMod("k", gocui.ModCtrl)}, Handler: func() error { actions = append(actions, "custom.ctrl"); return nil }},
			{CommandID: "custom.global-letter", Keys: []gocui.Key{gocui.NewKeyRune('z')}, Handler: func() error { actions = append(actions, "custom.global-letter"); return nil }},
		}
		gui.registerForkCommands(bindings, custom)
		setPane := func(view string, paneCtx types.Context) error {
			if _, err := heavy.SetCurrentView(view); err != nil {
				return err
			}
			gui.State.ContextMgr.ContextStack = []types.Context{paneCtx}
			return nil
		}
		branchesCtx := &forkDispatchContext{name: "localBranches", kind: types.SIDE_CONTEXT}
		heavy.SetKeybinding("", gocui.NewKeyName(gocui.KeyF10), func(*gocui.Gui, *gocui.View) error {
			return setPane("localBranches", branchesCtx)
		})
		heavy.SetKeybinding("", gocui.NewKeyName(gocui.KeyF9), func(*gocui.Gui, *gocui.View) error {
			return setPane("files", filesCtx)
		})
		menuCtx := &forkDispatchContext{name: "menu", kind: types.TEMPORARY_POPUP}
		heavy.SetKeybinding("", gocui.NewKeyName(gocui.KeyF8), func(*gocui.Gui, *gocui.View) error {
			return setPane("menu", menuCtx)
		})
		mainCtx := &forkDispatchContext{name: "main", kind: types.MAIN_CONTEXT}
		heavy.SetKeybinding("", gocui.NewKeyName(gocui.KeyF7), func(*gocui.Gui, *gocui.View) error {
			return setPane("main", mainCtx)
		})

		quitKey := gocui.NewKeyName(gocui.KeyF12)
		heavy.SetKeybinding("", quitKey, func(*gocui.Gui, *gocui.View) error { return gocui.ErrQuit })
		events := []struct {
			key tcell.Key
			ch  rune
			mod tcell.ModMask
		}{
			{key: tcell.KeyRune, ch: 'X'},
			{key: tcell.KeyRune, ch: 'x'},
			{key: tcell.KeyRune, ch: 'b'},
			{key: tcell.KeyRune, ch: 'v'},
			{key: tcell.KeyRune, ch: 'z'},
			{key: tcell.KeyRune, ch: 'm'},
			{key: tcell.KeyRune, ch: 'i'},
			{key: tcell.KeyRune, ch: 'k', mod: tcell.ModCtrl},
			{key: tcell.KeyF10},
			{key: tcell.KeyRune, ch: 'b'},
			{key: tcell.KeyF9},
			{key: tcell.KeyRune, ch: 'l'},
			{key: tcell.KeyRune, ch: 'e'},
			{key: tcell.KeyF8},
			{key: tcell.KeyRune, ch: 'g'},
			{key: tcell.KeyF7},
			{key: tcell.KeyRune, ch: 'g'},
			{key: tcell.KeyF12},
		}
		enqueued := make(chan struct{})
		go func() {
			for _, event := range events {
				ch := ""
				if event.ch != 0 {
					ch = string(event.ch)
				}
				heavy.ReplayKeyEvent(gocui.NewTcellKeyEventWrapper(tcell.NewEventKey(event.key, ch, event.mod)))
			}
			close(enqueued)
		}()
		loopErr := heavy.MainLoop()
		<-enqueued
		out = result{err: loopErr, actions: actions}
	}()
	select {
	case result := <-done:
		screenRestored = true
		gocui.Screen = oldScreen
		if !errors.Is(result.err, gocui.ErrQuit) {
			t.Fatalf("MainLoop result: %v, actions: %v", result.err, result.actions)
		}
		want := []string{
			"files.uppercase", "files.uppercase", "custom.global-letter", "files.mode-active",
			"custom.ctrl", "branches.only", "files.local", "global.e-fallback", "global.popup-key",
		}
		if len(result.actions) != len(want) {
			t.Fatalf("actions = %v, want %v", result.actions, want)
		}
		for i := range want {
			if result.actions[i] != want[i] {
				t.Fatalf("actions = %v, want %v", result.actions, want)
			}
		}
	case <-time.After(5 * time.Second):
		select {
		case closeGui := <-headless:
			closeGui()
		case <-time.After(time.Second):
		}
		select {
		case <-done:
			screenRestored = true
			gocui.Screen = oldScreen
		case <-time.After(time.Second):
			t.Fatal("headless dispatcher did not stop after Close; leaving its screen installed")
		}
		t.Fatal("headless dispatcher did not reach quit sentinel")
	}
}
