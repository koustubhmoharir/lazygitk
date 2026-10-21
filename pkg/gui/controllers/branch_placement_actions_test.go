package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestBranchPlacementSourceGuard(t *testing.T) {
	valid := historicalTestCommit("0123456789abcdef0123456789abcdef01234567")
	for _, tc := range []struct {
		name    string
		items   []*models.Commit
		start   int
		end     int
		allowed bool
	}{
		{name: "none"},
		{name: "range", items: []*models.Commit{valid, valid}, start: 0, end: 1},
		{name: "nil commit", items: []*models.Commit{nil}},
		{name: "synthetic", items: []*models.Commit{historicalTestCommit("HEAD")}},
		{name: "full object id", items: []*models.Commit{valid}, allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			controller := historicalTestBasicController(tc.items, tc.start, tc.end)
			oid, reason := controller.branchPlacementSource()
			if (reason == nil) != tc.allowed {
				t.Fatalf("allowed = %v, want %v (reason: %v)", reason == nil, tc.allowed, reason)
			}
			if tc.allowed && oid != valid.Hash() {
				t.Errorf("source OID = %q, want %q", oid, valid.Hash())
			}
		})
	}
}

func TestBranchPlacementHandlerRejectsInvalidSelectionWithoutGit(t *testing.T) {
	controller := historicalTestBasicController(nil, 0, 0)
	if err := controller.beginBranchPlacement(); err == nil {
		t.Fatal("expected invalid selection error")
	}
}

func TestBranchPlacementBindingMetadata(t *testing.T) {
	controller := historicalTestBasicController(nil, 0, 0)
	binding := controller.branchPlacementBinding(types.KeybindingsOpts{
		Config:  config.GetDefaultConfig().Keybinding,
		GetKeys: config.GetValidatedKeyBindingKeys,
	})
	if binding.CommandID != "branch.place-at-selected-commit" || binding.Verb != "place-branch" {
		t.Errorf("binding identity = %q/%q, want branch.place-at-selected-commit/place-branch", binding.CommandID, binding.Verb)
	}
	if len(binding.Keys) != 0 {
		t.Errorf("default keys = %v, want unbound", binding.Keys)
	}
	if !binding.OpensMenu {
		t.Error("binding should open the name prompt")
	}
	if binding.Description == "" {
		t.Error("binding description should be populated")
	}
}

func TestPlacementTipLabels(t *testing.T) {
	for _, tc := range []struct {
		name, oldOID, newOID, wantOld, wantNew string
	}{
		{
			name:    "overlapping prefix extends until tips differ",
			oldOID:  "abcdef0123456789abcdef0123456789abcdef01",
			newOID:  "abcdef0999999999abcdef0123456789abcdef01",
			wantOld: "abcdef01", wantNew: "abcdef09",
		},
		{
			name:    "ordinary labels use seven characters",
			oldOID:  "0123456789abcdef0123456789abcdef01234567",
			newOID:  "fedcba9876543210fedcba9876543210fedcba98",
			wantOld: "0123456", wantNew: "fedcba9",
		},
		{name: "empty inputs are safe", oldOID: "", newOID: "", wantOld: "", wantNew: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldLabel, newLabel := placementTipLabels(tc.oldOID, tc.newOID)
			if oldLabel != tc.wantOld || newLabel != tc.wantNew {
				t.Errorf("labels = %q/%q, want %q/%q", oldLabel, newLabel, tc.wantOld, tc.wantNew)
			}
		})
	}
}
