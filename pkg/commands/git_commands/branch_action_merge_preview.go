package git_commands

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// BranchActionMergePreviewOptions describes the ort strategy used for a native
// path preview. Callers must retain these options for the eventual merge too.
type BranchActionMergePreviewOptions struct {
	StrategyOptions         []string
	AllowUnrelatedHistories bool
}

// PreviewBranchActionOrtMerge returns paths that Git would check out, including
// conflict and directory-rename paths absent from either input tree. This does
// not authorize a merge or approve ignored-file loss. Invoke it in an explicit
// merge review: native Git may run the configured low-level merge drivers.
//
// Git's object-only merge-tree writes into private temporary object storage.
// The repository's objects remain available through an alternate, retaining
// existing alternate stores too. No refs, index or working files are changed
// by the preview itself. Git must support merge-tree --write-tree (2.38+).
func (self *BranchCommands) PreviewBranchActionOrtMerge(target BranchActionTarget, opts BranchActionMergePreviewOptions) (_ []string, runErr error) {
	if err := self.CheckBranchActionTarget(target); err != nil {
		return nil, err
	}
	options := slices.Clone(opts.StrategyOptions)
	if target.HeadOID() == "" {
		// A regular merge into an unborn HEAD installs the selected tree.
		paths, err := self.branchActionTreePaths(target.SourceOID()+"^{tree}", nil)
		if err != nil {
			return nil, err
		}
		if err := self.CheckBranchActionTarget(target); err != nil {
			return nil, err
		}
		return paths, nil
	}
	output, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--path-format=absolute", "--git-path", "objects").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}
	objects := trimTerminalLineEnding(output)
	if objects == "" {
		return nil, fmt.Errorf("Git returned an empty object directory")
	}
	store, err := os.MkdirTemp("", "lazygitk-merge-objects-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(store); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("remove merge preview object storage: %w", err))
		}
	}()
	objectEnvironment := func(cmd *oscommands.CmdObj) *oscommands.CmdObj {
		alternates := quoteMergeAlternate(objects)
		// Inspect the command's effective environment, not just this process:
		// the builder may already supply repository-specific object stores.
		var inherited string
		for _, entry := range cmd.GetCmd().Environ() {
			if value, found := strings.CutPrefix(entry, "GIT_ALTERNATE_OBJECT_DIRECTORIES="); found {
				inherited = value
			}
		}
		if inherited != "" {
			alternates += string(os.PathListSeparator) + inherited
		}
		return cmd.RemoveEnvVar("GIT_OBJECT_DIRECTORY").RemoveEnvVar("GIT_ALTERNATE_OBJECT_DIRECTORIES").
			AddEnvVars("GIT_OBJECT_DIRECTORY="+store, "GIT_ALTERNATE_OBJECT_DIRECTORIES="+alternates).
			SetWd(self.repoPaths.WorktreePath()).DontLog().DisableLockRetries()
	}
	args := NewGitCmd("merge-tree").Arg("--write-tree", "--no-messages", "-z").
		ArgIf(opts.AllowUnrelatedHistories, "--allow-unrelated-histories")
	for _, option := range options {
		args.Arg("--strategy-option=" + option)
	}
	args.Arg(target.HeadOID()+"^{commit}", target.SourceOID()+"^{commit}")
	stdout, _, err := objectEnvironment(self.cmd.New(args.ToArgv())).RunWithOutputs()
	if err != nil && !commandExitedWithOne(err) {
		return nil, fmt.Errorf("Git could not produce the required merge preview: %w", err)
	}
	// Exit 1 denotes conflicts, not a failed preview. Its result tree still
	// describes checkout paths; looking only at conflict names would miss data.
	tree, parseErr := mergePreviewTreeID(stdout, err != nil)
	if parseErr != nil {
		return nil, errors.Join(parseErr, err)
	}
	paths, err := self.branchActionTreePaths(tree, objectEnvironment)
	if err != nil {
		return nil, err
	}
	if err := self.CheckBranchActionTarget(target); err != nil {
		return nil, err
	}
	return paths, nil
}

// Validate the complete machine-readable response, not just its first hash.
// A configured driver can print to stdout; accepting its hash before Git's
// real result would produce an incomplete overwrite-risk prediction.
func mergePreviewTreeID(output string, conflicted bool) (string, error) {
	tree, rest, terminated := strings.Cut(output, "\x00")
	if !terminated || !IsFullObjectID(tree) {
		return "", fmt.Errorf("Git returned an invalid merge preview tree")
	}
	if rest != "" && !conflicted {
		return "", fmt.Errorf("unexpected output after a clean merge preview tree")
	}
	for rest != "" {
		entry, remaining, terminated := strings.Cut(rest, "\x00")
		if !terminated {
			return "", fmt.Errorf("Git returned unterminated merge preview conflict information")
		}
		prefix, path, hasPath := strings.Cut(entry, "\t")
		fields := strings.Fields(prefix)
		if !hasPath || path == "" || len(fields) != 3 {
			return "", fmt.Errorf("Git returned invalid merge preview conflict information")
		}
		_, modeErr := strconv.ParseUint(fields[0], 8, 32)
		if modeErr != nil || len(fields[1]) != len(tree) || !IsFullObjectID(fields[1]) ||
			(fields[2] != "1" && fields[2] != "2" && fields[2] != "3") {
			return "", fmt.Errorf("Git returned invalid merge preview conflict information")
		}
		rest = remaining
	}
	return tree, nil
}

func (self *BranchCommands) branchActionTreePaths(tree string, environment func(*oscommands.CmdObj) *oscommands.CmdObj) ([]string, error) {
	cmd := self.cmd.New(NewGitCmd("ls-tree").Arg("-r", "-z", "--name-only", tree).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog()
	if environment != nil {
		cmd = environment(cmd)
	}
	stdout, _, err := cmd.RunWithOutputs()
	if err != nil {
		return nil, err
	}
	if stdout != "" && !strings.HasSuffix(stdout, "\x00") {
		return nil, fmt.Errorf("Git returned unterminated merge preview paths")
	}
	return nulPaths([]byte(stdout)), nil
}

// Git's alternate-path list accepts C-quoted entries. Encode bytes with octal
// escapes rather than Go's Unicode escapes, which Git's C decoder rejects.
// Quoting also protects path-list separators and Windows drive letters.
func quoteMergeAlternate(path string) string {
	var result strings.Builder
	result.WriteByte('"')
	for i := range len(path) {
		b := path[i]
		switch {
		case b == '\\' || b == '"':
			result.WriteByte('\\')
			result.WriteByte(b)
		case b < ' ' || b >= 0x7f:
			fmt.Fprintf(&result, "\\%03o", b)
		default:
			result.WriteByte(b)
		}
	}
	result.WriteByte('"')
	return result.String()
}
