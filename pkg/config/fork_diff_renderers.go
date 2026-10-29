package config

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/jesseduffield/generics/set"
)

// Fork: when git.diffRenderers is empty, which includes unset and an explicit
// [], the fork uses delta followed by git's own diff, so that | cycles between
// them (roadmap item 15). Upstream's empty list meant git's own diff, with no
// GIT_PAGER. If delta is not on PATH, the delta entry is dropped and git's own
// diff is used, with a notice at startup (see pkg/gui/fork_diff_renderer_notice.go).
// A non-empty list the user configured is never changed, not even when it names
// delta.

// forkDeltaBlueStyles are the delta styles whose default colour is the
// terminal's basic blue, with what each adds to the colour. Many terminal
// palettes make that blue too dark to read on a black background.
var forkDeltaBlueStyles = []struct{ key, attributes string }{
	{"file-style", ""},
	{"file-decoration-style", " ul"},
	{"hunk-header-file-style", ""},
	{"hunk-header-line-number-style", ""},
	{"hunk-header-decoration-style", " box"},
	{"line-numbers-left-style", ""},
	{"line-numbers-right-style", ""},
}

// forkDeltaCommand returns the default delta renderer command. It gives the
// styles in forkDeltaBlueStyles the palette's Blue, except those that the git
// config sets. gitDeltaConfig is the git config's delta settings as `git config
// --get-regexp` prints them. A style set in a delta feature counts as set,
// whether or not the feature is in use. {{colorScheme}} is resolved per render
// like any other renderer command.
func forkDeltaCommand(gitDeltaConfig string) string {
	configured := set.New[string]()
	for _, line := range strings.Split(gitDeltaConfig, "\n") {
		name, _, _ := strings.Cut(line, " ")
		configured.Add(strings.ToLower(name[strings.LastIndex(name, ".")+1:]))
	}

	var command strings.Builder
	fmt.Fprintf(&command, `{{$blue := %q}}{{if eq colorScheme "light"}}{{$blue = %q}}{{end}}`,
		ForkPaletteBlue(false), ForkPaletteBlue(true))
	command.WriteString("delta --{{colorScheme}} --paging=never --tabs=4")
	for _, style := range forkDeltaBlueStyles {
		if !configured.Includes(style.key) {
			fmt.Fprintf(&command, ` --%s="{{$blue}}%s"`, style.key, style.attributes)
		}
	}
	return command.String()
}

// forkLookPath finds delta on PATH. It is a variable so that tests can decide
// whether delta is present without depending on the machine they run on.
var forkLookPath = exec.LookPath

// forkDefaultDiffRenderers is the list used when the user's list is empty and
// delta is on PATH.
func forkDefaultDiffRenderers(gitDeltaConfig string) []DiffRendererConfig {
	return []DiffRendererConfig{
		{Name: "delta", ColorArg: "always", Command: DiffRendererCommandType(forkDeltaCommand(gitDeltaConfig))},
		{Type: "rawGit", Name: "default"},
	}
}

// forkFallbackDiffRenderers is the list used when the user's list is empty and
// delta is not on PATH: git's own diff only.
func forkFallbackDiffRenderers() []DiffRendererConfig {
	return []DiffRendererConfig{
		{Type: "rawGit", Name: "default"},
	}
}

// forkDeltaAvailability caches the PATH lookup for delta, which does not change
// while lazygitk runs, so that renders don't repeat it.
type forkDeltaAvailability struct {
	checked bool
	found   bool
}

func (self *forkDeltaAvailability) available() bool {
	if !self.checked {
		_, err := forkLookPath("delta")
		self.found = err == nil
		self.checked = true
	}
	return self.found
}

// forkEffectiveDiffRenderers returns the renderer list the manager uses. A
// non-empty user list is returned unchanged. An empty one gets the fork default,
// without delta when delta is not on PATH; fellBack reports that case. The
// result is never empty.
func (self *DiffRendererConfigManager) forkEffectiveDiffRenderers() (diffRenderers []DiffRendererConfig, fellBack bool) {
	userDiffRenderers := self.getUserConfig().Git.DiffRenderers
	if len(userDiffRenderers) > 0 {
		return userDiffRenderers, false
	}
	if self.forkDelta.available() {
		return forkDefaultDiffRenderers(self.forkGitDeltaConfig()), false
	}
	return forkFallbackDiffRenderers(), true
}

// DefaultDiffRendererFellBack reports whether the fork default applies without
// delta, because delta is not on PATH. The GUI shows a notice for it once at
// startup.
func (self *DiffRendererConfigManager) DefaultDiffRendererFellBack() bool {
	_, fellBack := self.forkEffectiveDiffRenderers()
	return fellBack
}

// SetForkGitDeltaConfig sets where the default delta command finds the git
// config's delta settings: a function that returns them as `git config
// --get-regexp` prints them.
func (self *DiffRendererConfigManager) SetForkGitDeltaConfig(getGitDeltaConfig func() string) {
	self.getForkGitDeltaConfig = getGitDeltaConfig
}

func (self *DiffRendererConfigManager) forkGitDeltaConfig() string {
	if self.getForkGitDeltaConfig == nil {
		return ""
	}
	return self.getForkGitDeltaConfig()
}
