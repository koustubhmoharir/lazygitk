package git_commands

import "strings"

// Fork: the Commits selection can hold commits that the loaded list does not show: a
// filter or the commit limit hides them, or a rewrite replaced them. After a refresh the
// GUI asks which of them still matter. This is the question, answered by Git.

// maxClassifiedCommitSetHashes bounds ClassifySelectedCommits. A selection with more
// absent commits than this is not a few rewrites, and classifying each would cost one
// Git call per commit, so nothing is classified past it: the caller shows no notice and
// the selection keeps its hashes as they are.
const maxClassifiedCommitSetHashes = 50

// ClassifySelectedCommits sorts the hashes of selected commits that the loaded Commits
// list does not show. missing are the hashes whose objects no longer exist. unreachable
// are those whose objects exist but that neither HEAD nor any ref (refs/*) reaches,
// which is what a rewrite outside the selection leaves behind. Every other hash is
// reachable, so the list only hides it, and it is in neither result. Each hash must be a
// full object ID. More than maxClassifiedCommitSetHashes hashes give nil results.
func (self *CommitCommands) ClassifySelectedCommits(hashes []string) (missing []string, unreachable []string, err error) {
	if len(hashes) == 0 || len(hashes) > maxClassifiedCommitSetHashes {
		return nil, nil, nil
	}
	for _, hash := range hashes {
		if err := validateObjectID(hash); err != nil {
			return nil, nil, err
		}
	}

	// One rev-list prints the objects that exist. --ignore-missing drops the others, so
	// a hash that is not printed is missing.
	output, err := self.cmd.New(NewGitCmd("rev-list").
		Arg("--no-walk", "--ignore-missing").
		Arg(hashes...).
		Arg("--").
		ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		return nil, nil, err
	}
	present := make(map[string]bool, len(hashes))
	for _, line := range strings.Fields(output) {
		present[strings.ToLower(line)] = true
	}

	for _, hash := range hashes {
		if !present[strings.ToLower(hash)] {
			missing = append(missing, hash)
			continue
		}
		fromHead, err := self.reachesCommit("HEAD", hash)
		if err != nil {
			return nil, nil, err
		}
		if fromHead {
			continue
		}
		refs, err := self.cmd.New(NewGitCmd("for-each-ref").
			Arg("--contains="+hash, "--count=1", "--format=%(refname)").
			ToArgv()).DontLog().RunWithOutput()
		if err != nil {
			return nil, nil, err
		}
		if strings.TrimSpace(refs) == "" {
			unreachable = append(unreachable, hash)
		}
	}
	return missing, unreachable, nil
}
