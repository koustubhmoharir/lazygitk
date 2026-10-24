package git_commands

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// ResolveHistoricalComparisonCommit resolves only pasted hexadecimal object
// IDs. It deliberately bypasses refs so a hex-named ref cannot shadow an OID.
func (self *DiffCommands) ResolveHistoricalComparisonCommit(input string) (HistoricalDiffCommit, error) {
	input = strings.TrimFunc(input, unicode.IsSpace)
	if len(input) < 4 || len(input) > 64 {
		return HistoricalDiffCommit{}, fmt.Errorf("commit ID must contain 4 to 64 hexadecimal characters")
	}
	for _, char := range input {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return HistoricalDiffCommit{}, fmt.Errorf("commit ID must contain only hexadecimal characters")
		}
	}

	output, err := self.historicalComparisonOutput(NewGitCmd("rev-parse").Arg("--disambiguate=" + input).ToArgv())
	if err != nil {
		return HistoricalDiffCommit{}, err
	}
	candidates := strings.Fields(output)
	if len(candidates) == 0 {
		return HistoricalDiffCommit{}, fmt.Errorf("no Git object matches commit ID %q", input)
	}
	if len(candidates) != 1 {
		return HistoricalDiffCommit{}, fmt.Errorf("commit ID %q is ambiguous (%d Git objects match)", input, len(candidates))
	}
	if err := validateObjectID(candidates[0]); err != nil {
		return HistoricalDiffCommit{}, fmt.Errorf("git returned invalid object ID: %w", err)
	}
	kind, err := self.historicalComparisonOutput(NewGitCmd("cat-file").Arg("-t", candidates[0]).ToArgv())
	if err != nil {
		return HistoricalDiffCommit{}, err
	}
	if strings.TrimSpace(kind) != "commit" {
		return HistoricalDiffCommit{}, fmt.Errorf("Git object %q is %q, not a commit", candidates[0], strings.TrimSpace(kind))
	}
	// Parse stdout independently: Git may warn about a hex-named ref on stderr
	// even though the full object ID resolves correctly. Never recover by
	// searching a mixed diagnostic/data stream for a plausible hash.
	output, err = self.historicalComparisonOutput(NewGitCmd("rev-list").Arg("--parents", "-n", "1", candidates[0], "--").ToArgv())
	if err != nil {
		return HistoricalDiffCommit{}, err
	}
	fields := strings.Fields(output)
	if len(fields) == 0 || fields[0] != candidates[0] {
		return HistoricalDiffCommit{}, fmt.Errorf("git returned invalid commit parent list")
	}
	result := HistoricalDiffCommit{OID: candidates[0]}
	for _, parent := range fields[1:] {
		if err := validateObjectID(parent); err != nil {
			return HistoricalDiffCommit{}, fmt.Errorf("git returned invalid parent: %w", err)
		}
	}
	if len(fields) > 1 {
		result.FirstParent = fields[1]
	}
	return result, nil
}

func (self *DiffCommands) historicalComparisonOutput(args []string) (string, error) {
	output, diagnostic, err := self.cmd.New(args).SetWd(self.repoPaths.WorktreePath()).RunWithOutputs()
	if err != nil {
		return "", fmt.Errorf("resolve comparison commit: %w: %s", err, strings.TrimSpace(diagnostic))
	}
	return output, nil
}

// HistoricalCommitComparisonCmdObj builds, but does not execute, a literal
// path-scoped difftool comparison between two resolved commits.
func (self *DiffCommands) HistoricalCommitComparisonCmdObj(left, right HistoricalDiffCommit, paths []string, directory bool) (*oscommands.CmdObj, error) {
	if err := validateObjectID(left.OID); err != nil {
		return nil, fmt.Errorf("invalid left commit ID: %w", err)
	}
	if err := validateObjectID(right.OID); err != nil {
		return nil, fmt.Errorf("invalid right commit ID: %w", err)
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, path := range paths {
		if path != "." && !validTreePath(path) {
			return nil, fmt.Errorf("invalid repository path %q", path)
		}
	}
	args := NewGitCmd("--literal-pathspecs").Arg("difftool", "--no-prompt").ArgIf(directory, "--dir-diff").Arg(left.OID, right.OID).Arg("--").Arg(paths...)
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()), nil
}
