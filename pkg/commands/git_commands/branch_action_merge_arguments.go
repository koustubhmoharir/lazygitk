package git_commands

import (
	"fmt"
	"strings"
)

// parseBranchActionMergeArguments parses the configured merge argument string
// without invoking a shell, then checks it against Git's documented merge
// options. Positional arguments are deliberately rejected: the caller owns the
// merge source and destination.
func parseBranchActionMergeArguments(value string) ([]string, error) {
	args, _, err := parseBranchActionMergeOptionRecords(value)
	return args, err
}

type branchActionMergeArgument struct {
	Name     string
	Value    string
	HasValue bool
}

func parseBranchActionMergeOptionRecords(value string) (argv []string, records []branchActionMergeArgument, err error) {
	argv, err = tokenizeBranchActionMergeArguments(value)
	if err != nil {
		return nil, nil, err
	}
	records, err = validateBranchActionMergeArguments(argv)
	if err != nil {
		return nil, nil, err
	}
	return argv, records, nil
}

func tokenizeBranchActionMergeArguments(value string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote byte
	started := false

	flush := func() {
		if started {
			args = append(args, current.String())
			current.Reset()
			started = false
		}
	}

	for i := 0; i < len(value); i++ {
		b := value[i]
		if b == 0 {
			return nil, fmt.Errorf("merge arguments contain a NUL byte")
		}
		if quote == '\'' {
			if b == '\'' {
				quote = 0
			} else {
				current.WriteByte(b)
			}
			continue
		}
		if quote == '"' {
			switch b {
			case '"':
				quote = 0
			case '\\':
				i++
				if i == len(value) {
					return nil, fmt.Errorf("merge arguments end with an escape")
				}
				if value[i] == 0 {
					return nil, fmt.Errorf("merge arguments contain a NUL byte")
				}
				current.WriteByte(value[i])
			default:
				current.WriteByte(b)
			}
			continue
		}
		switch {
		case b == '\\':
			i++
			if i == len(value) {
				return nil, fmt.Errorf("merge arguments end with an escape")
			}
			if value[i] == 0 {
				return nil, fmt.Errorf("merge arguments contain a NUL byte")
			}
			current.WriteByte(value[i])
			started = true
		case b == '\'' || b == '"':
			quote = b
			started = true
		case isMergeArgumentWhitespace(b):
			flush()
		default:
			current.WriteByte(b)
			started = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("merge arguments contain an unclosed quote")
	}
	flush()
	return args, nil
}

func isMergeArgumentWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f'
}

func validateBranchActionMergeArguments(args []string) ([]branchActionMergeArgument, error) {
	var records []branchActionMergeArgument
	// Boolean long options, including only the negated spellings documented by
	// Git's merge help. Aliases are listed explicitly to reject abbreviations.
	booleanLong := map[string]bool{}
	for _, name := range []string{
		"stat", "summary", "compact-summary", "log", "squash", "commit",
		"edit", "ff", "ff-only", "rerere-autoupdate", "verify-signatures", "verbose",
		"quiet", "allow-unrelated-histories", "progress", "gpg-sign", "autostash",
		"overwrite-ignore", "signoff", "verify",
	} {
		booleanLong["--"+name] = true
	}
	for _, name := range []string{
		"stat", "summary", "compact-summary", "log", "squash", "commit",
		"edit", "ff", "rerere-autoupdate", "verify-signatures", "verbose",
		"quiet", "allow-unrelated-histories", "progress", "gpg-sign", "autostash",
		"overwrite-ignore", "signoff", "verify",
	} {
		booleanLong["--no-"+name] = true
	}

	valueLong := map[string]bool{
		"--strategy": true, "--strategy-option": true, "--message": true,
		"--file": true, "--cleanup": true, "--into-name": true,
	}
	optionalLong := map[string]bool{"--log": true, "--gpg-sign": true}
	shortValue := map[byte]bool{'s': true, 'X': true, 'm': true, 'F': true}
	shortOptional := byte('S')
	shortBool := map[byte]bool{'n': true, 'e': true, 'v': true, 'q': true}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return nil, fmt.Errorf("merge arguments may not contain positional sources or --")
		}
		if strings.HasPrefix(arg, "--") {
			name, value, hasValue := strings.Cut(arg, "=")
			if valueLong[name] {
				if !hasValue {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("merge option %q requires a value", name)
					}
					i++ // Values are consumed literally, even when they begin with '-'.
					value = args[i]
				}
				records = append(records, branchActionMergeArgument{Name: name, Value: value, HasValue: true})
				continue
			}
			if optionalLong[name] {
				record := branchActionMergeArgument{Name: name, Value: value, HasValue: hasValue}
				records = append(records, record)
				continue // Optional values are accepted only with '='.
			}
			if booleanLong[arg] {
				records = append(records, branchActionMergeArgument{Name: arg})
				continue
			}
			// Boolean reset forms that Git documents for value-taking options.
			if arg == "--no-message" || arg == "--no-strategy" || arg == "--no-strategy-option" || arg == "--no-cleanup" || arg == "--no-into-name" {
				records = append(records, branchActionMergeArgument{Name: arg})
				continue
			}
			return nil, fmt.Errorf("unsupported or invalid merge option %q", name)
		}
		if len(arg) < 2 || arg[0] != '-' {
			return nil, fmt.Errorf("merge arguments may not contain positional sources")
		}
		// Short options may be bundled when every member is a known boolean.
		// The first value-taking member consumes the token suffix or one next
		// token, and emits a single normalized record.
		for j := 1; j < len(arg); j++ {
			option := arg[j]
			if option == shortOptional {
				record := branchActionMergeArgument{Name: "--gpg-sign"}
				if j+1 < len(arg) {
					record.Value, record.HasValue = arg[j+1:], true
				}
				records = append(records, record)
				break // -S accepts an attached key, never a following token.
			}
			if shortBool[option] {
				name := map[byte]string{'n': "--no-stat", 'e': "--edit", 'v': "--verbose", 'q': "--quiet"}[option]
				records = append(records, branchActionMergeArgument{Name: name})
				continue
			}
			if shortValue[option] {
				name := map[byte]string{'s': "--strategy", 'X': "--strategy-option", 'm': "--message", 'F': "--file"}[option]
				value := arg[j+1:]
				hasValue := j+1 < len(arg)
				if !hasValue {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("merge option -%c requires a value", option)
					}
					i++
					value, hasValue = args[i], true
				}
				records = append(records, branchActionMergeArgument{Name: name, Value: value, HasValue: hasValue})
				break
			}
			return nil, fmt.Errorf("unsupported or invalid merge option %q", "-"+string(option))
		}
	}
	return records, nil
}
