package git_commands

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

type unmergedIndexRunner struct {
	oscommands.ICmdObjRunner
	stdout, stderr string
	err            error
}

func (runner unmergedIndexRunner) RunWithOutputs(_ *oscommands.CmdObj) (string, string, error) {
	return runner.stdout, runner.stderr, runner.err
}

func TestRebaseUnmergedIndexUsesOnlyStdout(t *testing.T) {
	queryErr := errors.New("index unreadable")
	for _, tc := range []struct {
		name, stdout, stderr string
		err                  error
		want                 bool
	}{
		{name: "clean"},
		{name: "warning only", stderr: "Git configuration warning\n"},
		{name: "literal unresolved path", stdout: "line\n\x1b[31mname\x00", want: true},
		{name: "failed query", stderr: "cannot read index", err: queryErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			common := buildGitCommon(commonDeps{})
			common.cmd = oscommands.NewDummyCmdObjBuilder(unmergedIndexRunner{stdout: tc.stdout, stderr: tc.stderr, err: tc.err})
			unmerged, err := NewRebaseCommands(common, nil, nil).HasUnmergedFiles()
			assert.Equal(t, tc.want, unmerged)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
