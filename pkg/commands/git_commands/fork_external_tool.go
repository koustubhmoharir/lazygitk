package git_commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// Fork: when no diff tool is configured, git difftool prints "... is not
// configured" and guesses one. lazygitk asks the user instead (see
// docs-master/fork/Design.md). The helpers here read the configured and the
// available tools, apply a chosen tool to a single launch through the
// environment, and save a choice to Git's global config. The difftool
// builders keep their arguments. The mode parameter is shared with the merge
// tool, which uses the same helpers with ExternalToolMerge.

// ExternalToolMode selects the Git tool a launch uses: git difftool reads
// diff.tool, falling back to merge.tool, and git mergetool reads merge.tool.
type ExternalToolMode int

const (
	ExternalToolDiff ExternalToolMode = iota
	ExternalToolMerge
)

// configKey is the key that names the tool for the mode's own setting.
func (mode ExternalToolMode) configKey() string {
	if mode == ExternalToolMerge {
		return "merge.tool"
	}
	return "diff.tool"
}

// subcommand is the Git command whose --tool-help lists the mode's tools.
func (mode ExternalToolMode) subcommand() string {
	if mode == ExternalToolMerge {
		return "mergetool"
	}
	return "difftool"
}

// ExternalTool is a tool that Git reports as available, with the description
// Git gives it. A user-defined tool's description is its command.
type ExternalTool struct {
	Name        string
	Description string
}

// ConfiguredExternalTool returns the tool Git uses for the mode, or "" when none
// is configured. A missing key is not an error. Like git difftool, the diff
// mode falls back to merge.tool when diff.tool is unset.
func (self *DiffCommands) ConfiguredExternalTool(mode ExternalToolMode) (string, error) {
	keys := []string{mode.configKey()}
	if mode == ExternalToolDiff {
		keys = append(keys, ExternalToolMerge.configKey())
	}
	for _, key := range keys {
		value, err := self.configuredValue(key)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
	}
	return "", nil
}

// configuredValue reads a key from the current config, bypassing the cache, so
// that a tool saved while lazygitk runs is seen at the next launch.
func (self *DiffCommands) configuredValue(key string) (string, error) {
	output, err := self.cmd.New(NewGitCmd("config").Arg("--get", key).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if commandExitedWithOne(err) {
		// git config --get exits 1 when the key is not set.
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// AvailableExternalTools lists the tools Git reports for the mode, in Git's
// order, with their descriptions.
func (self *DiffCommands) AvailableExternalTools(mode ExternalToolMode) ([]ExternalTool, error) {
	output, err := self.cmd.New(NewGitCmd(mode.subcommand()).Arg("--tool-help").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}
	return parseExternalToolHelp(output), nil
}

// externalToolUnavailableMarker starts the section of tools that are valid but
// not installed. Only the lines before it name available tools.
const externalToolUnavailableMarker = "The following tools are valid, but not currently available:"

// parseExternalToolHelp reads the output of `git difftool --tool-help`. Tool
// lines are indented by two tabs and come before the unavailable section. The
// header, the "user-defined:" label and the note about windowed tools are not
// indented that way, so they are skipped. A user-defined line is "NAME.cmd
// COMMAND", and its tool is NAME. Each tool appears once, first occurrence kept.
func parseExternalToolHelp(output string) []ExternalTool {
	var tools []ExternalTool
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.Contains(line, externalToolUnavailableMarker) {
			break
		}
		if !strings.HasPrefix(line, "\t\t") {
			continue
		}
		body := strings.TrimSpace(line)
		name, description := body, ""
		if index := strings.IndexAny(body, " \t"); index >= 0 {
			name, description = body[:index], strings.TrimSpace(body[index:])
		}
		if userDefined, ok := strings.CutSuffix(name, ".cmd"); ok {
			name = userDefined
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		tools = append(tools, ExternalTool{Name: name, Description: description})
	}
	return tools
}

// ApplyExternalTool makes Git use tool for the single command cmd, by setting
// the key through GIT_CONFIG_COUNT and GIT_CONFIG_KEY_n / GIT_CONFIG_VALUE_n.
// Any entries already in the environment are kept, and the new one is appended
// as the last. Nothing is written to the repository or the global config.
func ApplyExternalTool(cmd *oscommands.CmdObj, mode ExternalToolMode, tool string) error {
	count := 0
	for _, entry := range cmd.GetEnvVars() {
		if value, ok := strings.CutPrefix(entry, "GIT_CONFIG_COUNT="); ok {
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 {
				return fmt.Errorf("cannot add a tool to the environment: GIT_CONFIG_COUNT is %q", value)
			}
			count = parsed // Go keeps the last duplicate, so the last value wins.
		}
	}
	cmd.RemoveEnvVar("GIT_CONFIG_COUNT")
	cmd.AddEnvVars(
		fmt.Sprintf("GIT_CONFIG_COUNT=%d", count+1),
		fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", count, mode.configKey()),
		fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", count, tool),
	)
	return nil
}

// SaveExternalToolGlobally sets the mode's key in Git's global config, so that
// the tool is configured for every launch from now on.
func (self *DiffCommands) SaveExternalToolGlobally(mode ExternalToolMode, tool string) error {
	return self.cmd.New(NewGitCmd("config").Arg("--global", mode.configKey(), tool).ToArgv()).Run()
}
