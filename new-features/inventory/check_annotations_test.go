//go:build ignore

package main

import (
	"bytes"
	"testing"
)

func TestParseAndNormalize(t *testing.T) {
	base := []byte("package p\n type Binding struct { Handler func(); Keys []string; Guard bool; Description string }\n var b = Binding{CommandID: \"old\", Verb: \"oldverb\", Handler: run, Keys: []string{\"x\"}, Guard: true, Description: \"do it\"}\n")
	tests := []struct {
		name, source string
		equal        bool
	}{
		{"metadata format comments", "package p\n// moved\ntype Binding struct { Handler func(); Keys []string; Guard bool; Description string }\nvar b = Binding{\n CommandID: \"new\", Verb: \"newverb\", Handler: run, Keys: []string{\"x\"}, Guard: true, Description: \"do it\",\n}\n", true},
		{"handler", "package p\ntype Binding struct { Handler func(); Keys []string; Guard bool; Description string }\nvar b = Binding{CommandID: \"new\", Verb: \"v\", Handler: other, Keys: []string{\"x\"}, Guard: true, Description: \"do it\"}\n", false},
		{"keys", "package p\ntype Binding struct { Handler func(); Keys []string; Guard bool; Description string }\nvar b = Binding{CommandID: \"new\", Verb: \"v\", Handler: run, Keys: []string{\"y\"}, Guard: true, Description: \"do it\"}\n", false},
		{"guard", "package p\ntype Binding struct { Handler func(); Keys []string; Guard bool; Description string }\nvar b = Binding{CommandID: \"new\", Verb: \"v\", Handler: run, Keys: []string{\"x\"}, Guard: false, Description: \"do it\"}\n", false},
		{"description", "package p\ntype Binding struct { Handler func(); Keys []string; Guard bool; Description string }\nvar b = Binding{CommandID: \"new\", Verb: \"v\", Handler: run, Keys: []string{\"x\"}, Guard: true, Description: \"other\"}\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := parseAndNormalize("probe.go", base)
			if err != nil {
				t.Fatal(err)
			}
			b, err := parseAndNormalize("probe.go", []byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			if got := bytes.Equal(a, b); got != tt.equal {
				t.Fatalf("equal=%v, want %v", got, tt.equal)
			}
		})
	}
}
