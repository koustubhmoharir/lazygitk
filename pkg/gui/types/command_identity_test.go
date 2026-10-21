package types

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
)

func TestBindingCommandIdentity(t *testing.T) {
	binding := &Binding{CommandID: "static.id", Verb: "static-verb"}
	if got := binding.GetCommandID(); got != "static.id" {
		t.Fatalf("GetCommandID() = %q, want %q", got, "static.id")
	}
	if got := binding.GetVerb(); got != "static-verb" {
		t.Fatalf("GetVerb() = %q, want %q", got, "static-verb")
	}

	state := "first"
	binding.CommandIDFunc = func() string { return "dynamic." + state }
	binding.VerbFunc = func() string { return state }
	binding.Keys = []gocui.Key{gocui.NewKeyRune('a')}
	binding.Description = "first description"

	if got := binding.GetCommandID(); got != "dynamic.first" {
		t.Fatalf("GetCommandID() = %q, want %q", got, "dynamic.first")
	}
	if got := binding.GetVerb(); got != "first" {
		t.Fatalf("GetVerb() = %q, want %q", got, "first")
	}

	state = "second"
	binding.Keys = []gocui.Key{gocui.NewKeyRune('b')}
	binding.Description = "second description"
	if got := binding.GetCommandID(); got != "dynamic.second" {
		t.Fatalf("GetCommandID() after state change = %q, want %q", got, "dynamic.second")
	}
	if got := binding.GetVerb(); got != "second" {
		t.Fatalf("GetVerb() after state change = %q, want %q", got, "second")
	}
}

func TestMenuItemCommandIdentity(t *testing.T) {
	item := &MenuItem{CommandID: "static.id", Verb: "static-verb", Label: "first label"}
	if got := item.GetCommandID(); got != "static.id" {
		t.Fatalf("GetCommandID() = %q, want %q", got, "static.id")
	}
	if got := item.GetVerb(); got != "static-verb" {
		t.Fatalf("GetVerb() = %q, want %q", got, "static-verb")
	}

	state := "first"
	item.CommandIDFunc = func() string { return "dynamic." + state }
	item.VerbFunc = func() string { return state }
	item.Keys = []gocui.Key{gocui.NewKeyRune('a')}

	if got := item.GetCommandID(); got != "dynamic.first" {
		t.Fatalf("GetCommandID() = %q, want %q", got, "dynamic.first")
	}
	if got := item.GetVerb(); got != "first" {
		t.Fatalf("GetVerb() = %q, want %q", got, "first")
	}
	if got := item.ID(); got != "first label" {
		t.Fatalf("ID() = %q, want row label %q", got, "first label")
	}

	state = "second"
	item.Keys = []gocui.Key{gocui.NewKeyRune('b')}
	item.Label = "second label"
	if got := item.GetCommandID(); got != "dynamic.second" {
		t.Fatalf("GetCommandID() after state change = %q, want %q", got, "dynamic.second")
	}
	if got := item.GetVerb(); got != "second" {
		t.Fatalf("GetVerb() after state change = %q, want %q", got, "second")
	}
	if got := item.ID(); got != "second label" {
		t.Fatalf("ID() after label change = %q, want row label %q", got, "second label")
	}
}
