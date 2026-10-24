package git_commands

import (
	"fmt"
	"strings"
)

// CombineHistoryCommit is a commit captured from a Git first-parent history.
type CombineHistoryCommit struct {
	OID     string
	Parents []string
	Message string
}

// LinearCombinePlan contains a fully resolved plan for combining commits on a
// linear history suffix.
type LinearCombinePlan struct {
	keeper      string
	parent      string
	head        string
	selected    []CombineHistoryCommit
	history     []CombineHistoryCommit
	todo        string
	summary     string
	description string
}

func (plan LinearCombinePlan) KeeperOID() string   { return plan.keeper }
func (plan LinearCombinePlan) BaseOID() string     { return plan.parent }
func (plan LinearCombinePlan) HeadOID() string     { return plan.head }
func (plan LinearCombinePlan) Todo() string        { return plan.todo }
func (plan LinearCombinePlan) Summary() string     { return plan.summary }
func (plan LinearCombinePlan) Description() string { return plan.description }

func (plan LinearCombinePlan) SelectedOIDs() []string {
	oids := make([]string, len(plan.selected))
	for i, commit := range plan.selected {
		oids[i] = commit.OID
	}
	return oids
}

// PlanLinearCombine validates a captured ancestor-first first-parent chain and
// creates a deterministic interactive-rebase todo for the selected commits.
func PlanLinearCombine(history []CombineHistoryCommit, selected []string) (LinearCombinePlan, error) {
	if len(history) == 0 {
		return LinearCombinePlan{}, fmt.Errorf("history is empty")
	}
	format := 0
	positions := make(map[string]int, len(history))
	clonedHistory := cloneCombineHistory(history)
	for i := range clonedHistory {
		clonedHistory[i].OID = strings.ToLower(clonedHistory[i].OID)
		for j := range clonedHistory[i].Parents {
			clonedHistory[i].Parents[j] = strings.ToLower(clonedHistory[i].Parents[j])
		}
	}
	for i, commit := range clonedHistory {
		length := len(commit.OID)
		if !IsFullObjectID(commit.OID) {
			return LinearCombinePlan{}, fmt.Errorf("history commit %d has malformed object ID", i)
		}
		if format == 0 {
			format = length
		} else if length != format {
			return LinearCombinePlan{}, fmt.Errorf("history mixes object ID formats")
		}
		if _, exists := positions[commit.OID]; exists {
			return LinearCombinePlan{}, fmt.Errorf("history contains duplicate object ID %q", commit.OID)
		}
		positions[commit.OID] = i
		seenParents := make(map[string]struct{}, len(commit.Parents))
		for _, parent := range commit.Parents {
			if !IsFullObjectID(parent) || len(parent) != format {
				return LinearCombinePlan{}, fmt.Errorf("commit %q has malformed parent object ID", commit.OID)
			}
			if _, exists := seenParents[parent]; exists {
				return LinearCombinePlan{}, fmt.Errorf("commit %q has duplicate parent object ID", commit.OID)
			}
			seenParents[parent] = struct{}{}
		}
		if i > 0 && (len(commit.Parents) == 0 || commit.Parents[0] != clonedHistory[i-1].OID) {
			return LinearCombinePlan{}, fmt.Errorf("history is not in first-parent ancestor order at %q", commit.OID)
		}
	}
	if len(selected) < 2 {
		return LinearCombinePlan{}, fmt.Errorf("at least two commits must be selected")
	}
	selectedSet := make(map[string]struct{}, len(selected))
	for _, oid := range selected {
		oid = strings.ToLower(oid)
		if !IsFullObjectID(oid) || len(oid) != format {
			return LinearCombinePlan{}, fmt.Errorf("selected commit has malformed or mixed-format object ID")
		}
		if _, exists := selectedSet[oid]; exists {
			return LinearCombinePlan{}, fmt.Errorf("selected commit %q is duplicated", oid)
		}
		selectedSet[oid] = struct{}{}
		if _, exists := positions[oid]; !exists {
			return LinearCombinePlan{}, fmt.Errorf("selected commit %q is not in history", oid)
		}
	}
	keeperIndex := len(history)
	for oid := range selectedSet {
		if positions[oid] < keeperIndex {
			keeperIndex = positions[oid]
		}
	}
	for i := keeperIndex; i < len(clonedHistory); i++ {
		if len(clonedHistory[i].Parents) > 1 {
			return LinearCombinePlan{}, fmt.Errorf("cannot combine across merge commit %q", clonedHistory[i].OID)
		}
	}

	orderedSelected := make([]CombineHistoryCommit, 0, len(selectedSet))
	fixups := make([]string, 0, len(selectedSet)-1)
	picks := make([]string, 0, len(clonedHistory)-keeperIndex-1)
	for i := keeperIndex + 1; i < len(clonedHistory); i++ {
		commit := clonedHistory[i]
		if _, isSelected := selectedSet[commit.OID]; isSelected {
			orderedSelected = append(orderedSelected, commit)
			fixups = append(fixups, "fixup "+commit.OID)
		} else {
			picks = append(picks, "pick "+commit.OID)
		}
	}
	orderedSelected = append([]CombineHistoryCommit{clonedHistory[keeperIndex]}, orderedSelected...)
	messages := make([]string, len(orderedSelected))
	for i, commit := range orderedSelected {
		if strings.ContainsRune(commit.Message, '\x00') {
			return LinearCombinePlan{}, fmt.Errorf("commit %q message contains NUL", commit.OID)
		}
		messages[i] = commit.Message
	}
	summary := orderedSelected[0].Message
	if newline := strings.IndexByte(summary, '\n'); newline >= 0 {
		summary = summary[:newline]
	}
	parent := ""
	if len(clonedHistory[keeperIndex].Parents) > 0 {
		parent = clonedHistory[keeperIndex].Parents[0]
	}
	lines := append([]string{"reword " + clonedHistory[keeperIndex].OID}, fixups...)
	lines = append(lines, picks...)
	return LinearCombinePlan{
		keeper: clonedHistory[keeperIndex].OID, parent: parent,
		head:     clonedHistory[len(clonedHistory)-1].OID,
		selected: cloneCombineHistory(orderedSelected), history: clonedHistory,
		todo: strings.Join(lines, "\n") + "\n", summary: summary,
		description: strings.Join(messages, "\n\n"),
	}, nil
}

func cloneCombineHistory(history []CombineHistoryCommit) []CombineHistoryCommit {
	cloned := make([]CombineHistoryCommit, len(history))
	for i, commit := range history {
		cloned[i] = commit
		cloned[i].Parents = append([]string(nil), commit.Parents...)
	}
	return cloned
}
