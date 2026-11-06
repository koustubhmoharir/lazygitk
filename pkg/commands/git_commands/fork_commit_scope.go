package git_commands

import (
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
// A descendant scope reads the walk only as far as its selected commits, and parses
// only the rows it keeps (see loadCommitDescendants).
//
// Integration points: GetCommits loads a scoped list with loadCommitScope instead
// of loadCommits, and getLogCmd takes its revisions, order, --all and limit from
// commitScopeLog. Upstream lists never set ScopeHashes or ScopeRef, so they keep
// their path.

// commitScopeLimit is the number of commits in a limited list. getLogCmd applies
// it to unscoped lists with -300; a scoped ancestors list applies it while it reads
// the walk, because the walk has no git limit (see commitAncestorFilter); a
// descendant scope keeps its newest rows, which are the first ones it finds in its walk (see
// loadCommitDescendants).
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
		// date-order, so the walk can stop after its selected commits (see
		// loadCommitDescendants). It has no limit: the rows are limited there.
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
// dropped first, so that an ancestors log cannot fail on one.
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

	return self.loadCommitDescendants(scoped, parseLine)
}

// loadCommitDescendants loads the selected commits and their descendants, newest
// first. Only commits reachable from a ref can be descendants of a reachable
// selected commit, so the selected commits are first split into reachable and not.
// The walk runs newest first and stops once every reachable selected commit has been
// read: their descendants come before them, so no later line can be one. The lines
// read are then decided from the oldest, where parents come first (see descendantWalk),
// and only the rows kept are parsed into commits.
func (self *CommitLoader) loadCommitDescendants(opts GetCommitsOptions, parseLine func(string) (*models.Commit, bool)) ([]*models.Commit, error) {
	reachable, err := self.reachableFromAll(opts.ScopeHashes)
	if err != nil {
		return nil, err
	}
	if len(reachable) == 0 {
		return []*models.Commit{}, nil
	}

	walk := newDescendantWalk(opts.ScopeHashes, reachable)
	streamLine := func(line string) (*models.Commit, bool) {
		return nil, !walk.read(line)
	}
	// loadCommits returns nothing here: the lines are held by walk.
	if _, err := loadCommits(self.getLogCmd(opts), "", streamLine); err != nil {
		return nil, err
	}
	return walk.rows(opts.Limit, func(line string) *models.Commit {
		commit, _ := parseLine(line)
		return commit
	}), nil
}

// reachableFromAll returns the hashes, of those given, that a walk of --all reaches. The
// walk is streamed and stops once every hash has been seen, so a recent selection costs
// little. A hash that is never seen is unreachable. The check does not use
// `rev-list <hash> --not --all`: its limit walk trusts commit dates, and clock skew makes
// it wrong.
func (self *CommitLoader) reachableFromAll(hashes []string) ([]string, error) {
	wanted := make(map[string]bool, len(hashes))
	for _, hash := range hashes {
		wanted[hash] = true
	}
	seen := make(map[string]bool, len(hashes))
	err := self.cmd.New(NewGitCmd("rev-list").Arg("--all").ToArgv()).DontLog().RunAndProcessLines(func(line string) (bool, error) {
		if wanted[line] {
			seen[line] = true
		}
		return len(seen) == len(wanted), nil
	})
	if err != nil {
		return nil, err
	}
	reachable := []string{}
	for _, hash := range hashes {
		if seen[hash] {
			reachable = append(reachable, hash)
		}
	}
	return reachable, nil
}

// parseHashAndParents reads the hash and parents of a line of a log in prettyFormat,
// without its leading '+'. It reads no other field.
func parseHashAndParents(line string) (string, []string, bool) {
	fields := strings.SplitN(line, "\x00", 6)
	if len(fields) < 5 {
		return "", nil, false
	}
	return fields[0], strings.Fields(fields[4]), true
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

// descendantWalk holds the lines of a walk of --all, newest first, read up to the
// point where every reachable selected commit has been read. The walk holds the raw
// lines and their hashes and parents, not parsed commits: rows are parsed only when
// they are kept.
type descendantWalk struct {
	// selected holds every selected commit, including unreachable ones, which are
	// kept when they are seen, though no walk shows them.
	selected map[string]bool
	// awaited holds the reachable selected commits that the walk has not read yet.
	awaited map[string]bool
	lines   []descendantLine
}

type descendantLine struct {
	hash    string
	parents []string
	line    string
}

// newDescendantWalk returns a walk for the selected hashes. reachable is the subset
// of selected that a walk of --all reaches; the walk stops after the last of them.
func newDescendantWalk(selected []string, reachable []string) *descendantWalk {
	walk := &descendantWalk{
		selected: make(map[string]bool, len(selected)),
		awaited:  make(map[string]bool, len(reachable)),
	}
	for _, hash := range selected {
		walk.selected[hash] = true
	}
	for _, hash := range reachable {
		walk.awaited[hash] = true
	}
	return walk
}

// read records the next line of the walk. It returns false when the walk can stop
// before this line, because no later line can be a descendant of a selected commit.
// Lines that are not commits are skipped.
func (self *descendantWalk) read(line string) bool {
	if len(self.awaited) == 0 {
		return false
	}
	hash, parents, ok := parseHashAndParents(line)
	if !ok {
		return true
	}
	delete(self.awaited, hash)
	self.lines = append(self.lines, descendantLine{hash: hash, parents: parents, line: line})
	return true
}

// rows returns the rows that are kept, newest first: a commit is kept if it is selected or
// one of its parents is kept. The decision runs from the oldest line, where parents come
// first. When limit is set, only the newest commitScopeLimit kept rows are parsed; they
// are the first kept rows of the walk, which is newest first.
func (self *descendantWalk) rows(limit bool, parse func(line string) *models.Commit) []*models.Commit {
	kept := make(map[string]bool, len(self.selected))
	for hash := range self.selected {
		kept[hash] = true
	}
	keep := make([]bool, len(self.lines))
	for i := len(self.lines) - 1; i >= 0; i-- {
		entry := self.lines[i]
		keepEntry := kept[entry.hash]
		for _, parent := range entry.parents {
			if kept[parent] {
				keepEntry = true
				break
			}
		}
		if keepEntry {
			kept[entry.hash] = true
			keep[i] = true
		}
	}

	result := []*models.Commit{}
	count := 0
	for i, entry := range self.lines {
		if !keep[i] {
			continue
		}
		if limit && count == commitScopeLimit {
			break
		}
		count++
		if commit := parse(entry.line); commit != nil {
			result = append(result, commit)
		}
	}
	return result
}
