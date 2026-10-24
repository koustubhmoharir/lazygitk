package cherrypicking

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func markTestCommit(hash, name string) *models.Commit {
	return models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{
		Hash: hash, Name: name, Parents: []string{"parent-" + hash},
		Tags: []string{"tag-" + hash}, Status: models.StatusUnpushed,
		AuthorName: "author", AuthorEmail: "author@example.com", UnixTimestamp: 42,
	})
}

func TestMarkCommitsIsIdempotentAndUnionsInSelectionOrder(t *testing.T) {
	mode := New()
	first, second, third := markTestCommit("one", "first"), markTestCommit("two", "second"), markTestCommit("three", "third")
	mode.MarkCommits([]*models.Commit{first, second})
	mode.MarkCommits([]*models.Commit{second, third, nil})

	if got, want := commitHashes(mode.CherryPickedCommits), []string{"one", "two", "three"}; !equalStrings(got, want) {
		t.Fatalf("marked hashes = %v, want %v", got, want)
	}
	if mode.ContextKey != "" {
		t.Fatalf("ContextKey = %q, want empty", mode.ContextKey)
	}
}

func TestUnmarkCommitsRemovesOnlySelected(t *testing.T) {
	mode := New()
	mode.MarkCommits([]*models.Commit{markTestCommit("one", "first"), markTestCommit("two", "second"), markTestCommit("three", "third")})
	mode.UnmarkCommits([]*models.Commit{markTestCommit("two", "other instance"), nil})
	if got, want := commitHashes(mode.CherryPickedCommits), []string{"one", "three"}; !equalStrings(got, want) {
		t.Fatalf("marked hashes = %v, want %v", got, want)
	}
}

func TestNilSelectionsAreIgnored(t *testing.T) {
	mode := New()
	mode.MarkCommits([]*models.Commit{nil})
	mode.UnmarkCommits([]*models.Commit{nil})
	if len(mode.CherryPickedCommits) != 0 {
		t.Fatalf("unexpected marks: %v", mode.CherryPickedCommits)
	}
}

func TestPasteHidesMarksUntilFreshMarkAndUnmarkDoesNotResurface(t *testing.T) {
	mode := New()
	old := markTestCommit("old", "old")
	mode.MarkCommits([]*models.Commit{old})
	mode.DidPaste = true
	if mode.Active() {
		t.Fatal("pasted marks should be hidden")
	}
	mode.UnmarkCommits([]*models.Commit{markTestCommit("old", "old")})
	if !mode.DidPaste || len(mode.CherryPickedCommits) != 0 {
		t.Fatal("unmark should retain hidden state and remove the selected hidden mark")
	}

	mode.CherryPickedCommits = []*models.Commit{old}
	mode.DidPaste = true
	mode.MarkCommits([]*models.Commit{markTestCommit("fresh", "fresh")})
	if mode.DidPaste || len(mode.CherryPickedCommits) != 1 || mode.CherryPickedCommits[0].Hash() != "fresh" {
		t.Fatal("first mark after paste should start a fresh visible buffer")
	}
}

func TestMarkCommitsSnapshotsCommitOwnershipAndMetadata(t *testing.T) {
	mode := New()
	original := markTestCommit("commit", "subject")
	mode.MarkCommits([]*models.Commit{original})
	snapshot := mode.CherryPickedCommits[0]

	original.Name = "changed"
	*original.HashPtr() = "changed-hash"
	*original.ParentPtrs()[0] = "changed-parent"
	original.Tags[0] = "changed-tag"

	if snapshot.Hash() != "commit" || snapshot.Name != "subject" || snapshot.Parents()[0] != "parent-commit" || snapshot.Tags[0] != "tag-commit" {
		t.Fatal("snapshot changed after mutating the source commit")
	}
	if snapshot.HashPtr() == original.HashPtr() || snapshot.ParentPtrs()[0] == original.ParentPtrs()[0] {
		t.Fatal("snapshot retained mutable source pointers")
	}
	if snapshot.AuthorName != "author" || snapshot.AuthorEmail != "author@example.com" || snapshot.Status != models.StatusUnpushed || snapshot.UnixTimestamp != 42 {
		t.Fatal("snapshot did not retain commit metadata")
	}
}

func commitHashes(commits []*models.Commit) []string {
	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		hashes = append(hashes, commit.Hash())
	}
	return hashes
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
