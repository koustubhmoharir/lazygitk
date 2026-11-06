package git_commands

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
)

// Fork: predicts the paths that Git's merge backend creates while it replays a
// range, which the ignored-file review must also cover (07-F1). Directory-rename
// detection moves a replayed add from old/<rest> to new/<rest> when old/ was renamed
// to new/, so that path is in neither the destination tree nor the replayed paths.

// rebaseMergeTreeChainLimit caps the replayed commits that the merge-tree chain
// predicts. A longer replay uses the mirrored-path rule alone.
const rebaseMergeTreeChainLimit = 200

// defaultMergeRenameLimit is Git's default for merge.renameLimit.
const defaultMergeRenameLimit = 7000

// mergeTreeChainRuns counts the merge-tree chains that start. Tests read it to
// prove that the gate skipped the chain; nothing else depends on it.
var mergeTreeChainRuns atomic.Int64

// predictedRebaseCandidates returns the extra candidate files that the replay may
// create beyond the destination tree, the replayed paths and the extras. known holds
// the replayed and extra paths.
//
// The mirrored-path rule runs first. It runs no merge drivers and costs one diff and
// one log. If none of its candidates collides with an ignored file, nothing is added
// and the merge-tree chain is skipped. The chain would run configured merge drivers
// without changing the answer.
//
// Otherwise, for a linear replay onto a destination (Git 2.40 or later, within the
// cap, not reordered), the chain gives the exact paths and replaces the mirrored
// candidates. Anything else uses the mirrored candidates. Merge commits, root
// rebases and reordered replays use the mirrored rule.
//
// Trade-off: the gate assumes the mirrored rule covers every path that Git's
// directory-rename detection can create. The diffs therefore raise the rename limit
// to the one merge uses, so that detection finds the same renames.
func (self *RebaseCommands) predictedRebaseCandidates(replay rebaseReplay, known []string) (map[string]struct{}, error) {
	mirrored, err := self.mirroredRenameCandidates(replay, known)
	if err != nil {
		return nil, err
	}
	risks, err := self.collidingIgnoredPaths(mirrored)
	if err != nil {
		return nil, err
	}
	if len(risks) == 0 {
		return map[string]struct{}{}, nil
	}
	if !replay.reordered && replay.destination != "" {
		if paths, ok := self.mergeTreeChainPaths(replay, rebaseMergeTreeChainLimit); ok {
			chain := make(map[string]struct{}, len(paths))
			for _, path := range paths {
				chain[path] = struct{}{}
			}
			return chain, nil
		}
	}
	return mirrored, nil
}

// renamePair is a file rename: a path that Git reports as moved from oldPath to newPath.
type renamePair struct {
	oldPath, newPath string
}

// dirRename is the directory rename implied by a file rename: paths under oldDir move
// to newDir. An empty newDir is the root.
type dirRename struct {
	oldDir, newDir string
}

// mirroredRenameCandidates maps each known path under an old directory to the new
// directory of every directory rename implied by the replay's file renames. The
// result is a set of file paths; directories are never added.
func (self *RebaseCommands) mirroredRenameCandidates(replay rebaseReplay, known []string) (map[string]struct{}, error) {
	renames, err := self.replayRenames(replay)
	if err != nil {
		return nil, err
	}
	candidates := make(map[string]struct{})
	for _, dir := range directoryRenamesOf(renames) {
		for _, path := range known {
			if mirrored, ok := mirrorPath(path, dir); ok {
				candidates[mirrored] = struct{}{}
			}
		}
	}
	return candidates, nil
}

// directoryRenamesOf derives the directory renames from file renames. A file rename
// a/b/c/f -> x/y/c/f implies a/b/c -> x/y/c. The trailing components match, so it
// also implies a/b -> x/y, and so on while the last components keep matching. An old
// directory that is the root implies nothing, because Git never renames the root.
func directoryRenamesOf(renames []renamePair) []dirRename {
	var dirs []dirRename
	for _, rename := range renames {
		oldDir, newDir := pathDirectory(rename.oldPath), pathDirectory(rename.newPath)
		for oldDir != "" {
			dirs = append(dirs, dirRename{oldDir: oldDir, newDir: newDir})
			if newDir == "" || lastComponent(oldDir) != lastComponent(newDir) {
				break
			}
			oldDir, newDir = pathDirectory(oldDir), pathDirectory(newDir)
		}
	}
	return dirs
}

