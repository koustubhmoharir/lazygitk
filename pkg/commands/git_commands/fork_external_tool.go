package git_commands

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// Fork: when no diff tool is configured, git difftool prints "... is not
// configured" and guesses one. lazygitk asks the user instead (see
// docs-master/fork/Design.md). The helpers here read the tool Git would use,
// pin a tool to a single launch with --tool=<name>, and save a choice to Git's
// global config. The mode parameter is shared with the merge tool, which uses
// the same helpers with ExternalToolMerge.

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

// guiDefaultKey is the key that chooses GUI or terminal tools for the mode.
func (mode ExternalToolMode) guiDefaultKey() string {
	if mode == ExternalToolMerge {
		return "mergetool.guiDefault"
	}
	return "difftool.guiDefault"
}

// subcommand is the Git command whose --tool-help lists the mode's tools.
func (mode ExternalToolMode) subcommand() string {
	if mode == ExternalToolMerge {
		return "mergetool"
	}
	return "difftool"
}

// toolKeys lists the keys Git reads for the mode, in precedence order, as
// get_configured_merge_tool in git-mergetool--lib does. GUI mode reads the
// *.guitool keys first.
func (mode ExternalToolMode) toolKeys(gui bool) []string {
	switch {
	case mode == ExternalToolMerge && gui:
		return []string{"merge.guitool", "merge.tool"}
	case mode == ExternalToolMerge:
		return []string{"merge.tool"}
	case gui:
		return []string{"diff.guitool", "merge.guitool", "diff.tool", "merge.tool"}
	default:
		return []string{"diff.tool", "merge.tool"}
	}
}

// ExternalTool is a tool that Git reports as available, with the description
// Git gives it. A user-defined tool's description is its command.
type ExternalTool struct {
	Name        string
	Description string
}

// ConfiguredExternalTool returns the tool Git would use for the mode, or "" when
// none is configured. A missing key is not an error. The keys follow Git's GUI
// selection (see guiMode), so a tool in diff.guitool or merge.guitool wins in
// GUI mode. The name is not checked against the known tools: a name Git does
// not know is returned as configured, and a launch pinned to it fails in Git.
func (self *DiffCommands) ConfiguredExternalTool(mode ExternalToolMode) (string, error) {
	gui, err := self.guiMode(mode)
	if err != nil {
		return "", err
	}
	for _, key := range mode.toolKeys(gui) {
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

// guiMode reports whether Git runs the mode in GUI mode, as gui_mode in
// git-mergetool--lib does. A non-empty GIT_MERGETOOL_GUI (set by --gui and
// --no-gui) decides first, and only "true" means GUI. Otherwise the mode's
// guiDefault decides: "auto" is GUI when DISPLAY is set in the environment Git
// gets, and a bad boolean is an error, as it is for Git.
func (self *DiffCommands) guiMode(mode ExternalToolMode) (bool, error) {
	if override := os.Getenv("GIT_MERGETOOL_GUI"); override != "" {
		return override == "true", nil
	}
	key := mode.guiDefaultKey()
	value, err := self.configuredValue(key)
	if err != nil || value == "" {
		return false, err
	}
	if strings.EqualFold(value, "auto") {
		return os.Getenv("DISPLAY") != "", nil
	}
	return self.configuredBool(key)
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

// configuredBool reads a boolean key the way Git does: yes, on and 1 are true,
// and a value that is not a boolean is an error.
func (self *DiffCommands) configuredBool(key string) (bool, error) {
	output, err := self.cmd.New(NewGitCmd("config").Arg("--bool", "--get", key).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "true", nil
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

// ApplyExternalTool pins tool for the single command cmd by inserting
// --tool=<tool> right after the mode's subcommand. Git then uses exactly that
// tool. It ignores the configured and GUI choices, and it never guesses: a name
// it does not know fails with Git's own error. Nothing is written to the
// repository or the global config.
func ApplyExternalTool(cmd *oscommands.CmdObj, mode ExternalToolMode, tool string) error {
	execCmd := cmd.GetCmd()
	index := subcommandIndex(execCmd.Args, mode.subcommand())
	if index < 0 {
		return fmt.Errorf("cannot pin the %s tool: the command is not git %s", mode.subcommand(), mode.subcommand())
	}
	execCmd.Args = slices.Insert(execCmd.Args, index+1, "--tool="+tool)
	return nil
}

// subcommandIndex returns the index of subcommand in argv, or -1. Only the words
// before the "--" that ends the options are searched, so a path is never taken
// for the subcommand.
func subcommandIndex(argv []string, subcommand string) int {
	for index, arg := range argv {
		if arg == "--" {
			break
		}
		if arg == subcommand {
			return index
		}
	}
	return -1
}

// SaveExternalToolGlobally sets the mode's key in Git's global config, so that
// the tool is configured for every launch from now on.
func (self *DiffCommands) SaveExternalToolGlobally(mode ExternalToolMode, tool string) error {
	return self.cmd.New(NewGitCmd("config").Arg("--global", mode.configKey(), tool).ToArgv()).Run()
}
