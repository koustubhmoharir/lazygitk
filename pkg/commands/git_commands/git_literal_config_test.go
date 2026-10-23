package git_commands

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitLiteralConfig(t *testing.T) {
	root := initHistoricalRepo(t)
	key := "branch.foo=bar.mergeOptions"
	gitTest(t, root, "config", "--file", root+"/.git/config", key, "disk-value")
	diskBefore := gitTest(t, root, "config", "--file", root+"/.git/config", "--get", key)

	// Git's -c syntax splits at the first '=' and therefore addresses the
	// wrong key when the subsection itself contains '='.
	old := exec.Command("git", "-C", root, "-c", key+"=", "config", "--get", key)
	oldOutput, oldErr := old.CombinedOutput()
	if !assert.NoError(t, oldErr, string(oldOutput)) {
		return
	}
	assert.Equal(t, diskBefore, strings.TrimSpace(string(oldOutput)))

	cmd := exec.Command("git", append([]string{"-C", root}, NewGitCmd("config").
		configWithLiteralKey(key, "unused-value", literalMergeOptionsEnv).
		Arg("--get", key).ToArgv()[1:]...)...)
	inherited := []string{
		literalMergeOptionsEnv + "=", "GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=preserved",
		"GIT_CONFIG_KEY_1=" + key, "GIT_CONFIG_VALUE_1=inherited-options",
	}
	cmd.Env = append(cmd.Environ(), inherited...)
	output, err := cmd.CombinedOutput()
	if !assert.NoError(t, err, string(output)) {
		return
	}
	assert.Empty(t, strings.TrimSpace(string(output)), "the literal CLI override must also override inherited Git settings")
	assert.Equal(t, diskBefore, gitTest(t, root, "config", "--file", root+"/.git/config", "--get", key))
	user := exec.Command("git", append([]string{"-C", root}, NewGitCmd("config").
		configWithLiteralKey(key, "", literalMergeOptionsEnv).Arg("--get", "user.name").ToArgv()[1:]...)...)
	user.Env = append(user.Environ(), inherited...)
	userOutput, err := user.CombinedOutput()
	assert.NoError(t, err)
	assert.Equal(t, "preserved", strings.TrimSpace(string(userOutput)))

	cmd = exec.Command("git", append([]string{"-C", root}, NewGitCmd("config").
		configWithLiteralKey(key, "unused=value", literalMergeOptionsEnv).
		Arg("--get", key).ToArgv()[1:]...)...)
	cmd.Env = append(cmd.Environ(), literalMergeOptionsEnv+"=enabled=value")
	output, err = cmd.CombinedOutput()
	if !assert.NoError(t, err, string(output)) {
		return
	}
	assert.Equal(t, "enabled=value", strings.TrimSpace(string(output)))
}

func TestGitLiteralConfigArgv(t *testing.T) {
	assert.Equal(t,
		[]string{"git", "-c", "branch.topic.mergeOptions=false", "merge"},
		NewGitCmd("merge").configWithLiteralKey("branch.topic.mergeOptions", "false", literalMergeOptionsEnv).ToArgv(),
	)
	assert.Equal(t,
		[]string{"git", "--config-env=branch.foo=bar.mergeOptions=LAZYGITK_MERGE_OPTIONS", "merge"},
		NewGitCmd("merge").configWithLiteralKey("branch.foo=bar.mergeOptions", "", literalMergeOptionsEnv).ToArgv(),
	)
}

func TestGitLiteralConfigRemoteMirror(t *testing.T) {
	root := initHistoricalRepo(t)
	key := "remote.peer=one.mirror"
	gitTest(t, root, "config", key, "true")
	cmd := exec.Command("git", append([]string{"-C", root}, NewGitCmd("config").
		configWithLiteralKey(key, "false", literalMirrorEnv).
		Arg("--type=bool", "--get", key).ToArgv()[1:]...)...)
	cmd.Env = append(cmd.Environ(), literalMirrorEnv+"=false")
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err)
	assert.Equal(t, "false", strings.TrimSpace(string(output)))
	assert.Equal(t, "true", gitTest(t, root, "config", "--get", key))
}
