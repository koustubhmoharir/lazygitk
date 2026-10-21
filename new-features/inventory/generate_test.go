//go:build ignore

package main

import (
	"go/parser"
	"go/token"
	"testing"
)

func collectSource(t *testing.T, source string) inventory {
	t.Helper()
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "probe.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var inv inventory
	collect(fs, "probe.go", f, &inv)
	return inv
}

func TestCollectTypedBindingLiterals(t *testing.T) {
	inv := collectSource(t, `package probe
import "types"
func build() []*types.Binding {
	items := []*types.Binding{{Keys: keysA, Handler: runA, CommandIDFunc: commandID, VerbFunc: verb}, {Keys: keysB}}
	items = append(items, []*types.Binding{{Keys: keysC}}...)
	items = append(items, &types.Binding{Keys: keysD})
	return items
}`)
	if got, want := len(inv.Factories), 4; got != want {
		t.Fatalf("got %d factories, want %d: %#v", got, want, inv.Factories)
	}
	for _, factory := range inv.Factories {
		if factory.Kind != "Binding" {
			t.Errorf("got kind %q, want Binding", factory.Kind)
		}
	}
	if got, want := inv.Factories[0].Fields["CommandIDFunc"], "commandID"; got != want {
		t.Errorf("CommandIDFunc = %q, want %q", got, want)
	}
	if got, want := inv.Factories[0].Fields["VerbFunc"], "verb"; got != want {
		t.Errorf("VerbFunc = %q, want %q", got, want)
	}
}

func TestCollectNestedMenuLeavesWithoutContainerFalsePositives(t *testing.T) {
	inv := collectSource(t, `package probe
import "types"
type unrelated struct { Label string; Items []*types.MenuItem }
func build() {
	_ = unrelated{Label: "container", Items: []*types.MenuItem{{LabelColumns: []string{"nested", "column"}, OnPress: run}}}
	_ = struct { Value int }{Value: 1}
}`)
	if got, want := len(inv.Factories), 1; got != want {
		t.Fatalf("got %d factories, want %d: %#v", got, want, inv.Factories)
	}
	if got, want := inv.Factories[0].Fields["LabelColumns"], `[]string{"nested", "column"}`; got != want {
		t.Fatalf("got nested menu label columns %q, want %q", got, want)
	}
}

func TestCollectConditionsAndLowLevelCalls(t *testing.T) {
	inv := collectSource(t, `package probe
func build() {
	if ready {
		if permitted {
			gui.SetKeybinding("main", key, handler)
		} else {
			gui.SetEditKeybindings(left, right, backspace, delete)
		}
	}
}`)
	if got, want := len(inv.LowLevelCalls), 2; got != want {
		t.Fatalf("got %d low-level calls, want %d", got, want)
	}
	if got, want := inv.LowLevelCalls[0].Condition, "(ready) && (permitted)"; got != want {
		t.Errorf("then condition = %q, want %q", got, want)
	}
	if got, want := inv.LowLevelCalls[1].Condition, "(ready) && !(permitted)"; got != want {
		t.Errorf("else condition = %q, want %q", got, want)
	}
}

func TestDoesNotClassifyUnrelatedBindingContainer(t *testing.T) {
	inv := collectSource(t, `package probe
type Binding struct { Keys []int }
func build() { _ = Binding{Keys: keys} }`)
	if len(inv.Factories) != 0 {
		t.Fatalf("unrelated container classified as factory: %#v", inv.Factories)
	}
}
