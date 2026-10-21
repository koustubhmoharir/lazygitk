package custom_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type identityTestContext struct {
	types.Context
	key      types.ContextKey
	viewName string
}

func (c identityTestContext) GetKey() types.ContextKey { return c.key }
func (c identityTestContext) GetViewName() string      { return c.viewName }

type identityTestContexts struct{ tree *context.ContextTree }

func (c identityTestContexts) Contexts() *context.ContextTree { return c.tree }

type identityTestGui struct {
	types.IGuiCommon
	menuOptions []types.CreateMenuOptions
}

func (g *identityTestGui) Menu(opts types.CreateMenuOptions) error {
	g.menuOptions = append(g.menuOptions, opts)
	return nil
}

func identityTestHelperCommon(tree *context.ContextTree, gui types.IGuiCommon) *helpers.HelperCommon {
	return &helpers.HelperCommon{
		Common:       common.NewDummyCommon(),
		IGuiCommon:   gui,
		IGetContexts: identityTestContexts{tree: tree},
	}
}

func TestKeybindingCreatorPropagatesCommandIdentityAcrossContexts(t *testing.T) {
	tree := &context.ContextTree{
		Global: identityTestContext{key: context.FILES_CONTEXT_KEY, viewName: "files-view"},
		Status: identityTestContext{key: context.LOCAL_COMMITS_CONTEXT_KEY, viewName: "commits-view"},
	}
	creator := NewKeybindingCreator(identityTestHelperCommon(tree, &identityTestGui{}))

	for _, tc := range []struct {
		name string
		cmd  config.CustomCommand
	}{
		{name: "explicit", cmd: config.CustomCommand{ID: "custom.release", Context: "files,commits", Key: config.Keybinding{"x", "y"}, Command: "echo hi"}},
		{name: "fallback", cmd: config.CustomCommand{Context: "files,commits", Key: config.Keybinding{"x", "y"}, Command: "echo hi"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			bindings, err := creator.call(tc.cmd, func() error { calls++; return nil })
			if err != nil {
				t.Fatal(err)
			}
			if len(bindings) != 2 {
				t.Fatalf("got %d bindings, want one for each context", len(bindings))
			}
			for i, binding := range bindings {
				if binding.CommandID != tc.cmd.CommandIdentity() || binding.Verb != "run-custom-command" {
					t.Errorf("identity = (%q, %q), want (%q, run-custom-command)", binding.CommandID, binding.Verb, tc.cmd.CommandIdentity())
				}
				wantView := []string{"files-view", "commits-view"}[i]
				if binding.ViewName != wantView || binding.Description != tc.cmd.GetDescription() {
					t.Errorf("binding compatibility fields = view %q, description %q", binding.ViewName, binding.Description)
				}
				if len(binding.Keys) != 2 || binding.Handler == nil {
					t.Fatalf("keys/handler not preserved: keys=%v handler nil=%t", binding.Keys, binding.Handler == nil)
				}
				if err := binding.Handler(); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 2 {
				t.Errorf("alternate context handlers invoked %d times, want 2", calls)
			}
		})
	}
}

func TestCustomMenuPropagatesCommandIdentity(t *testing.T) {
	gui := &identityTestGui{}
	root := config.CustomCommand{
		ID: "custom.root", Key: config.Keybinding{"m"}, Description: "Root menu",
		CommandMenu: []config.CustomCommand{{
			ID: "custom.nested", Key: config.Keybinding{"n"}, Description: "Nested menu",
			CommandMenu: []config.CustomCommand{{ID: "custom.leaf", Key: config.Keybinding{"l", "k"}, Context: "global", Command: "echo leaf", Description: "Leaf"}},
		}},
	}
	common := identityTestHelperCommon(&context.ContextTree{}, gui)
	common.UserConfig().CustomCommands = []config.CustomCommand{root}
	client := &Client{
		c:                 common,
		keybindingCreator: NewKeybindingCreator(common),
		handlerCreator:    &HandlerCreator{},
	}

	bindings, err := client.GetCustomCommandKeybindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 {
		t.Fatalf("got %d openers, want 1", len(bindings))
	}
	rootBinding := bindings[0]
	if rootBinding.CommandID != root.CommandIdentity() || rootBinding.Verb != "open-custom-menu" || !rootBinding.OpensMenu {
		t.Errorf("root opener identity/behavior = (%q, %q, %t)", rootBinding.CommandID, rootBinding.Verb, rootBinding.OpensMenu)
	}
	if len(rootBinding.Keys) != 1 || rootBinding.Handler == nil {
		t.Errorf("root opener lost shortcut or handler: keys=%v handler nil=%t", rootBinding.Keys, rootBinding.Handler == nil)
	}
	if err := rootBinding.Handler(); err != nil {
		t.Fatal(err)
	}
	rootItems := gui.menuOptions[len(gui.menuOptions)-1].Items
	if len(rootItems) != 1 {
		t.Fatalf("root menu has %d items, want 1", len(rootItems))
	}
	nested := rootItems[0]
	if nested.CommandID != root.CommandMenu[0].CommandIdentity() || nested.Verb != "open-custom-menu" || !nested.OpensMenu {
		t.Errorf("nested opener identity/behavior = (%q, %q, %t)", nested.CommandID, nested.Verb, nested.OpensMenu)
	}
	if len(nested.Keys) != 1 || nested.OnPress == nil {
		t.Errorf("nested opener lost shortcut or handler: keys=%v handler nil=%t", nested.Keys, nested.OnPress == nil)
	}
	if err := nested.OnPress(); err != nil {
		t.Fatal(err)
	}
	leaf := gui.menuOptions[len(gui.menuOptions)-1].Items[0]
	if leaf.CommandID != root.CommandMenu[0].CommandMenu[0].CommandIdentity() || leaf.Verb != "run-custom-command" || leaf.OpensMenu {
		t.Errorf("leaf identity/behavior = (%q, %q, %t)", leaf.CommandID, leaf.Verb, leaf.OpensMenu)
	}
	if len(leaf.Keys) != 2 || leaf.OnPress == nil {
		t.Errorf("leaf lost shortcut or handler: keys=%v handler nil=%t", leaf.Keys, leaf.OnPress == nil)
	}
}
