package config

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/jesseduffield/generics/set"
	"github.com/samber/lo"
)

// Fork: when git.diffRenderers is empty, which includes unset and an explicit
// [], the fork uses delta followed by git's own diff, so that | cycles between
// them (roadmap item 15). Upstream's empty list meant git's own diff, with no
// GIT_PAGER. If delta is not on PATH, the delta entry is dropped and git's own
// diff is used, with a notice at startup (see pkg/gui/fork_diff_renderer_notice.go).
// A non-empty list the user configured is never changed, not even when it names
// delta.

// forkDeltaStyle is a delta setting that the default delta renderer passes,
// with its value for the render values, or "" to leave it to delta.
type forkDeltaStyle struct {
	key   string
	value func(DiffRendererValues) string
}

// forkDeltaStyles are the delta settings that the default delta renderer
// passes, so that its colours follow the palette.
var forkDeltaStyles = []forkDeltaStyle{
	// delta rounds colours to its 256-colour palette unless it sees COLORTERM,
	// which the screen may show true colour without.
	{"true-color", func(values DiffRendererValues) string {
		return lo.Ternary(values.TrueColor, "always", "")
	}},

	// delta's default for these is the terminal's basic blue, which many
	// terminal palettes make too dark to read on a black background.
	{"file-style", forkDeltaBlue("")},
	{"file-decoration-style", forkDeltaBlue(" ul")},
	{"hunk-header-file-style", forkDeltaBlue("")},
	{"hunk-header-line-number-style", forkDeltaBlue("")},
	{"hunk-header-decoration-style", forkDeltaBlue(" box")},
	{"line-numbers-left-style", forkDeltaBlue("")},
	{"line-numbers-right-style", forkDeltaBlue("")},

	// On a dark background, delta's default backgrounds for the changed part
	// of a line, and for whole lines once rounded to 256 colours, are bright
	// enough to drown the dimmer syntax colours of added lines, such as
	// comments. These are dark greens and reds, a little brighter for the
	// changed part of a line and the marker of an empty line. The 256-colour
	// palette has no dark greens or reds, so without true colour delta keeps
	// its own.
	{"plus-style", forkDeltaDarkTint("syntax #0a300c")},
	{"plus-emph-style", forkDeltaDarkTint("syntax #104a14")},
	{"plus-empty-line-marker-style", forkDeltaDarkTint("normal #104a14")},
	{"minus-style", forkDeltaDarkTint("normal #4a1218")},
	{"minus-emph-style", forkDeltaDarkTint("normal #6e1e28")},
	{"minus-empty-line-marker-style", forkDeltaDarkTint("normal #6e1e28")},
}

func forkDeltaBlue(attributes string) func(DiffRendererValues) string {
	return func(values DiffRendererValues) string {
		return ForkPaletteBlue(values.LightBackground) + attributes
	}
}

func forkDeltaDarkTint(style string) func(DiffRendererValues) string {
	return func(values DiffRendererValues) string {
		return lo.Ternary(values.TrueColor && !values.LightBackground, style, "")
	}
}

// forkDeltaCommand returns the default delta renderer command for values. It
// passes the settings in forkDeltaStyles, except those that the git config sets.
// gitDeltaConfig is the git config's delta settings as `git config
// --get-regexp` prints them. A setting in a delta feature counts as set, whether
// or not the feature is in use.
func forkDeltaCommand(gitDeltaConfig string, values DiffRendererValues) string {
	configured := set.New[string]()
	for _, line := range strings.Split(gitDeltaConfig, "\n") {
		name, _, _ := strings.Cut(line, " ")
		configured.Add(strings.ToLower(name[strings.LastIndex(name, ".")+1:]))
	}

	command := "delta --" + lo.Ternary(values.LightBackground, "light", "dark") + " --paging=never --tabs=4"
	for _, style := range forkDeltaStyles {
		if value := style.value(values); value != "" && !configured.Includes(style.key) {
			command += fmt.Sprintf(` --%s="%s"`, style.key, value)
		}
	}
	return command
}

// forkLookPath finds delta on PATH. It is a variable so that tests can decide
// whether delta is present without depending on the machine they run on.
var forkLookPath = exec.LookPath

// forkDefaultDiffRenderers is the list used when the user's list is empty and
// delta is on PATH.
func forkDefaultDiffRenderers(gitDeltaConfig string) []DiffRendererConfig {
	return []DiffRendererConfig{
		{Name: "delta", ColorArg: "always", forkResolveCommand: func(values DiffRendererValues) string {
			return forkDeltaCommand(gitDeltaConfig, values)
		}},
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
