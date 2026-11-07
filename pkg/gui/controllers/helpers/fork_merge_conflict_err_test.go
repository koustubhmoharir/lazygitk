package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Fork: every kind of conflict that Git reports is recognised, not only content
// conflicts in a file whose name happens to match "Merge conflict in file".
func TestIsMergeConflictErrRecognisesEveryConflictKind(t *testing.T) {
	for _, output := range []string{
		"Auto-merging a.txt\nCONFLICT (add/add): Merge conflict in a.txt\nerror: could not apply 1234567... add a.txt",
		"CONFLICT (modify/delete): b.txt deleted in HEAD and modified in 1234567 (change b).  Version 1234567 (change b) of b.txt left in tree.\nerror: could not apply 1234567... change b",
		"CONFLICT (rename/delete): c.txt renamed to d.txt in 1234567, but deleted in HEAD.",
		"Auto-merging e.txt\nCONFLICT (content): Merge conflict in e.txt",
	} {
		assert.True(t, isMergeConflictErr(output), output)
	}
	assert.False(t, isMergeConflictErr("fatal: bad revision 'nope'"))
}
