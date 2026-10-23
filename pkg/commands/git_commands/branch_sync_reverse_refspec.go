package git_commands

import (
	"fmt"
	"strings"
)

// branchSyncRemoteRefsForTracking returns configured positive fetch sources
// whose destination maps to trackingRef. These are configured mappings, not a
// claim about the server state observed by the last fetch: a cached tracking
// ref may predate a negative fetch exclusion. Callers must present multiple
// candidates for an explicit choice rather than assuming one was fetched.
func (self *SyncCommands) branchSyncRemoteRefsForTracking(specs []string, trackingRef string) ([]string, error) {
	if !strings.HasPrefix(trackingRef, "refs/") {
		return nil, fmt.Errorf("tracking ref %q must be fully qualified", trackingRef)
	}
	if err := self.checkBranchSyncRef(trackingRef, false); err != nil {
		return nil, fmt.Errorf("invalid tracking ref %q: %w", trackingRef, err)
	}

	parsed := make([]branchSyncFetchRefspec, 0, len(specs))
	for _, spec := range specs {
		refspec, err := self.parseBranchSyncFetchRefspec(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid configured fetch refspec: %w", err)
		}
		parsed = append(parsed, refspec)
	}

	result := make([]string, 0, len(parsed))
	seen := make(map[string]struct{}, len(parsed))
	for _, refspec := range parsed {
		if refspec.negative || !refspec.hasColon || refspec.destination == "" {
			continue
		}
		capture, matches := matchBranchSyncRefPattern(refspec.destination, trackingRef)
		if !matches {
			continue
		}

		source := refspec.source
		if refspec.wildcard {
			source = strings.Replace(source, "*", capture, 1)
		}
		if !strings.HasPrefix(source, "refs/") {
			return nil, fmt.Errorf("matching fetch source is abbreviated; a full refs/ source is required")
		}
		if err := self.checkBranchSyncRef(source, false); err != nil {
			return nil, fmt.Errorf("invalid expanded fetch source %q: %w", source, err)
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		result = append(result, source)
	}
	return result, nil
}
