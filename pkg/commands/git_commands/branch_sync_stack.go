package git_commands

import (
	"fmt"
	"sort"
	"strings"
)

// BranchStackMember is a direct local branch whose tip is in the selected
// branch's ancestry and is not already reachable from any supplied main ref.
// It is inspection data only; callers must revalidate it before acting.
type BranchStackMember struct {
	Name string
	OID  string
}

// InspectBranchStack returns local branch tips below name in native Git walk
// order. The selected branch and exact main refs are excluded.
func (self *SyncCommands) InspectBranchStack(name, expectedOID string, mainRefs []string) ([]BranchStackMember, error) {
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(info.OID, expectedOID) {
		return nil, ErrBranchSyncChanged
	}
	branches := NewBranchCommands(self.GitCommon)
	mainOIDs := make([]string, 0, len(mainRefs))
	excluded := map[string]bool{"refs/heads/" + name: true}
	for _, ref := range mainRefs {
		if !strings.HasPrefix(ref, "refs/") {
			return nil, fmt.Errorf("main ref %q must be a full ref name", ref)
		}
		if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
			return nil, fmt.Errorf("invalid main ref %q: %w", ref, err)
		}
		state, err := branches.readPlacementRef(ref)
		if err != nil {
			return nil, err
		}
		if state.symbolic != "" {
			return nil, fmt.Errorf("main ref %q is symbolic", ref)
		}
		if !state.exists {
			return nil, fmt.Errorf("main ref %q does not exist", ref)
		}
		if err := branches.validateLiteralCommitOID(state.oid); err != nil {
			return nil, fmt.Errorf("main ref %q does not point to a commit: %w", ref, err)
		}
		mainOIDs = append(mainOIDs, state.oid)
		excluded[ref] = true
	}
	const format = "%(refname)%00%(objectname)%00%(symref)%00"
	stdout, _, err := self.cmd.New(NewGitCmd("for-each-ref").Arg("--format="+format, "refs/heads").ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return nil, err
	}
	headRows, err := parseBranchStackHeads(stdout)
	if err != nil {
		return nil, err
	}
	args := NewGitCmd("rev-list").Config("core.warnAmbiguousRefs=false").Arg("--topo-order", info.OID+"^{commit}")
	if len(mainOIDs) > 0 {
		args.Arg("--not")
		for _, oid := range mainOIDs {
			args.Arg(oid + "^{commit}")
		}
	}
	walk, _, err := self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return nil, err
	}
	order := make(map[string]int)
	for i, oid := range strings.Fields(walk) {
		if err := validateObjectID(oid); err != nil {
			return nil, fmt.Errorf("malformed rev-list output: %w", err)
		}
		if _, ok := order[strings.ToLower(oid)]; ok {
			return nil, fmt.Errorf("duplicate commit in rev-list output")
		}
		order[strings.ToLower(oid)] = i
	}
	byOID := make(map[string][]string)
	for _, row := range headRows {
		if row.symref != "" || excluded[row.ref] {
			continue
		}
		oid := strings.ToLower(row.oid)
		if _, ok := order[oid]; ok {
			byOID[oid] = append(byOID[oid], strings.TrimPrefix(row.ref, "refs/heads/"))
		}
	}
	var result []BranchStackMember
	for _, oid := range strings.Fields(walk) {
		oid = strings.ToLower(oid)
		names := byOID[oid]
		sort.Strings(names)
		for _, branch := range names {
			result = append(result, BranchStackMember{Name: branch, OID: oid})
		}
	}
	return result, nil
}

type branchStackHead struct{ ref, oid, symref string }

func parseBranchStackHeads(output string) ([]branchStackHead, error) {
	var rows []branchStackHead
	for offset := 0; offset < len(output); {
		fields := make([]string, 0, 3)
		for len(fields) < 3 {
			n := strings.IndexByte(output[offset:], '\x00')
			if n < 0 {
				return nil, fmt.Errorf("malformed local branch snapshot")
			}
			end := offset + n
			fields = append(fields, output[offset:end])
			offset = end + 1
		}
		if offset >= len(output) || output[offset] != '\n' {
			return nil, fmt.Errorf("malformed local branch snapshot terminator")
		}
		offset++
		if !strings.HasPrefix(fields[0], "refs/heads/") || fields[1] == "" {
			return nil, fmt.Errorf("malformed local branch snapshot record")
		}
		if err := validateObjectID(fields[1]); err != nil {
			return nil, err
		}
		rows = append(rows, branchStackHead{fields[0], strings.ToLower(fields[1]), fields[2]})
	}
	return rows, nil
}