// mirrorPath returns where Git's directory rename moves path, when path lies under
// the renamed old directory.
func mirrorPath(path string, dir dirRename) (string, bool) {
	if !strings.HasPrefix(path, dir.oldDir+"/") {
		return "", false
	}
	rest := path[len(dir.oldDir)+1:]
	if dir.newDir == "" {
		return rest, true
	}
	return dir.newDir + "/" + rest, true
}

// replayRenames lists the file renames that can drive a directory rename during the
// replay: those between the fork point and the destination, those inside the
// replayed range, and those of the extras. The diffs use Git's merge rename limit,
// so that rename detection finds at least what the merge backend does.
func (self *RebaseCommands) replayRenames(replay rebaseReplay) ([]renamePair, error) {
	renames := slices.Clone(replay.extraRenames)
	limit, err := self.mergeRenameLimit()
	if err != nil {
		return nil, err
	}
	limitArg := fmt.Sprintf("diff.renameLimit=%d", limit)
	if replay.destination != "" && replay.upstream != "" && replay.head != "" {
		fork, err := self.forkPoint(replay.upstream, replay.head)
		if err != nil {
			return nil, err
		}
		if fork != "" {
			output, err := self.selectedRebaseGitOutput(NewGitCmd("diff").Config(limitArg).
				Arg("-M", "--name-status", "-z", fork, replay.destination+"^{commit}", "--").ToArgv())
			if err != nil {
				return nil, fmt.Errorf("list destination renames: %w", err)
			}
			_, pairs := parseNameStatus(output)
			renames = append(renames, pairs...)
		}
	}
	if len(replay.logRevs) > 0 {
		output, err := self.selectedRebaseGitOutput(NewGitCmd("log").Config(limitArg).
			Arg("--format=", "--name-status", "-M", "-m", "--root", "-z").Arg(replay.logRevs...).Arg("--").ToArgv())
		if err != nil {
			return nil, fmt.Errorf("list replayed renames: %w", err)
		}
		_, pairs := parseNameStatus(output)
		renames = append(renames, pairs...)
	}
	return renames, nil
}

// mergeRenameLimit returns merge.renameLimit, or Git's default when it is unset.
func (self *RebaseCommands) mergeRenameLimit() (int, error) {
	output, err := self.selectedRebaseGitOutput(NewGitCmd("config").Arg("--int", "--get", "merge.renameLimit").ToArgv())
	if commandExitedWithOne(err) {
		return defaultMergeRenameLimit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query merge.renameLimit: %w", err)
	}
	limit, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("invalid merge.renameLimit %q", output)
	}
	return limit, nil
}

type rebaseReplayCommit struct {
	oid, parent string
}

// mergeTreeChainPaths returns every path of every intermediate tree of the replay.
// It reports false when the chain does not apply: Git is older than 2.40 (which
// added merge-tree --merge-base), the replay has a merge or root commit, or it has
// more than limit commits. A failure is logged at debug level and also reports
// false, so the caller falls back.
func (self *RebaseCommands) mergeTreeChainPaths(replay rebaseReplay, limit int) ([]string, bool) {
	if self.version == nil || !self.version.IsAtLeast(2, 40, 0) {
		return nil, false
	}
	commits, linear, err := self.linearReplayCommits(replay.logRevs, limit)
	if err != nil {
		self.Log.WithField("error", err).Debug("rebase prediction: listing the replay failed; using the mirrored-path rule")
		return nil, false
	}
	if !linear {
		return nil, false
	}
	mergeTreeChainRuns.Add(1)
	store, err := self.newPrivateObjectStore()
	if err != nil {
		self.Log.WithField("error", err).Debug("rebase prediction: private object storage failed; using the mirrored-path rule")
		return nil, false
	}
	paths, err := self.predictMergeTreeChain(store, replay.destination, commits)
	if removeErr := store.remove(); removeErr != nil {
		self.Log.WithField("error", removeErr).Debug("rebase prediction: removing private object storage failed")
	}
	if err != nil {
		self.Log.WithField("error", err).Debug("rebase prediction: merge-tree chain failed; using the mirrored-path rule")
		return nil, false
	}
	return paths, true
}

