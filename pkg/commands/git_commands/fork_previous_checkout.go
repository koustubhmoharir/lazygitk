package git_commands

import (
	"regexp"
	"strings"
)

// Fork: s p returns to the previous checkout. Upstream reads Git's @{-1}, which
// is ambiguous when a branch and a tag share the previous branch's name: Git then
// prints no ref, and an empty answer used to mean a detached commit. This reads
// the source of the newest checkout entry in HEAD's reflog instead, the same
// entry the s p menu label names (forkCheckoutReflogPattern in the controllers).

// PreviousCheckoutKind says what s p returns to.
type PreviousCheckoutKind uint8

const (
	// PreviousCheckoutNone means HEAD's reflog records no earlier checkout.
	PreviousCheckoutNone PreviousCheckoutKind = iota
	// PreviousCheckoutBranch means the recorded source is an existing local
	// branch. FullRef names it.
	PreviousCheckoutBranch
	// PreviousCheckoutCommit means HEAD was detached at the recorded commit. OID
	// names it.
	PreviousCheckoutCommit
	// PreviousCheckoutGone means the recorded source is neither an existing
	// local branch nor a commit, such as a branch that has since been deleted.
	PreviousCheckoutGone
)

// PreviousCheckoutTarget is the result of ClassifyPreviousCheckout. Source is the
// name the reflog recorded.
type PreviousCheckoutTarget struct {
	Kind    PreviousCheckoutKind
	Source  string
	FullRef string
	OID     string
}

var previousCheckoutReflogPattern = regexp.MustCompile(`^checkout: moving from (\S+) to \S+`)

// ClassifyPreviousCheckout reads the newest "checkout: moving from" entry of
// HEAD's reflog and classifies its source. A full object ID source was a
// detached HEAD and must name a commit. Any other source must still be a local
// branch. Nothing is checked out here.
func (self *BranchCommands) ClassifyPreviousCheckout() (PreviousCheckoutTarget, error) {
	output, err := self.cmd.New(NewGitCmd("log").
		Arg("-g", "-E", "--grep-reflog=^checkout: moving from ", "-n1", "--format=%gs", "HEAD").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return PreviousCheckoutTarget{}, err
	}
	match := previousCheckoutReflogPattern.FindStringSubmatch(trimTerminalLineEnding(output))
	if match == nil {
		return PreviousCheckoutTarget{Kind: PreviousCheckoutNone}, nil
	}
	source := match[1]
	if IsFullObjectID(source) {
		if err := self.validateLiteralCommitOID(source); err != nil {
			return PreviousCheckoutTarget{}, err
		}
		return PreviousCheckoutTarget{Kind: PreviousCheckoutCommit, Source: source, OID: strings.ToLower(source)}, nil
	}
	ref := "refs/heads/" + source
	state, err := self.lookupCheckoutByNameRef(ref)
	if err != nil {
		return PreviousCheckoutTarget{}, err
	}
	if state.exists && state.symbolic == "" {
		return PreviousCheckoutTarget{Kind: PreviousCheckoutBranch, Source: source, FullRef: ref}, nil
	}
	return PreviousCheckoutTarget{Kind: PreviousCheckoutGone, Source: source}, nil
}
