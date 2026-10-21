package config

import (
	"strings"
	"testing"
)

func TestCustomCommandIdentity(t *testing.T) {
	const secret = "literal-command-secret-93b7"
	base := CustomCommand{
		Command: "git fetch " + secret + " {{.Form.Remote}}",
		Prompts: []CustomCommandPrompt{{Type: "input", Key: "Remote", Title: "Remote", InitialValue: "prompt-default-secret-51cd"}},
		Output:  "log",
		After:   &CustomCommandAfterHook{CheckForConflicts: true},
	}
	want := base.CommandIdentity()

	for name, changed := range map[string]CustomCommand{
		"key":          {Key: Keybinding{"x"}, Command: base.Command, Prompts: base.Prompts, Output: base.Output, After: base.After},
		"context":      {Context: "files", Command: base.Command, Prompts: base.Prompts, Output: base.Output, After: base.After},
		"description":  {Description: "another label", Command: base.Command, Prompts: base.Prompts, Output: base.Output, After: base.After},
		"loading text": {LoadingText: "wait", Command: base.Command, Prompts: base.Prompts, Output: base.Output, After: base.After},
		"output title": {OutputTitle: "title", Command: base.Command, Prompts: base.Prompts, Output: base.Output, After: base.After},
	} {
		t.Run(name, func(t *testing.T) {
			if got := changed.CommandIdentity(); got != want {
				t.Fatalf("identity changed: %s != %s", got, want)
			}
		})
	}

	aliases := base
	if aliases.CommandIdentity() != want {
		t.Fatal("identical definition alias has a different identity")
	}
	for name, changed := range map[string]CustomCommand{
		"command": {Command: base.Command + " --prune", Prompts: base.Prompts, Output: base.Output, After: base.After},
		"prompts": {Command: base.Command, Prompts: append(base.Prompts, CustomCommandPrompt{Type: "confirm", Body: "continue?"}), Output: base.Output, After: base.After},
		"output":  {Command: base.Command, Prompts: base.Prompts, Output: "popup", After: base.After},
		"after":   {Command: base.Command, Prompts: base.Prompts, Output: base.Output},
	} {
		if changed.CommandIdentity() == want {
			t.Errorf("%s change did not change identity", name)
		}
	}

	explicit := base
	explicit.ID = "release.fetch"
	if got := explicit.CommandIdentity(); got != "custom.user.release.fetch" {
		t.Fatalf("unexpected explicit identity: %s", got)
	}
	explicit.Command = "changed"
	if got := explicit.CommandIdentity(); got != "custom.user.release.fetch" {
		t.Fatalf("explicit identity changed with definition: %s", got)
	}

	menuA := CustomCommand{CommandMenu: []CustomCommand{{Command: "one"}, {Command: "two"}}}
	menuB := CustomCommand{CommandMenu: []CustomCommand{{Command: "two"}, {Command: "one"}}}
	if menuA.CommandIdentity() != menuB.CommandIdentity() {
		t.Fatal("menu identity changed after reordering children")
	}
	if strings.Contains(want, secret) || strings.Contains(want, "prompt-default-secret-51cd") {
		t.Fatalf("identity exposes command or prompt secret: %s", want)
	}
}

func TestCustomCommandIdentitiesValidation(t *testing.T) {
	tests := []struct {
		name     string
		commands []CustomCommand
		wantErr  string
	}{
		{name: "invalid first character", commands: []CustomCommand{{ID: ".bad"}}, wantErr: `".bad" at customCommands[0]`},
		{name: "invalid character", commands: []CustomCommand{{ID: "bad/id"}}, wantErr: `"bad/id" at customCommands[0]`},
		{name: "duplicate", commands: []CustomCommand{{ID: "same"}, {ID: "same"}}, wantErr: `"same" at customCommands[0] and customCommands[1]`},
		{name: "nested duplicate", commands: []CustomCommand{{ID: "same", CommandMenu: []CustomCommand{{ID: "same"}}}}, wantErr: `"same" at customCommands[0] and customCommands[0].commandMenu[0]`},
		{name: "cross tree duplicate", commands: []CustomCommand{{CommandMenu: []CustomCommand{{ID: "same"}}}, {ID: "same"}}, wantErr: `"same" at customCommands[0].commandMenu[0] and customCommands[1]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCustomCommandIdentities(test.commands)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}

	config := GetDefaultConfig()
	config.CustomCommands = []CustomCommand{{Context: "files, commits", Command: "echo {{.Selected.Name}}"}}
	if err := config.Validate(); err != nil {
		t.Fatalf("valid no-ID multi-context command rejected: %v", err)
	}

	for _, commands := range [][]CustomCommand{
		{{ID: ".invalid", Command: "echo ok"}},
		{{ID: "duplicate", Command: "echo one"}, {ID: "duplicate", Command: "echo two"}},
	} {
		config := GetDefaultConfig()
		config.CustomCommands = commands
		if err := config.Validate(); err == nil {
			t.Errorf("UserConfig.Validate accepted invalid custom command IDs: %#v", commands)
		}
	}

	config = GetDefaultConfig()
	config.CustomCommands = []CustomCommand{{
		Description: "menu",
		CommandMenu: []CustomCommand{{ID: "nested.ok-1", Command: "echo ok", Context: "global"}},
	}}
	if err := config.Validate(); err != nil {
		t.Fatalf("valid explicit ID on nested menu command rejected: %v", err)
	}
}
