package filtering

import "slices"

// Fork: a commit scope shows the commits in Hashes together with their ancestors,
// or, when Descendants is set, together with their descendants. It is a third kind
// of filtering mode next to path and author: setting one replaces the others.
type CommitScope struct {
	Hashes      []string
	Descendants bool
}

// Active reports whether a scope is set. A scope always has at least one hash.
func (s CommitScope) Active() bool {
	return len(s.Hashes) > 0
}

// SetCommitScope sets the scope. The mode keeps its own copy of the hashes.
func (m *Filtering) SetCommitScope(scope CommitScope) {
	m.commitScope = CommitScope{Hashes: slices.Clone(scope.Hashes), Descendants: scope.Descendants}
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
