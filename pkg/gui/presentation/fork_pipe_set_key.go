package presentation

import (
	"crypto/sha256"
	"encoding/hex"

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

// commitListKey returns a digest of the hashes of commits, in order.
func commitListKey(commits []*models.Commit) string {
	digest := sha256.New()
	for _, commit := range commits {
		digest.Write([]byte(commit.Hash()))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// DropPipeSetCache forgets every cached pipe set, so that the next render computes
// the graph again. F5 calls it (fork_full_refresh.go), as a way out of a graph that
// looks wrong for any reason.
func DropPipeSetCache() {
	mutex.Lock()
	defer mutex.Unlock()
	pipeSetCache = make(map[pipeSetCacheKey][][]graph.Pipe)
}
