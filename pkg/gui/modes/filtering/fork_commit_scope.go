package filtering

import "slices"

// Fork: a commit scope shows a set of commits and their history. It is a third kind
// of filtering mode next to path and author: setting one replaces the others.
//
// Hashes with Descendants unset show the commits in Hashes together with their
// ancestors; with Descendants set, together with their descendants. Ref, when set,
// shows that ref's commit with its ancestors: a local branch also shows its upstream,
// and a remote branch or tag shows alone. Ref is a full ref name (refs/heads/...,
// refs/remotes/... or refs/tags/...), and it is resolved on every load, so that the
// scope follows the ref when it moves or is fetched.
type CommitScope struct {
	Hashes      []string
	Descendants bool
	Ref         string
}

// Active reports whether a scope is set. A scope has either hashes or a ref.
func (s CommitScope) Active() bool {
	return len(s.Hashes) > 0 || s.Ref != ""
}

// SetCommitScope sets the scope. The mode keeps its own copy of the hashes.
func (m *Filtering) SetCommitScope(scope CommitScope) {
	m.commitScope = CommitScope{Hashes: slices.Clone(scope.Hashes), Descendants: scope.Descendants, Ref: scope.Ref}
}

// GetCommitScope returns the scope, which is empty when none is set.
func (m *Filtering) GetCommitScope() CommitScope {
	return m.commitScope
}

// HasCommitScope reports whether a commit scope is set.
func (m *Filtering) HasCommitScope() bool {
	return m.commitScope.Active()
}

// HasPathOrAuthorFilter reports whether the path or author filter is set. Only
// these filters apply to the reflog and the graph; a commit scope does not.
func (m *Filtering) HasPathOrAuthorFilter() bool {
	return m.path != "" || m.author != ""
}
