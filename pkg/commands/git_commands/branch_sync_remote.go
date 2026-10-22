package git_commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

// BranchSyncRemoteInfo is fresh inspection data, not permission to push or
// fetch. URLs are configured values, not a claim that URL rewriting is absent.
// A command plan must revalidate Signature and the chosen branch/refspec.
type BranchSyncRemoteInfo struct {
	Name          string
	URLs          []string
	PushURLs      []string
	FetchRefspecs []string
	PushRefspecs  []string
	Mirror        bool
	Signature     string
}

type branchSyncConfigEntry struct {
	key, value string
	hasValue   bool
}

// InspectBranchSyncRemote retains every configured mapping, including multiple
// destinations for one source. It never resolves ambiguity by taking the first.
// Remote "." is a local-upstream case, handled separately from named remotes.
func (self *SyncCommands) InspectBranchSyncRemote(name string) (BranchSyncRemoteInfo, error) {
	if name == "" || name == "." || strings.ContainsAny(name, "\x00\r\n") {
		return BranchSyncRemoteInfo{}, fmt.Errorf("choose a configured named remote")
	}
	stdout, _, err := self.cmd.New(NewGitCmd("config").Arg("--null", "--list", "--includes").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return BranchSyncRemoteInfo{}, fmt.Errorf("inspect remote configuration: %w", err)
	}
	entries, err := parseBranchSyncConfig(stdout)
	if err != nil {
		return BranchSyncRemoteInfo{}, err
	}
	info := BranchSyncRemoteInfo{Name: name}
	digest := sha256.New()
	prefix := "remote." + name + "."
	for _, entry := range entries {
		key, remoteEntry := strings.CutPrefix(entry.key, prefix)
		// Subsection names can contain dots, whereas the final variable name
		// cannot. Do not include remote "a.b" when inspecting remote "a".
		remoteEntry = remoteEntry && key != "" && !strings.ContainsRune(key, '.')
		if remoteEntry || branchSyncURLRewriteEntry(entry.key) {
			writeBranchSyncConfigEntry(digest, entry)
		}
		if !remoteEntry {
			continue
		}
		switch key {
		case "url":
			info.URLs = append(info.URLs, entry.value)
		case "pushurl":
			info.PushURLs = append(info.PushURLs, entry.value)
		case "fetch":
			info.FetchRefspecs = append(info.FetchRefspecs, entry.value)
		case "push":
			info.PushRefspecs = append(info.PushRefspecs, entry.value)
		}
	}
	if len(info.URLs) == 0 && len(info.PushURLs) == 0 {
		return BranchSyncRemoteInfo{}, fmt.Errorf("remote %q has no configured fetch or push URL", name)
	}
	for _, url := range append(append([]string{}, info.URLs...), info.PushURLs...) {
		if url == "" {
			return BranchSyncRemoteInfo{}, fmt.Errorf("remote %q has an empty configured URL", name)
		}
	}
	// Ask Git to normalize boolean spellings and valueless keys. Its runner
	// executes a clone, so missing-key detection must use the native exit error.
	value, _, err := self.cmd.New(NewGitCmd("config").Arg("--type=bool", "--get", prefix+"mirror").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil && !commandExitedWithOne(err) {
		return BranchSyncRemoteInfo{}, fmt.Errorf("inspect remote mirroring: %w", err)
	}
	if err == nil {
		switch trimTerminalLineEnding(value) {
		case "true":
			info.Mirror = true
		case "false":
		default:
			return BranchSyncRemoteInfo{}, fmt.Errorf("invalid normalized remote mirror value")
		}
	}
	// Include the separately queried normalization too. A concurrent config
	// edit between reads must not make an inconsistent snapshot compare equal
	// to a later consistent one when a command plan revalidates it.
	writeFingerprintString(digest, fmt.Sprintf("mirror=%t", info.Mirror))
	info.Signature = hex.EncodeToString(digest.Sum(nil))
	return info, nil
}

func parseBranchSyncConfig(stdout string) ([]branchSyncConfigEntry, error) {
	if stdout == "" {
		return nil, nil
	}
	if !strings.HasSuffix(stdout, "\x00") {
		return nil, fmt.Errorf("truncated sync configuration query")
	}
	entries := []branchSyncConfigEntry{}
	for _, record := range strings.Split(strings.TrimSuffix(stdout, "\x00"), "\x00") {
		key, value, hasValue := strings.Cut(record, "\n")
		if key == "" {
			return nil, fmt.Errorf("malformed sync configuration query")
		}
		entries = append(entries, branchSyncConfigEntry{key: key, value: value, hasValue: hasValue})
	}
	return entries, nil
}

func branchSyncURLRewriteEntry(key string) bool {
	return strings.HasPrefix(key, "url.") &&
		(strings.HasSuffix(key, ".insteadof") || strings.HasSuffix(key, ".pushinsteadof"))
}

func writeBranchSyncConfigEntry(h hash.Hash, entry branchSyncConfigEntry) {
	writeFingerprintString(h, entry.key)
	writeFingerprintString(h, entry.value)
	if entry.hasValue {
		_, _ = h.Write([]byte{1})
	} else {
		_, _ = h.Write([]byte{0})
	}
}
