package git_commands

import (
	"fmt"
	"strings"
)

// Fork: the palette action "Checkout by name" classifies its typed input here,
// on a worker, before the UI thread dispatches it to a guarded checkout flow.
// Upstream's CheckoutRef and CheckoutRemoteBranch run plain `git checkout
// <input>`, which can overwrite ignored files, treat a non-ref as a path and
// offer autostash. The input is therefore only ever read as literal refs or
// as one commit name, and is never passed to checkout.

type CheckoutByNameKind uint8

const (
	// CheckoutByNameNotFound means no branch, remote branch, tag or commit
	// matches. For a plain name the user may create a branch with that name.
	CheckoutByNameNotFound CheckoutByNameKind = iota
	// CheckoutByNamePrevious is "-": the previous checkout, as s p does.
	CheckoutByNamePrevious
	// CheckoutByNameLocalBranch is refs/heads/<name>. A local branch wins over a
	// tag or remote branch with the same name.
	CheckoutByNameLocalBranch
	// CheckoutByNameRemoteBranch is refs/remotes/<name>.
	CheckoutByNameRemoteBranch
	// CheckoutByNameTag is refs/tags/<name>.
	CheckoutByNameTag
	// CheckoutByNameCommit is a commit named by an object ID or a revision that
	// peels to a commit; the checkout is detached at OID.
	CheckoutByNameCommit
	// CheckoutByNameAmbiguous means the name matches more than one ref, or a ref
	// that Git would also consider. Refs lists the full ref names.
	CheckoutByNameAmbiguous
)

// CheckoutByNameTarget is the result of ClassifyCheckoutByName. FullRef is set
// for the branch, remote branch and tag kinds, OID for the commit kind, and
// Refs for the ambiguous kind.
type CheckoutByNameTarget struct {
	Kind    CheckoutByNameKind
	FullRef string
	OID     string
	Refs    []string
}

// ClassifyCheckoutByName classifies the trimmed input against the repository's
// current refs, in this order:
//  1. "-" is the previous checkout.
//  2. A full ref (refs/heads/, refs/remotes/ or refs/tags/) is that exact ref.
//  3. refs/heads/<input> is a local branch.
//  4. refs/remotes/<input> is a remote branch.
//  5. refs/tags/<input> is a tag.
//  6. Otherwise the input is a commit if rev-parse resolves it to one.
//
// Rules 4 and 5 both matching is ambiguous. So is a name that Git would also
// resolve to another ref (refs/<input> or refs/remotes/<input>/HEAD). Nothing
// is ever checked out here; the caller dispatches on the result.
func (self *BranchCommands) ClassifyCheckoutByName(input string) (CheckoutByNameTarget, error) {
	input = strings.TrimSpace(input)
	if input == "-" {
		return CheckoutByNameTarget{Kind: CheckoutByNamePrevious}, nil
	}

	if strings.HasPrefix(input, "refs/") {
		kind, ok := checkoutByNameFullRefKind(input)
		if !ok {
			return CheckoutByNameTarget{Kind: CheckoutByNameNotFound, FullRef: input}, nil
		}
		state, err := self.lookupCheckoutByNameRef(input)
		if err != nil {
			return CheckoutByNameTarget{}, err
		}
		if !state.exists {
			return CheckoutByNameTarget{Kind: CheckoutByNameNotFound, FullRef: input}, nil
		}
		if err := checkoutByNameDirectRef(input, state); err != nil {
			return CheckoutByNameTarget{}, err
		}
		return CheckoutByNameTarget{Kind: kind, FullRef: input}, nil
	}

	localRef := "refs/heads/" + input
	local, err := self.lookupCheckoutByNameRef(localRef)
	if err != nil {
		return CheckoutByNameTarget{}, err
	}
	if local.exists {
		if err := checkoutByNameDirectRef(localRef, local); err != nil {
			return CheckoutByNameTarget{}, err
		}
		return CheckoutByNameTarget{Kind: CheckoutByNameLocalBranch, FullRef: localRef}, nil
	}

	remoteRef, tagRef := "refs/remotes/"+input, "refs/tags/"+input
	remote, err := self.lookupCheckoutByNameRef(remoteRef)
	if err != nil {
		return CheckoutByNameTarget{}, err
	}
	tag, err := self.lookupCheckoutByNameRef(tagRef)
	if err != nil {
		return CheckoutByNameTarget{}, err
	}
	if remote.exists && tag.exists {
		return CheckoutByNameTarget{Kind: CheckoutByNameAmbiguous, Refs: []string{remoteRef, tagRef}}, nil
	}
	// Git's own DWIM would also consider these, so refuse rather than let rev-parse
	// pick one of them.
	dwim := []string{"refs/" + input, "refs/remotes/" + input + "/HEAD"}
	var dwimMatches []string
	for _, ref := range dwim {
		state, err := self.lookupCheckoutByNameRef(ref)
		if err != nil {
			return CheckoutByNameTarget{}, err
		}
		if state.exists {
			dwimMatches = append(dwimMatches, ref)
		}
	}
	if len(dwimMatches) > 0 {
		return CheckoutByNameTarget{Kind: CheckoutByNameAmbiguous, Refs: dwimMatches}, nil
	}
	if remote.exists {
		if err := checkoutByNameDirectRef(remoteRef, remote); err != nil {
			return CheckoutByNameTarget{}, err
		}
		return CheckoutByNameTarget{Kind: CheckoutByNameRemoteBranch, FullRef: remoteRef}, nil
	}
	if tag.exists {
		if err := checkoutByNameDirectRef(tagRef, tag); err != nil {
			return CheckoutByNameTarget{}, err
		}
		return CheckoutByNameTarget{Kind: CheckoutByNameTag, FullRef: tagRef}, nil
	}

	oid, found, err := self.resolveCheckoutByNameCommit(input)
	if err != nil {
		return CheckoutByNameTarget{}, err
	}
	if !found {
		return CheckoutByNameTarget{Kind: CheckoutByNameNotFound}, nil
	}
	return CheckoutByNameTarget{Kind: CheckoutByNameCommit, OID: oid}, nil
}

