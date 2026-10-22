package git_commands

import (
	"fmt"
	"strings"
)

type branchSyncTrackingMapping struct {
	refs      []string
	preferred string
	excluded  bool
}

type branchSyncFetchRefspec struct {
	source      string
	destination string
	negative    bool
	wildcard    bool
	hasColon    bool
}

// branchSyncTrackingRefs inspects configured fetch refspecs for native push's
// tracking lookup. Unlike fetch source resolution, non-pattern sources match
// exactly, without DWIM expansion. It performs no fetch and changes no refs.
func (self *SyncCommands) branchSyncTrackingRefs(specs []string, remoteRef string) (branchSyncTrackingMapping, error) {
	if !strings.HasPrefix(remoteRef, "refs/") {
		return branchSyncTrackingMapping{}, fmt.Errorf("remote ref %q must be fully qualified", remoteRef)
	}
	if err := self.checkBranchSyncRef(remoteRef, false); err != nil {
		return branchSyncTrackingMapping{}, fmt.Errorf("invalid remote ref %q: %w", remoteRef, err)
	}

	parsed := make([]branchSyncFetchRefspec, 0, len(specs))
	for _, spec := range specs {
		refspec, err := self.parseBranchSyncFetchRefspec(spec)
		if err != nil {
			return branchSyncTrackingMapping{}, fmt.Errorf("invalid fetch refspec %q: %w", spec, err)
		}
		parsed = append(parsed, refspec)
	}

	result := branchSyncTrackingMapping{}
	seen := make(map[string]struct{})
	for _, refspec := range parsed {
		if !refspec.negative {
			continue
		}
		if _, matches := matchBranchSyncRefPattern(refspec.source, remoteRef); matches {
			result.excluded = true
		}
	}

	preferredChosen := false
	for _, refspec := range parsed {
		if refspec.negative || !refspec.hasColon {
			continue
		}
		capture, matches := matchBranchSyncRefPattern(refspec.source, remoteRef)
		if !matches {
			continue
		}
		firstPreference := !preferredChosen
		preferredChosen = true
		if refspec.destination == "" {
			continue
		}
		destination := refspec.destination
		if refspec.wildcard {
			destination = strings.Replace(destination, "*", capture, 1)
		}
		if !strings.HasPrefix(destination, "refs/") {
			return branchSyncTrackingMapping{}, fmt.Errorf("fetch destination %q is abbreviated; a full refs/ destination is required", destination)
		}
		if err := self.checkBranchSyncRef(destination, false); err != nil {
			return branchSyncTrackingMapping{}, fmt.Errorf("invalid expanded fetch destination %q: %w", destination, err)
		}
		if _, exists := seen[destination]; !exists {
			seen[destination] = struct{}{}
			result.refs = append(result.refs, destination)
		}
		if firstPreference && !result.excluded {
			result.preferred = destination
		}
	}
	return result, nil
}

func (self *SyncCommands) parseBranchSyncFetchRefspec(spec string) (branchSyncFetchRefspec, error) {
	if strings.ContainsRune(spec, '\x00') {
		return branchSyncFetchRefspec{}, fmt.Errorf("contains NUL")
	}
	refspec := branchSyncFetchRefspec{}
	if strings.HasPrefix(spec, "+") {
		spec = spec[1:]
		if strings.HasPrefix(spec, "^") {
			return branchSyncFetchRefspec{}, fmt.Errorf("force prefix is invalid on a negative refspec")
		}
	}
	if strings.HasPrefix(spec, "^") {
		refspec.negative = true
		spec = spec[1:]
	}
	source, destination, hasColon := strings.Cut(spec, ":")
	if hasColon && strings.Contains(destination, ":") {
		return branchSyncFetchRefspec{}, fmt.Errorf("contains more than one colon")
	}
	if !hasColon {
		destination = ""
	}
	if refspec.negative && hasColon {
		return branchSyncFetchRefspec{}, fmt.Errorf("negative refspec cannot have a destination")
	}
	if source == "@" {
		source = "HEAD"
	}
	wildSource, wildDestination := strings.Count(source, "*"), strings.Count(destination, "*")
	if refspec.negative {
		if wildSource > 1 || wildDestination != 0 {
			return branchSyncFetchRefspec{}, fmt.Errorf("negative refspec may contain at most one source wildcard and no destination")
		}
	} else if wildSource != wildDestination || wildSource > 1 {
		return branchSyncFetchRefspec{}, fmt.Errorf("positive wildcard source and destination must each contain exactly one wildcard")
	}
	if isFullNativeObjectID(source) {
		if refspec.negative {
			return branchSyncFetchRefspec{}, fmt.Errorf("negative refspecs cannot use object IDs")
		}
		return refspec, fmt.Errorf("object ID sources do not map to a named ref")
	}
	if source != "" {
		if err := self.checkBranchSyncRef(source, strings.Contains(source, "*")); err != nil {
			return branchSyncFetchRefspec{}, fmt.Errorf("invalid source: %w", err)
		}
	}
	if destination != "" {
		if err := self.checkBranchSyncRef(destination, strings.Contains(destination, "*")); err != nil {
			return branchSyncFetchRefspec{}, fmt.Errorf("invalid destination: %w", err)
		}
	}
	refspec.source, refspec.destination, refspec.hasColon = source, destination, hasColon
	refspec.wildcard = wildSource == 1
	return refspec, nil
}

func (self *SyncCommands) checkBranchSyncRef(ref string, pattern bool) error {
	args := []string{"check-ref-format", "--allow-onelevel"}
	if pattern {
		args = append(args, "--refspec-pattern")
	}
	args = append(args, ref)
	_, _, err := self.cmd.New(NewGitCmd(args[0]).Arg(args[1:]...).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return fmt.Errorf("git check-ref-format rejected it: %w", err)
	}
	return nil
}

func matchBranchSyncRefPattern(pattern, ref string) (string, bool) {
	if !strings.Contains(pattern, "*") {
		return "", pattern == ref
	}
	prefix, suffix, _ := strings.Cut(pattern, "*")
	if !strings.HasPrefix(ref, prefix) || !strings.HasSuffix(ref, suffix) || len(ref) < len(prefix)+len(suffix) {
		return "", false
	}
	return ref[len(prefix) : len(ref)-len(suffix)], true
}

func isFullNativeObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
