package git_commands

import (
	"fmt"
	"strings"
)

// Fork: the flat `y` copy menu acts on the commit a branch or tag points at.
// These lookups run when the action is pressed, so they are never cached.

// ResolveRefCommitHash returns the full hash of the commit that fullRefName
// points at. Annotated tags are peeled to their commit. fullRefName must be a
// full ref name such as refs/tags/v1, so that a branch and a tag with the same
// short name cannot be confused.
func (self *CommitCommands) ResolveRefCommitHash(fullRefName string) (string, error) {
	output, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", "--end-of-options", fullRefName+"^{commit}").ToArgv()).RunWithOutput()
	if err != nil {
		return "", err
	}
	hash := strings.TrimSpace(output)
	if err := validateObjectID(hash); err != nil {
		return "", fmt.Errorf("git returned invalid commit object ID: %w", err)
	}
	return hash, nil
}

// TagsPointingAt returns the names of the tags on the commit with the given
// full hash, in the order git tag lists them (by ref name). Annotated tags
// count when they peel to that commit.
func (self *CommitCommands) TagsPointingAt(hash string) ([]string, error) {
	if err := validateObjectID(hash); err != nil {
		return nil, err
	}
	output, err := self.cmd.New(NewGitCmd("tag").Arg("--points-at", hash).ToArgv()).RunWithOutput()
	if err != nil {
		return nil, err
	}
	// Git does not allow whitespace in tag names.
	return strings.Fields(output), nil
}
