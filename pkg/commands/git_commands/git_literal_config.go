package git_commands

import "strings"

const (
	literalMergeOptionsEnv = "LAZYGITK_MERGE_OPTIONS"
	literalPruneTagsEnv    = "LAZYGITK_PRUNE_TAGS"
	literalMirrorEnv       = "LAZYGITK_REMOTE_MIRROR"
)

// configWithLiteralKey overrides a config key whose subsection contains '='.
// For such keys Git's -c parser splits at the first '=', so --config-env is
// needed to preserve the literal key. The caller must add envName=value to
// the command's CmdObj with AddEnvVars; for this special case Git reads the
// value from that environment variable. These environment names are fixed
// constants shared with callers.
func (self *GitCommandBuilder) configWithLiteralKey(key, value, envName string) *GitCommandBuilder {
	if !strings.ContainsRune(key, '=') {
		return self.Config(key + "=" + value)
	}
	self.args = append([]string{"--config-env=" + key + "=" + envName}, self.args...)
	return self
}
