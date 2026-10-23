package git_commands

import (
	"reflect"
	"testing"
)

func TestBranchActionMergePolicy(t *testing.T) {
	tests := []struct {
		name      string
		defaultFF string
		defaults  []string
		records   []branchActionMergeArgument
		want      branchActionMergePolicy
	}{
		{
			name:      "defaults are cloned",
			defaultFF: "true", defaults: []string{"ort"},
			want: branchActionMergePolicy{ff: "true", commit: true, strategies: []string{"ort"}},
		},
		{
			name:      "last fast forward option wins",
			defaultFF: "true",
			records:   []branchActionMergeArgument{{Name: "--no-ff"}, {Name: "--ff-only"}, {Name: "--ff"}},
			want:      branchActionMergePolicy{ff: "true", commit: true},
		},
		{
			name:    "squash defaults commit off but no squash restores default",
			records: []branchActionMergeArgument{{Name: "--squash"}, {Name: "--no-squash"}},
			want:    branchActionMergePolicy{commit: true},
		},
		{
			name:    "commit after squash is invalid in native git",
			records: []branchActionMergeArgument{{Name: "--squash"}, {Name: "--commit"}},
			want:    branchActionMergePolicy{squash: true, commit: false, invalidCombination: true},
		},
		{
			name:    "commit before squash is also invalid in native git",
			records: []branchActionMergeArgument{{Name: "--commit"}, {Name: "--squash"}},
			want:    branchActionMergePolicy{squash: true, commit: false, invalidCombination: true},
		},
		{
			name:    "explicit no commit survives squash reset",
			records: []branchActionMergeArgument{{Name: "--squash"}, {Name: "--no-commit"}, {Name: "--no-squash"}},
			want:    branchActionMergePolicy{commit: false},
		},
		{
			name:    "no ff conflicts with squash",
			records: []branchActionMergeArgument{{Name: "--squash"}, {Name: "--no-ff"}},
			want:    branchActionMergePolicy{ff: "false", squash: true, commit: false, invalidCombination: true},
		},
		{
			name:    "boolean resets",
			records: []branchActionMergeArgument{{Name: "--allow-unrelated-histories"}, {Name: "--no-allow-unrelated-histories"}, {Name: "--no-commit"}, {Name: "--commit"}},
			want:    branchActionMergePolicy{commit: true},
		},
		{
			name:     "first strategy replaces defaults and later strategies append",
			defaults: []string{"ort"},
			records:  []branchActionMergeArgument{{Name: "--strategy", Value: "ours"}, {Name: "--strategy", Value: "recursive"}},
			want:     branchActionMergePolicy{commit: true, strategies: []string{"ours", "recursive"}},
		},
		{
			name:     "no strategy is a native no-op",
			defaults: []string{"ort"},
			records:  []branchActionMergeArgument{{Name: "--strategy", Value: "ours"}, {Name: "--no-strategy"}},
			want:     branchActionMergePolicy{commit: true, strategies: []string{"ours"}},
		},
		{
			name:     "no strategy leaves default fallback available",
			defaults: []string{"ort"},
			records:  []branchActionMergeArgument{{Name: "--no-strategy"}},
			want:     branchActionMergePolicy{commit: true, strategies: []string{"ort"}},
		},
		{
			name:    "strategy option reset",
			records: []branchActionMergeArgument{{Name: "--strategy-option", Value: "ours"}, {Name: "--strategy-option", Value: "theirs"}, {Name: "--no-strategy-option"}, {Name: "--strategy-option", Value: "renormalize"}},
			want:    branchActionMergePolicy{commit: true, strategyOptions: []string{"renormalize"}},
		},
		{
			name:    "quoted option value is not interpreted",
			records: []branchActionMergeArgument{{Name: "--message", Value: "--squash", HasValue: true}},
			want:    branchActionMergePolicy{commit: true, customMessage: true},
		},
		{
			name:    "no message clears text but have message remains sticky",
			records: []branchActionMergeArgument{{Name: "--message", Value: "message"}, {Name: "--no-message"}, {Name: "--file", Value: "path"}},
			want:    branchActionMergePolicy{commit: true, customMessage: true},
		},
		{
			name:    "file remains a custom message after no message",
			records: []branchActionMergeArgument{{Name: "--file", Value: "path"}, {Name: "--no-message"}},
			want:    branchActionMergePolicy{commit: true, customMessage: true},
		},
		{
			name:    "no message alone does not set have message",
			records: []branchActionMergeArgument{{Name: "--no-message"}},
			want:    branchActionMergePolicy{commit: true},
		},
		{
			name:    "irrelevant records ignored",
			records: []branchActionMergeArgument{{Name: "--quiet"}},
			want:    branchActionMergePolicy{commit: true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := resolveBranchActionMergePolicy(test.defaultFF, test.defaults, test.records)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("resolveBranchActionMergePolicy() = %#v, want %#v", got, test.want)
			}
		})
	}

	defaults := []string{"ort", "resolve"}
	got := resolveBranchActionMergePolicy("false", defaults, nil)
	defaults[0] = "changed"
	if got.strategies[0] != "ort" {
		t.Fatalf("default strategies were not cloned: %#v", got.strategies)
	}
	got.strategies[0] = "changed"
	if defaults[1] != "resolve" {
		t.Fatalf("policy strategies alias input: %#v", defaults)
	}
}
