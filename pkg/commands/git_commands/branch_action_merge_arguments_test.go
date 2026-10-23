package git_commands

import (
	"reflect"
	"strings"
	"testing"
)

func TestBranchActionMergeArgumentsQuotedAndLiteralValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"quotes and empty", `-m "hello world" --file ''`, []string{"-m", "hello world", "--file", ""}},
		{"single quote grouping", `-m 'its literal'`, []string{"-m", "its literal"}},
		{"escaped whitespace", `--message hello\ world`, []string{"--message", "hello world"}},
		{"shell syntax stays literal", `-m '$(touch /tmp/x) ` + "`" + `whoami` + "`" + `'`, []string{"-m", "$(touch /tmp/x) `whoami`"}},
		{"escaped quote", `--message "say \"hi\""`, []string{"--message", `say "hi"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBranchActionMergeArguments(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBranchActionMergeArgumentsMalformedInput(t *testing.T) {
	for _, input := range []string{"-m 'unfinished", `-m "unfinished`, `-m value\`, "-m \x00"} {
		if got, err := parseBranchActionMergeArguments(input); err == nil {
			t.Errorf("parseBranchActionMergeArguments(%q) = %#v, nil; want error", input, got)
		}
	}
}

func TestBranchActionMergeArgumentsValueOptions(t *testing.T) {
	for _, input := range []string{
		`--strategy=ours --strategy-option theirs`,
		`-sours -X -leading --message=-dash -m -leading -Fpath --file -path --cleanup=strip --into-name branch`,
		`-m '' --file ""`,
	} {
		if _, err := parseBranchActionMergeArguments(input); err != nil {
			t.Errorf("parseBranchActionMergeArguments(%q): %v", input, err)
		}
	}
	for _, input := range []string{`--message`, `-m`, `--strategy-option`} {
		if _, err := parseBranchActionMergeArguments(input); err == nil {
			t.Errorf("parseBranchActionMergeArguments(%q) succeeded without a value", input)
		}
	}
}

func TestBranchActionMergeArgumentsOptionalValuesDoNotConsumeSources(t *testing.T) {
	for _, input := range []string{`--log source`, `-S source`, `--gpg-sign source`} {
		if _, err := parseBranchActionMergeArguments(input); err == nil {
			t.Errorf("parseBranchActionMergeArguments(%q) accepted extra source", input)
		}
	}
	for _, input := range []string{`--log=4`, `--gpg-sign=key`, `-Skey`, `-S`} {
		if _, err := parseBranchActionMergeArguments(input); err != nil {
			t.Errorf("parseBranchActionMergeArguments(%q): %v", input, err)
		}
	}
}

func TestBranchActionMergeArgumentsShortBundlesAndWhitelist(t *testing.T) {
	for _, input := range []string{`-qvn`, `-e -mmessage`, `--no-message --no-strategy --no-strategy-option --no-cleanup --no-into-name`, `--ff-only --no-ff --squash --no-commit --allow-unrelated-histories --no-autostash --verify`} {
		if _, err := parseBranchActionMergeArguments(input); err != nil {
			t.Errorf("parseBranchActionMergeArguments(%q): %v", input, err)
		}
	}
	for _, input := range []string{`source`, `-- source`, `--abort`, `--no-abort`, `--quit`, `--continue`, `--help`, `--strat`, `--unknown`, `-z`} {
		if _, err := parseBranchActionMergeArguments(input); err == nil {
			t.Errorf("parseBranchActionMergeArguments(%q) unexpectedly succeeded", input)
		}
	}
}

func TestBranchActionMergeArgumentsErrorsDoNotEchoValues(t *testing.T) {
	for _, tt := range []struct {
		input      string
		wantError  string
		mustRedact string
	}{
		{input: `https://user:secret@example.test/repo`, wantError: "positional sources", mustRedact: "secret"},
		{input: `--unknown=private-value`, wantError: "--unknown", mustRedact: "private-value"},
	} {
		_, err := parseBranchActionMergeArguments(tt.input)
		if err == nil {
			t.Fatalf("parseBranchActionMergeArguments(%q) succeeded", tt.input)
		}
		if !strings.Contains(err.Error(), tt.wantError) {
			t.Errorf("error %q does not identify %q", err, tt.wantError)
		}
		if strings.Contains(err.Error(), tt.mustRedact) {
			t.Errorf("error %q leaks rejected input value", err)
		}
	}
}

func TestBranchActionMergeOptionRecords(t *testing.T) {
	input := `-m --squash -qvXours -Ssigning-key --no-message --no-ff`
	argv, records, err := parseBranchActionMergeOptionRecords(input)
	if err != nil {
		t.Fatal(err)
	}
	wantArgv := []string{"-m", "--squash", "-qvXours", "-Ssigning-key", "--no-message", "--no-ff"}
	if !reflect.DeepEqual(argv, wantArgv) {
		t.Fatalf("argv = %#v, want original tokens %#v", argv, wantArgv)
	}
	wantRecords := []branchActionMergeArgument{
		{Name: "--message", Value: "--squash", HasValue: true},
		{Name: "--quiet"},
		{Name: "--verbose"},
		{Name: "--strategy-option", Value: "ours", HasValue: true},
		{Name: "--gpg-sign", Value: "signing-key", HasValue: true},
		{Name: "--no-message"},
		{Name: "--no-ff"},
	}
	if !reflect.DeepEqual(records, wantRecords) {
		t.Fatalf("records = %#v, want %#v", records, wantRecords)
	}
}

func TestBranchActionMergeOptionRecordsOptionalAndBundledOptions(t *testing.T) {
	argv, records, err := parseBranchActionMergeOptionRecords(`-qvn -S --log=2 --no-strategy --no-cleanup`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(argv, []string{"-qvn", "-S", "--log=2", "--no-strategy", "--no-cleanup"}) {
		t.Fatalf("argv changed: %#v", argv)
	}
	want := []branchActionMergeArgument{
		{Name: "--quiet"},
		{Name: "--verbose"},
		{Name: "--no-stat"},
		{Name: "--gpg-sign"},
		{Name: "--log", Value: "2", HasValue: true},
		{Name: "--no-strategy"},
		{Name: "--no-cleanup"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("records = %#v, want %#v", records, want)
	}
}