// lookupCheckoutByNameRef reads an exact ref. A name that Git cannot store as a
// ref, such as one with a space, cannot match any ref, so it is not looked up:
// symbolic-ref would fail on it rather than report it missing.
func (self *BranchCommands) lookupCheckoutByNameRef(ref string) (branchPlacementRef, error) {
	_, err := self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		if commandExitedWithOne(err) {
			return branchPlacementRef{}, nil
		}
		return branchPlacementRef{}, err
	}
	return self.readPlacementRef(ref)
}

func checkoutByNameFullRefKind(ref string) (CheckoutByNameKind, bool) {
	switch {
	case strings.HasPrefix(ref, "refs/heads/") && len(ref) > len("refs/heads/"):
		return CheckoutByNameLocalBranch, true
	case strings.HasPrefix(ref, "refs/remotes/") && len(ref) > len("refs/remotes/"):
		return CheckoutByNameRemoteBranch, true
	case strings.HasPrefix(ref, "refs/tags/") && len(ref) > len("refs/tags/"):
		return CheckoutByNameTag, true
	}
	return 0, false
}

// checkoutByNameDirectRef refuses a symbolic ref such as origin/HEAD. Its
// target is a different ref, and the checkout flows need a direct ref.
func checkoutByNameDirectRef(ref string, state branchPlacementRef) error {
	if state.symbolic != "" {
		return fmt.Errorf("ref %q is symbolic; choose its direct target", ref)
	}
	return nil
}

// resolveCheckoutByNameCommit peels the input to a commit with rev-parse. The
// input goes after --end-of-options, so it cannot be read as an option.
func (self *BranchCommands) resolveCheckoutByNameCommit(input string) (string, bool, error) {
	output, err := self.cmd.New(NewGitCmd("rev-parse").
		Arg("--verify", "--quiet", "--end-of-options", input+"^{commit}").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		if commandExitedWithOne(err) {
			return "", false, nil
		}
		return "", false, err
	}
	oid := strings.ToLower(strings.TrimSpace(output))
	if err := validateObjectID(oid); err != nil {
		return "", false, fmt.Errorf("git returned invalid commit object ID: %w", err)
	}
	return oid, true, nil
}
