package config

import "os/exec"

// Fork: when git.diffRenderers is empty, which includes unset and an explicit
// [], the fork uses delta followed by git's own diff, so that | cycles between
// them (roadmap item 15). Upstream's empty list meant git's own diff, with no
// GIT_PAGER. If delta is not on PATH, the delta entry is dropped and git's own
// diff is used, with a notice at startup (see pkg/gui/fork_diff_renderer_notice.go).
// A non-empty list the user configured is never changed, not even when it names
// delta.

// forkDeltaCommand is the default delta renderer command. {{colorScheme}} is
// resolved per render like any other renderer command.
const forkDeltaCommand = "delta --{{colorScheme}} --paging=never --tabs=4"

// forkLookPath finds delta on PATH. It is a variable so that tests can decide
// whether delta is present without depending on the machine they run on.
var forkLookPath = exec.LookPath

// forkDefaultDiffRenderers is the list used when the user's list is empty and
// delta is on PATH.
func forkDefaultDiffRenderers() []DiffRendererConfig {
	return []DiffRendererConfig{
		{Name: "delta", ColorArg: "always", Command: forkDeltaCommand},
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
		return forkDefaultDiffRenderers(), false
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
