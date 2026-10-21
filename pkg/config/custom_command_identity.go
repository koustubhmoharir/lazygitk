package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

const customCommandIdentityVersion = 1

var customCommandIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type customCommandIdentityPayload struct {
	Version int                     `json:"version"`
	Command string                  `json:"command"`
	Prompts []CustomCommandPrompt   `json:"prompts"`
	Output  string                  `json:"output"`
	After   *CustomCommandAfterHook `json:"after"`
}

// CommandIdentity returns the explicit ID, or a hash of the command definition.
// Set ID when identity must remain stable across definition changes.
func (c CustomCommand) CommandIdentity() string {
	if c.ID != "" {
		return "custom.user." + c.ID
	}

	if len(c.CommandMenu) > 0 {
		identities := make([]string, 0, len(c.CommandMenu))
		for _, child := range c.CommandMenu {
			identities = append(identities, child.CommandIdentity())
		}
		sort.Strings(identities)
		return "custom.menu." + hashCustomCommandIdentity(identities)
	}

	payload := customCommandIdentityPayload{
		Version: customCommandIdentityVersion,
		Command: c.Command,
		Prompts: c.Prompts,
		Output:  c.Output,
		After:   c.After,
	}
	return "custom.auto." + hashCustomCommandIdentity(payload)
}

func hashCustomCommandIdentity(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		// These payloads contain only JSON-compatible config values.
		panic(fmt.Sprintf("marshal custom command identity: %v", err))
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func validateCustomCommandIdentities(commands []CustomCommand) error {
	seen := make(map[string]string)
	var visit func([]CustomCommand, string) error
	visit = func(commands []CustomCommand, path string) error {
		for i := range commands {
			command := &commands[i]
			commandPath := fmt.Sprintf("%s[%d]", path, i)
			if command.ID != "" {
				if !customCommandIDPattern.MatchString(command.ID) {
					return fmt.Errorf("custom command ID %q at %s must start with an ASCII letter or digit and contain only ASCII letters, digits, '.', '_' or '-'.", command.ID, commandPath)
				}
				if previousPath, ok := seen[command.ID]; ok {
					return fmt.Errorf("duplicate custom command ID %q at %s and %s.", command.ID, previousPath, commandPath)
				}
				seen[command.ID] = commandPath
			}
			if err := visit(command.CommandMenu, commandPath+".commandMenu"); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(commands, "customCommands")
}
