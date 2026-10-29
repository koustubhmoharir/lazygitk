package config

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/utils"
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
// with its value for a background.
type forkDeltaStyle struct {
	key   string
	value func(lightBackground bool, backgroundColor string) string
}

// forkDeltaStyles are the delta settings that the default delta renderer
// passes, so that its colours follow the palette and the terminal's
// background.
var forkDeltaStyles = []forkDeltaStyle{
	// The palette's colours are true colours, which delta would round to
	// its 256-colour palette unless it sees COLORTERM. gocui takes true
	// colours whatever the terminal supports.
	{"true-color", func(bool, string) string { return "always" }},

	// delta's default for these is the terminal's basic blue, which many
	// terminal palettes make too dark to read on a black background.
	{"file-style", forkDeltaBlue("")},
	{"file-decoration-style", forkDeltaBlue(" ul")},
	{"hunk-header-file-style", forkDeltaBlue("")},
	{"hunk-header-line-number-style", forkDeltaBlue("")},
	{"hunk-header-decoration-style", forkDeltaBlue(" box")},
	{"line-numbers-left-style", forkDeltaBlue("")},
	{"line-numbers-right-style", forkDeltaBlue("")},

	// delta's default backgrounds for changed lines are bright enough to drown
	// the dimmer syntax colours of added lines, such as comments. These are a
	// little of the palette's Green or Red over the terminal's background, and
	// a little more for the changed part of a line and for the marker of an
	// empty line. Deleted lines keep delta's plain text.
	{"plus-style", forkDeltaTint("syntax", forkPaletteGreen, forkDeltaLineTint)},
	{"plus-emph-style", forkDeltaTint("syntax", forkPaletteGreen, forkDeltaChangeTint)},
	{"plus-empty-line-marker-style", forkDeltaTint("normal", forkPaletteGreen, forkDeltaChangeTint)},
	{"minus-style", forkDeltaTint("normal", forkPaletteRed, forkDeltaLineTint)},
	{"minus-emph-style", forkDeltaTint("normal", forkPaletteRed, forkDeltaChangeTint)},
	{"minus-empty-line-marker-style", forkDeltaTint("normal", forkPaletteRed, forkDeltaChangeTint)},
}

// How much of the palette's Green or Red tints a changed line, and the changed
// part of it. On a black background, delta's default syntax colours keep a
// contrast of at least 3.5 and 2.7 over them.
const (
	forkDeltaLineTint   = 0.12
	forkDeltaChangeTint = 0.22
)

func forkDeltaBlue(attributes string) func(bool, string) string {
	return func(lightBackground bool, _ string) string {
		return ForkPaletteBlue(lightBackground) + attributes
	}
}

func forkDeltaTint(foreground string, color func(lightBackground bool) string, amount float64) func(bool, string) string {
	return func(lightBackground bool, backgroundColor string) string {
		if len(backgroundColor) != 7 || !utils.IsValidHexValue(backgroundColor) {
			backgroundColor = lo.Ternary(lightBackground, "#ffffff", "#000000")
		}
		return foreground + " " + mixHexColors(backgroundColor, color(lightBackground), amount)
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
		if !configured.Includes(style.key) {
			command += fmt.Sprintf(` --%s="%s"`, style.key, style.value(values.LightBackground, values.BackgroundColor))
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