// linearReplayCommits lists the replayed commits oldest first, in the order that
// Git's rebase replays them. linear is false when a commit is a merge or a root, or
// when there are more than limit commits.
func (self *RebaseCommands) linearReplayCommits(logRevs []string, limit int) ([]rebaseReplayCommit, bool, error) {
	output, err := self.selectedRebaseGitOutput(NewGitCmd("rev-list").Arg("--reverse", "--topo-order", "--parents").Arg(logRevs...).ToArgv())
	if err != nil {
		return nil, false, fmt.Errorf("list replayed commits: %w", err)
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, true, nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) > limit {
		return nil, false, nil
	}
	commits := make([]rebaseReplayCommit, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			// A merge lists two parents, and a root lists none.
			return nil, false, nil
		}
		commits = append(commits, rebaseReplayCommit{oid: fields[0], parent: fields[1]})
	}
	return commits, true, nil
}

// predictMergeTreeChain merges each replayed commit onto the result of the previous
// one, as the sequencer does. After a conflicted step the conflicted tree continues
// the chain: it still holds every path, although a manual resolution made later
// cannot be predicted exactly.
func (self *RebaseCommands) predictMergeTreeChain(store *privateObjectStore, destination string, commits []rebaseReplayCommit) ([]string, error) {
	cur := destination + "^{commit}"
	seen := make(map[string]struct{})
	var paths []string
	for i, commit := range commits {
		tree, err := self.writeMergeTreeInStore(store, NewGitCmd("merge-tree").
			Arg("--write-tree", "--no-messages", "-z", "--merge-base="+commit.parent, cur, commit.oid))
		if err != nil {
			return nil, err
		}
		stepPaths, err := self.treePaths(tree, store.environment)
		if err != nil {
			return nil, err
		}
		for _, path := range stepPaths {
			if _, exists := seen[path]; !exists {
				seen[path] = struct{}{}
				paths = append(paths, path)
			}
		}
		if i == len(commits)-1 {
			break
		}
		if cur, err = self.commitTreeInStore(store, tree, cur); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

// forkPoint returns the merge base of upstream and head, or "" when they share no
// history.
func (self *RebaseCommands) forkPoint(upstream, head string) (string, error) {
	output, err := self.selectedRebaseGitOutput(NewGitCmd("merge-base").Arg(upstream+"^{commit}", head+"^{commit}").ToArgv())
	if commandExitedWithOne(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find rebase fork point: %w", err)
	}
	return trimTerminalLineEnding(string(output)), nil
}

// parseNameStatus splits -z --name-status output into the paths it names, and the
// file renames and copies it lists. A status token is followed by one path, or by
// two for a rename or copy.
func parseNameStatus(raw []byte) (paths []string, renames []renamePair) {
	tokens := nulPaths(raw)
	for i := 0; i < len(tokens); {
		status := tokens[i]
		i++
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if i+1 < len(tokens) {
				paths = append(paths, tokens[i], tokens[i+1])
				renames = append(renames, renamePair{oldPath: tokens[i], newPath: tokens[i+1]})
			}
			i += 2
			continue
		}
		if i < len(tokens) {
			paths = append(paths, tokens[i])
		}
		i++
	}
	return paths, renames
}

// pathDirectory returns the directory of a slash-separated path, or "" for the root.
func pathDirectory(path string) string {
	if slash := strings.LastIndexByte(path, '/'); slash >= 0 {
		return path[:slash]
	}
	return ""
}

// lastComponent returns the final component of a slash-separated path.
func lastComponent(path string) string {
	return path[strings.LastIndexByte(path, '/')+1:]
}
