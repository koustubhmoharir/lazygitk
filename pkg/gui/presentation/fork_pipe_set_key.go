package presentation

import (
	"crypto/sha256"
	"encoding/hex"
	"unsafe"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/graph"
)

// Fork: upstream keys the pipe set cache by the first commit and the commit count,
// which is enough when the list is the history of one ref. The fork draws the graph
// for lists that can differ while those stay the same: the commit scope filters
// keep HEAD first and are cut at the same limit, and the whole-graph log gains
// commits on other branches after a fetch without HEAD moving. A list that reused
// another's pipes would draw its lanes in the wrong places, so the key is a digest
// of every hash in the list. The integration point is loadPipesets.
//
// Hashing every commit on each render is slow for a long history, and the graph is
// rendered on every selection change. So the digest is memoized per list. A commit
// list is published as a fresh slice and never changed in place, so a slice with the
// same first element, length and contents always has the same digest. The memo is
// keyed on the address of the first element and the length, which tells apart a
// list from a subslice that starts at the same commit. The memo keeps the slice
// itself, so its backing array cannot be freed and its address reused by another
// list while the entry exists. Only the last commitListMemoSize lists are kept, as
// a few lists (for example commits[rebaseOffset:] and the two divergence sections)
// are in use during one render. The memo is only accessed under the presentation
// mutex, like the pipe set cache.

const commitListMemoSize = 8

type commitListMemoEntry struct {
	// commits is kept so that the address in the key stays unique while the entry exists.
	commits []*models.Commit
	key     string
}

// commitListMemo holds the most recent entries, the oldest first.
var commitListMemo []commitListMemoEntry

// commitListKey returns a digest of the hashes of commits, in order. The caller must
// hold mutex.
func commitListKey(commits []*models.Commit) string {
	for _, entry := range commitListMemo {
		if len(entry.commits) == len(commits) && unsafe.SliceData(entry.commits) == unsafe.SliceData(commits) {
			return entry.key
		}
	}

	digest := sha256.New()
	for _, commit := range commits {
		digest.Write([]byte(commit.Hash()))
		digest.Write([]byte{0})
	}
	key := hex.EncodeToString(digest.Sum(nil))

	if len(commitListMemo) == commitListMemoSize {
		copy(commitListMemo, commitListMemo[1:])
		commitListMemo = commitListMemo[:commitListMemoSize-1]
	}
	commitListMemo = append(commitListMemo, commitListMemoEntry{commits: commits, key: key})
	return key
}

// DropPipeSetCache forgets every cached pipe set, so that the next render computes
// the graph again. F5 calls it (fork_full_refresh.go), as a way out of a graph that
// looks wrong for any reason. It also forgets the memoized list digests.
func DropPipeSetCache() {
	mutex.Lock()
	defer mutex.Unlock()
	pipeSetCache = make(map[pipeSetCacheKey][][]graph.Pipe)
	commitListMemo = nil
}
