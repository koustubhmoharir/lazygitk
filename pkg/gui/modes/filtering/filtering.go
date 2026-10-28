package filtering

type Filtering struct {
	path               string      // the filename that gets passed to git log
	author             string      // the author that gets passed to git log
	selectedCommitHash string      // the commit that was selected before we entered filtering mode
	commitScope        CommitScope // Fork: replaces path and author (see fork_commit_scope.go)
}

func New(path string, author string) Filtering {
	return Filtering{path: path, author: author}
}

func (m *Filtering) Active() bool {
	// Fork: a commit scope filter also counts, so that the filter-mode restrictions apply to it.
	return m.path != "" || m.author != "" || m.commitScope.Active()
}

func (m *Filtering) Reset() {
	m.path = ""
	m.author = ""
	m.commitScope = CommitScope{}
}

func (m *Filtering) SetPath(path string) {
	m.path = path
}

func (m *Filtering) GetPath() string {
	return m.path
}

func (m *Filtering) SetAuthor(author string) {
	m.author = author
}

func (m *Filtering) GetAuthor() string {
	return m.author
}

func (m *Filtering) SetSelectedCommitHash(hash string) {
	m.selectedCommitHash = hash
}

func (m *Filtering) GetSelectedCommitHash() string {
	return m.selectedCommitHash
}
