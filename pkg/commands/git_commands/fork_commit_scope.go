package git_commands

import (
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: the commit scope filter shows the selected commits with their ancestors,
// or with their descendants (commits reachable from any ref that have a selected
// commit as an ancestor). A branch scope shows a local branch and its upstream
// with their ancestors.
//
// Integration points: GetCommits loads a scoped list with loadCommitScope instead
// of loadCommits, and getLogCmd takes its revisions, order, --all and limit from
// commitScopeLog. Upstream lists never set ScopeHashes or ScopeBranch, so they keep
// their path.

// hasCommitScope reports whether the list is limited to a commit scope.
func (opts GetCommitsOptions) hasCommitScope() bool {
	return len(opts.ScopeHashes) > 0 || opts.ScopeBranch != ""
}

// commitScopeLogArgs holds the parts of the git log arguments that a commit
// scope can change.
type commitScopeLogArgs struct {
	revisions []string
	order     string
	all       bool
	limit     bool
}

// commitScopeLog returns the git log arguments for opts. refSpec is the revision
// spec that an unscoped list walks.
func commitScopeLog(opts GetCommitsOptions, refSpec string, configuredOrder string) commitScopeLogArgs {
	if !opts.hasCommitScope() {
		return commitScopeLogArgs{revisions: []string{refSpec}, order: configuredOrder, all: opts.All, limit: opts.Limit}
	}
	if !opts.ScopeDescendants {
		// Ancestors: the selected commits and everything they can reach. This
		// replaces the ref, so --all does not apply, but the normal limit and
		// order do.
		return commitScopeLogArgs{revisions: opts.ScopeHashes, order: configuredOrder, limit: opts.Limit}
	}
	// Descendants: every ref, with no limit, so that no descendant is missed. The
	// order must keep children before their parents, which topo-order guarantees.
	// The date orders already do, so only the default order needs changing.
	order := configuredOrder
	if order == "default" {
		order = "topo-order"
	}
	return commitScopeLogArgs{order: order, all: true}
}

// loadCommitScope loads the commits of a commit scope with parseLine, as
// loadCommits does for unscoped lists. Selected hashes that no longer exist are
// dropped first, so that a vanished commit cannot make the descendants search
// read the whole history, and an ancestors log cannot fail on it.
func (self *CommitLoader) loadCommitScope(opts GetCommitsOptions, parseLine func(string) (*models.Commit, bool)) ([]*models.Commit, error) {
	if opts.ScopeBranch != "" {
		return self.loadBranchScope(opts, parseLine)
	}
	hashes, err := self.existingCommitScopeHashes(opts.ScopeHashes)
	if err != nil {
		return nil, err
	}
	if len(hashes) == 0 {
		return []*models.Commit{}, nil
	}

	scoped := opts
	scoped.ScopeHashes = hashes
	if !opts.ScopeDescendants {
		return loadCommits(self.getLogCmd(scoped), "", parseLine)
	}

	selected := make(map[string]bool, len(hashes))
	for _, hash := range hashes {
		selected[hash] = true
	}
	remaining := len(selected)
	// Children come before their parents, so once the last selected commit has
	// been read, no later commit can be a descendant of a selected one. Stopping
	// is done by the line after that commit: returning stop drops the line it is
	// returned for, which is what we want, since that line is not needed.
	untilLastSelected := func(line string) (*models.Commit, bool) {
		if remaining == 0 {
			return nil, true
		}
		commit, stop := parseLine(line)
		if commit != nil && selected[commit.Hash()] {
			remaining--
		}
		return commit, stop
	}
	commits, err := loadCommits(self.getLogCmd(scoped), "", untilLastSelected)
	if err != nil {
		return nil, err
	}
	return keepCommitDescendants(commits, selected), nil
}

// loadBranchScope loads the commits of a branch scope: the local branch and its
// upstream, with their ancestors, in the normal order and limit. The branch is
// resolved on every load, so commits fetched into the upstream appear. A branch
// that no longer exists gives an empty list.
//
// The resolved refs are passed on as the revisions of an ancestors scope, which
// takes ref names as readily as hashes.
func (self *CommitLoader) loadBranchScope(opts GetCommitsOptions, parseLine func(string) (*models.Commit, bool)) ([]*models.Commit, error) {
	branchRef := "refs/heads/" + opts.ScopeBranch
	if err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", "--quiet", branchRef).ToArgv()).DontLog().Run(); err != nil {
		// The branch is gone (or the lookup failed, which the log below would
		// report too), so there is nothing to show.
		return []*models.Commit{}, nil
	}

	revisions := []string{branchRef}
	// The upstream is optional: rev-parse --quiet exits non-zero when there is none,
	// or when it does not resolve. Either way the branch alone is shown. The
	// @{upstream} suffix takes the short branch name; the full ref name does not work.
	if upstream, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", "--quiet", "--symbolic-full-name", opts.ScopeBranch+"@{upstream}").ToArgv()).DontLog().RunWithOutput(); err == nil {
		if upstream = strings.TrimSpace(upstream); upstream != "" {
			revisions = append(revisions, upstream)
		}
	}

	scoped := opts
	scoped.ScopeBranch = ""
	scoped.ScopeHashes = revisions
	scoped.ScopeDescendants = false
	return loadCommits(self.getLogCmd(scoped), "", parseLine)
}

// existingCommitScopeHashes returns the hashes that name a commit in the
// repository, without duplicates. Hashes that are missing are left out.
func (self *CommitLoader) existingCommitScopeHashes(hashes []string) ([]string, error) {
	for _, hash := range hashes {
		if err := validateObjectID(hash); err != nil {
			return nil, err
		}
	}
	output, err := self.cmd.New(NewGitCmd("rev-list").
		Arg("--no-walk", "--ignore-missing").
		Arg(hashes...).
		ToArgv()).
		DontLog().
		RunWithOutput()
	if err != nil {
		return nil, err
	}
	existing := []string{}
	seen := map[string]bool{}
	for _, hash := range utils.SplitLines(output) {
		if !seen[hash] {
			seen[hash] = true
			existing = append(existing, hash)
		}
	}
	return existing, nil
}

// keepCommitDescendants returns the commits that are selected or have a kept
// parent, in their original order. commits must be in topo-order, with children
// before their parents. The pass runs backwards, so that every parent's decision
// is made before the commit that refers to it.
func keepCommitDescendants(commits []*models.Commit, selected map[string]bool) []*models.Commit {
	kept := make(map[string]bool, len(commits))
	keep := make([]bool, len(commits))
	for i := len(commits) - 1; i >= 0; i-- {
		commit := commits[i]
		if selected[commit.Hash()] || slices.ContainsFunc(commit.Parents(), func(parent string) bool { return kept[parent] }) {
			keep[i] = true
			kept[commit.Hash()] = true
		}
	}

	result := make([]*models.Commit, 0, len(kept))
	for i, commit := range commits {
		if keep[i] {
			result = append(result, commit)
		}
	}
	return result
}
