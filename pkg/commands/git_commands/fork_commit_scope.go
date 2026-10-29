package git_commands

import (
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: the commit scope filter shows the selected commits with their ancestors,
// or with their descendants (commits reachable from any ref that have a selected
// commit as an ancestor). A ref scope shows a local branch and its upstream with
// their ancestors, or a remote branch or tag alone with its ancestors.
//
// A scope walks the same commits as the unfiltered list, in the same order, and
// drops the commits outside the scope. So the rows of a scope are a subsequence of
// the unfiltered rows, and commits that were adjacent in the unfiltered list stay
// adjacent. A walk that starts from the selected commits alone would group its
// rows by line of history instead.
//
// Integration points: GetCommits loads a scoped list with loadCommitScope instead
// of loadCommits, and getLogCmd takes its revisions, order, --all and limit from
// commitScopeLog. Upstream lists never set ScopeHashes or ScopeRef, so they keep
// their path.

// commitScopeLimit is the number of commits in a limited list. getLogCmd applies
// it to unscoped lists with -300; a scoped ancestors list applies it while it reads
// the walk, because the walk has no git limit (see commitAncestorFilter).
const commitScopeLimit = 300

// hasCommitScope reports whether the list is limited to a commit scope.
func (opts GetCommitsOptions) hasCommitScope() bool {
	return len(opts.ScopeHashes) > 0 || opts.ScopeRef != ""
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
// spec that an unscoped list walks, and configuredOrder is git.log.order.
func commitScopeLog(opts GetCommitsOptions, refSpec string, configuredOrder string) commitScopeLogArgs {
	if !opts.hasCommitScope() {
		return commitScopeLogArgs{revisions: []string{refSpec}, order: configuredOrder, all: opts.All, limit: opts.Limit}
	}
	order := scopeWalkOrder(configuredOrder)
	if opts.ScopeDescendants {
		// Descendants can be on any ref, so the walk is always --all, whatever the
		// unfiltered list shows. Children come before their parents in topo-order and
		// date-order, which keepCommitDescendants relies on. The walk has no limit,
		// so that no descendant is missed.
		return commitScopeLogArgs{revisions: []string{refSpec}, order: order, all: true}
	}
	// Ancestors: the walk is the unfiltered one. Its tips are the ones the unfiltered
	// list walks, plus the tips of a ref scope when the list is not --all, since
	// those need not be reachable from refSpec. Selected commits of a hash scope come
	// from the visible list, so they are reachable already.
	revisions := []string{refSpec}
	if !opts.All {
		revisions = append(revisions, opts.ScopeTips...)
	}
	return commitScopeLogArgs{revisions: revisions, order: order, all: opts.All}
}

// scopeWalkOrder returns the order of a scope walk. The default order is not
// guaranteed to list children before their parents when commit dates are out of
// order, and the scope filters need that, so the default walks with --date-order.
// The two agree when commit dates follow the history, which is the usual case.
func scopeWalkOrder(configuredOrder string) string {
	if configuredOrder == "default" {
		return "date-order"
	}
	return configuredOrder
}

// loadCommitScope loads the commits of a commit scope with parseLine, as
// loadCommits does for unscoped lists. Selected hashes that no longer exist are
// dropped first, so that a vanished commit cannot make the descendants search
// read the whole history, and an ancestors log cannot fail on it.
func (self *CommitLoader) loadCommitScope(opts GetCommitsOptions, parseLine func(string) (*models.Commit, bool)) ([]*models.Commit, error) {
	if opts.ScopeRef != "" {
		return self.loadRefScope(opts, parseLine)
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
		return self.loadCommitAncestors(scoped, parseLine)
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

// loadCommitAncestors loads the ancestors of opts.ScopeHashes from the unfiltered
// walk. The filter stops the walk as soon as the ancestors are complete or the
// limit is reached, so the rest of the history is not read.
func (self *CommitLoader) loadCommitAncestors(opts GetCommitsOptions, parseLine func(string) (*models.Commit, bool)) ([]*models.Commit, error) {
	filter := newCommitAncestorFilter(opts.ScopeHashes, opts.Limit)
	scopedParseLine := func(line string) (*models.Commit, bool) {
		// Stopping before the line of the next commit keeps the last kept commit, as
		// untilLastSelected does for descendants.
		if filter.done() {
			return nil, true
		}
		commit, stop := parseLine(line)
		if stop || commit == nil || !filter.keep(commit) {
			return nil, stop
		}
		return commit, false
	}
	return loadCommits(self.getLogCmd(opts), "", scopedParseLine)
}

// loadRefScope loads the commits of a ref scope with parseLine, in the normal
// order and limit. The ref is resolved on every load, so commits fetched into it
// appear, and a ref that no longer resolves gives an empty list. A local branch
// also shows its upstream; a remote branch or tag shows alone.
//
// The resolved commits are the selection of an ancestors scope, and they are also
// its tips, since the walk must start from them when it is not --all.
func (self *CommitLoader) loadRefScope(opts GetCommitsOptions, parseLine func(string) (*models.Commit, bool)) ([]*models.Commit, error) {
	// ^{commit} peels an annotated tag to its commit.
	refHash, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", "--quiet", opts.ScopeRef+"^{commit}").ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		// The ref is gone (or the lookup failed, which the log below would report
		// too), so there is nothing to show.
		return []*models.Commit{}, nil
	}

	tips := []string{strings.TrimSpace(refHash)}
	// The upstream is optional: rev-parse --quiet exits non-zero when there is none,
	// or when it does not resolve. Either way the branch alone is shown. The
	// @{upstream} suffix takes the short branch name; the full ref name does not work.
	if branch, isBranch := strings.CutPrefix(opts.ScopeRef, "refs/heads/"); isBranch {
		if upstream, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", "--quiet", branch+"@{upstream}").ToArgv()).DontLog().RunWithOutput(); err == nil {
			if upstream = strings.TrimSpace(upstream); upstream != "" {
				tips = append(tips, upstream)
			}
		}
	}

	scoped := opts
	scoped.ScopeRef = ""
	scoped.ScopeHashes = tips
	scoped.ScopeTips = tips
	scoped.ScopeDescendants = false
	return self.loadCommitAncestors(scoped, parseLine)
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

// commitAncestorFilter picks the ancestors of the selected commits out of a walk
// that lists children before their parents. A commit is kept when it is selected
// or is a parent of a kept commit. Because children come first, that is settled
// when the walk reaches the commit, so one pass decides every commit.
type commitAncestorFilter struct {
	// wanted holds the selected commits and the parents of kept commits that the
	// walk has not reached yet.
	wanted map[string]bool
	limit  bool
	kept   int
}

// newCommitAncestorFilter returns a filter for the ancestors of selected. When
// limit is set, the filter is done once commitScopeLimit commits are kept.
func newCommitAncestorFilter(selected []string, limit bool) *commitAncestorFilter {
	wanted := make(map[string]bool, len(selected))
	for _, hash := range selected {
		wanted[hash] = true
	}
	return &commitAncestorFilter{wanted: wanted, limit: limit}
}

// done reports whether the walk can stop before its next commit, because no
// commit that is still wanted can come after it, or the limit has been reached.
// A walk that stops early reads only the commits that are needed.
func (self *commitAncestorFilter) done() bool {
	return len(self.wanted) == 0 || (self.limit && self.kept >= commitScopeLimit)
}

// keep reports whether commit is in the scope, and if it is, it wants the parents
// of commit. Commits must be given in walk order, and only while !done().
func (self *commitAncestorFilter) keep(commit *models.Commit) bool {
	hash := commit.Hash()
	if !self.wanted[hash] {
		return false
	}
	delete(self.wanted, hash)
	for _, parent := range commit.Parents() {
		self.wanted[parent] = true
	}
	self.kept++
	return true
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
