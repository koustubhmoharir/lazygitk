package git_commands

import (
	"regexp"
	"strings"
)

// Fork: undo and redo of a checkout (undo_controller.go) read the reflog's
// "from" or "to" text. That text is a branch name or the commit ID as the user
// typed it. This classifies it without guessing: an existing local branch
// wins, as the reflog wording does; otherwise a hexadecimal ID must name one
// commit. Nothing is checked out here.

// ReflogCheckoutKind says what an undo or redo of a checkout returns to.
type ReflogCheckoutKind uint8

const (
	// ReflogCheckoutUnknown means the text is neither a local branch nor a
	// hexadecimal ID naming one commit, so undo and redo refuse it.
	ReflogCheckoutUnknown ReflogCheckoutKind = iota
	// ReflogCheckoutBranch means the text names an existing local branch.
	// FullRef names it.
	ReflogCheckoutBranch
	// ReflogCheckoutCommit means the text is an ID of one commit. OID names it,
	// in full.
	ReflogCheckoutCommit
)

// ReflogCheckoutTarget is the result of ClassifyReflogCheckoutTarget.
type ReflogCheckoutTarget struct {
	Kind    ReflogCheckoutKind
	FullRef string
	OID     string
}

var abbreviatedObjectID = regexp.MustCompile(`^[0-9a-fA-F]{4,64}$`)

// ClassifyReflogCheckoutTarget classifies target as a local branch or a commit.
// An abbreviated ID is accepted only when Git resolves it to a commit whose full
// ID begins with it, so a same-named tag or an ambiguous prefix is refused.
func (self *BranchCommands) ClassifyReflogCheckoutTarget(target string) (ReflogCheckoutTarget, error) {
	ref := "refs/heads/" + target
	state, err := self.lookupCheckoutByNameRef(ref)
	if err != nil {
		return ReflogCheckoutTarget{}, err
	}
	if state.exists {
		return ReflogCheckoutTarget{Kind: ReflogCheckoutBranch, FullRef: ref}, nil
	}
	if !abbreviatedObjectID.MatchString(target) {
		return ReflogCheckoutTarget{Kind: ReflogCheckoutUnknown}, nil
	}
	output, err := self.cmd.New(NewGitCmd("rev-parse").
		Arg("--verify", "--quiet", target+"^{commit}").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		// Git cannot resolve this to one commit (absent or ambiguous): refuse.
		return ReflogCheckoutTarget{Kind: ReflogCheckoutUnknown}, nil
	}
	oid := strings.ToLower(trimTerminalLineEnding(output))
	if !IsFullObjectID(oid) || !strings.HasPrefix(oid, strings.ToLower(target)) {
		return ReflogCheckoutTarget{Kind: ReflogCheckoutUnknown}, nil
	}
	if err := self.validateLiteralCommitOID(oid); err != nil {
		return ReflogCheckoutTarget{}, err
	}
	return ReflogCheckoutTarget{Kind: ReflogCheckoutCommit, OID: oid}, nil
}
