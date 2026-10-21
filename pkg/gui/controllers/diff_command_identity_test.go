package controllers

import "testing"

func TestDiffSelectionIdentity(t *testing.T) {
	tests := []struct {
		name      string
		source    diffCommandSource
		secondary bool
		discard   bool
		id        string
		verb      string
	}{
		{name: "working tree primary stage", source: diffCommandSourceWorkingTree, id: "diff.stage-selection", verb: "stage"},
		{name: "working tree primary discard", source: diffCommandSourceWorkingTree, discard: true, id: "diff.discard-worktree-selection", verb: "discard"},
		{name: "working tree secondary unstage", source: diffCommandSourceWorkingTree, secondary: true, id: "diff.unstage-selection", verb: "unstage"},
		{name: "working tree secondary discard alias", source: diffCommandSourceWorkingTree, secondary: true, discard: true, id: "diff.unstage-selection", verb: "unstage"},
		{name: "commit primary toggle patch", source: diffCommandSourceCommit, id: "patch.toggle-selection", verb: "toggle-patch-selection"},
		{name: "commit primary discard", source: diffCommandSourceCommit, discard: true, id: "commit.discard-selection", verb: "discard-from-commit"},
		{name: "commit secondary remove from patch", source: diffCommandSourceCommit, secondary: true, id: "patch.remove-selection", verb: "remove-from-patch"},
		{name: "commit secondary discard", source: diffCommandSourceCommit, secondary: true, discard: true, id: "commit.discard-selection", verb: "discard-from-commit"},
		{name: "none primary fallback", source: diffCommandSourceNone, id: "diff.primary-selection", verb: "primary-action"},
		{name: "none primary discard fallback", source: diffCommandSourceNone, discard: true, id: "diff.discard-selection", verb: "discard"},
		{name: "none secondary fallback", source: diffCommandSourceNone, secondary: true, id: "diff.primary-selection", verb: "primary-action"},
		{name: "none secondary discard fallback", source: diffCommandSourceNone, secondary: true, discard: true, id: "diff.discard-selection", verb: "discard"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id, verb := diffSelectionIdentity(test.source, test.secondary, test.discard)
			if id != test.id || verb != test.verb {
				t.Fatalf("diffSelectionIdentity() = (%q, %q), want (%q, %q)", id, verb, test.id, test.verb)
			}
		})
	}
}
